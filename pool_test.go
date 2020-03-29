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
	"sync/atomic"
	"testing"
	"time"
)

func TestRoutinePool(t *testing.T) {
	pool := New(WithMaxWorkingSize(50),
		WithMaxIdleSize(10),
		WithMaxLifeTime(10*time.Second),
		WithMaxWaitingSize(20))
	defer pool.Close()
	var count int32
	for i := 0; i < 80; i++ {
		err := pool.Execute(func() {
			time.Sleep(time.Duration(10) * time.Second)
			atomic.AddInt32(&count, 1)
		})
		if i >= 70 && !IsRejected(err) {
			t.Errorf("expect rejected, but not")
		} else if nil != err {
			t.Log(err)
		}
	}
	time.Sleep(9 * time.Second)
	c := atomic.LoadInt32(&count)
	if c != 0 {
		t.Errorf("expect 0, got %d", c)
	}
	time.Sleep(2 * time.Second)
	c = atomic.LoadInt32(&count)
	if c != 50 {
		t.Errorf("expect 50, got %d", c)
	}
	time.Sleep(8 * time.Second)
	c = atomic.LoadInt32(&count)
	if c != 50 {
		t.Errorf("expect 50, got %d", c)
	}
	time.Sleep(1 * time.Second)
	c = atomic.LoadInt32(&count)
	if c != 70 {
		t.Errorf("expect 70, got %d", c)
	}
}
