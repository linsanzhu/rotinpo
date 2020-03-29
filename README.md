## rotinpo
    
**an easy-to-use and efficient go routine pool**

> ## Quick Start

```go
package main

import (
    "fmt"

    "github.com/folivora-ice/rotinpo"
)

func main() {
    pool := rotinpo.New(
        WithMaxWorkingSize(50),
        WithMaxIdleSize(10),
        WithMaxLifeTime(10 * time.Second),
        WithMaxWaitingSize(20),
    )
    err := pool.Execute(rotinpo.Task(func() {
        fmt.Println("hello rotinpo")
    }))
    if rotinpo.IsRejected(err) {
        fmt.Println("the task is reject")
    }

    pool.WaitExecute(rotinpo.Task(func() {
        fmt.Println("wait until could be execute or cached")
    }))

    pool.Close()
}
```

