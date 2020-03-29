// Copyright 2020-03-29 folivora-ice
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rotinpo

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrorRejected = errors.New("reject task: pool is full")
)

func IsRejected(err error) bool {
	return err != nil && err == ErrorRejected
}

type RoutinePool struct {
	// the maximum number of idle go routines while no tasks can be executed
	maxIdleSize int32

	// the maximum number of waiting tasks
	maxWaitingSize int32

	// the maximum number of concurrently running go routines
	maxWorkingSize int32

	// The maximum idle time of the go routine.
	// When maxLifeTime is exceeded and no tasks are available,
	// the go routine will exit.
	maxLifeTime time.Duration

	// when the number of go routines exceeds maxIdleSize, the task
	// will be cached in waitingQueue if possible, otherwise if number
	// of working go routines does not exceed maxWoringSize, a new go
	// routine will be created
	waitingQueue chan Task

	// the number of usable go routine
	working int32

	//
	ctx   context.Context
	close context.CancelFunc
}

type Option interface {
	apply(*RoutinePool)
}

type optionFunc func(*RoutinePool)

func (f optionFunc) apply(pool *RoutinePool) {
	f(pool)
}

// specify the pool's capacity, if cap is non-positive,
func WithMaxWorkingSize(maxWorkingSize int32) Option {
	return optionFunc(func(pool *RoutinePool) {
		pool.maxWorkingSize = maxWorkingSize
	})

}

func WithMaxIdleSize(maxIdleSize int32) Option {
	return optionFunc(func(pool *RoutinePool) {
		pool.maxIdleSize = maxIdleSize
	})
}

func WithMaxLifeTime(duration time.Duration) Option {
	return optionFunc(func(pool *RoutinePool) {
		pool.maxLifeTime = duration
	})
}

func WithMaxWaitingSize(maxWaitingSize int32) Option {
	return optionFunc(func(pool *RoutinePool) {
		pool.maxWaitingSize = maxWaitingSize
	})
}

// New return an initialized routine pool
func New(opts ...Option) *RoutinePool {
	var pool = RoutinePool{}
	for _, opt := range opts {
		opt.apply(&pool)
	}
	pool.waitingQueue = make(chan Task, pool.maxWaitingSize)
	pool.ctx, pool.close = context.WithCancel(context.Background())
	return &pool
}

// execute a task or cache it if there are some space in waitingQueue,
// if the number of working go routine exceeds maxWorkingSize and no space
// in waitingQueue, a ErrorRejected error will be returned
func (pool *RoutinePool) Execute(task Task) error {
	w := atomic.AddInt32(&pool.working, 1)
	if w > pool.maxIdleSize {
		select {
		case pool.waitingQueue <- task:
			atomic.AddInt32(&pool.working, -1)
			return nil
		default:
			if w > pool.maxWorkingSize {
				atomic.AddInt32(&pool.working, -1)
				return ErrorRejected
			}
		}
	}

	worker, err := pool.createWorker()
	if nil != err {
		return err
	}
	return worker.work(pool.ctx, task)
}

// try to execute a task or cache it, if it fails, wait util the
// waitingQueue has space
func (pool *RoutinePool) WaitExecute(task Task) {
	if err := pool.Execute(task); nil != err {
		pool.waitingQueue <- task
	}
}

// close the pool and stop all working go routine
func (pool *RoutinePool) Close() {
	if nil != pool.close {
		pool.close()
	}
}

func (pool *RoutinePool) createWorker() (*worker, error) {
	return &worker{pool: pool}, nil
}

type Task func()

func (t Task) fire() {
	t()
}

type worker struct {
	pool *RoutinePool

	closed chan bool
}

func (w worker) work(ctx context.Context, tasks ...Task) error {
	if nil == w.pool {
		return errors.New("start work: no pool be bound")
	}

	go func() {
		defer w.close()
		for _, task := range tasks {
			task.fire()
		}

		for {
			select {
			case task := <-w.pool.waitingQueue:
				task.fire()
			case <-time.After(w.pool.maxLifeTime):
				if atomic.AddInt32(&w.pool.working, -1) >= w.pool.maxIdleSize {
					return
				}
				if atomic.AddInt32(&w.pool.working, 1) > w.pool.maxIdleSize {
					atomic.AddInt32(&w.pool.working, -1)
					return
				}
			case <-ctx.Done():
				atomic.AddInt32(&w.pool.working, -1)
				return
			case <-w.closed:
				atomic.AddInt32(&w.pool.working, -1)
				return
			}
		}
	}()
	return nil
}

func (w worker) close() {
	defer func() {
		recover()
	}()
	close(w.closed)
}
