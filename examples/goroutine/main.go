package main

import (
	"fmt"
	"github.com/metrico/micro-gc/context"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

const maxHeapSize = 32 * 1024 * 1024 // 32 MiB
func main() {
	fmt.Println("start")
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	wg.Add(2)

	go startGoroutine(&wg)
	startGoroutine(&wg)

	wg.Wait()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Println("total memory used after the operation: ", m.Sys/1024/1024, "MiB")
}

func startGoroutine(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < 100; i++ {
		newCtxID := rand.Uint32()
		context.SetContext(newCtxID)
		arr := make([]byte, 10*1024*1024)
		_ = arr
		time.Sleep(100 * time.Millisecond)
		context.SetContext(0)
		context.ReleaseContext(newCtxID)
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
	}
}
