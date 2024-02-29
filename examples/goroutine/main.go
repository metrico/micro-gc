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
	rand.Seed(time.Now().UnixNano())

	var wg sync.WaitGroup
	wg.Add(2)

	mutex := &sync.Mutex{} // Create a mutex for synchronization

	go startGoroutine(&wg, mutex)
	go startGoroutine(&wg, mutex)

	wg.Wait()
}

func startGoroutine(wg *sync.WaitGroup, mutex *sync.Mutex) {
	defer wg.Done()

	var m runtime.MemStats

	for i := 0; i < 100; i++ {
		prevCtxID := context.GetContextID()
		newCtxID := rand.Uint32()
		context.SetContext(newCtxID)
		arr := make([]int, 10*1024*1024)
		fmt.Println("Allocating array with capacity", cap(arr)/1024/1024, "MiB")
		time.Sleep(100 * time.Millisecond)
		context.SetContext(prevCtxID)
		context.ReleaseContext(newCtxID)
		context.ReadMemStats(&m)
		fmt.Println("heap in use", m.HeapInuse)
		if m.HeapInuse > maxHeapSize {
			fmt.Println("Heap size exceeded 32 MB")
		}
		fmt.Println("Total memory used after one iteration:", m.Sys/1024/1024, "MiB")
	}

	runtime.ReadMemStats(&m)
	fmt.Println("Total memory used after the operation:", m.Sys/1024/1024, "MiB")
	fmt.Println("Total heap used after the operation:", m.HeapSys/1024/1024, "MiB")
}
