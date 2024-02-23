package microgc

import (
	"fmt"
	"runtime"
)

func init() {

	const arraySize = 30 * 1024 * 1024   // 30MB
	const maxHeapSize = 60 * 1024 * 1024 // 60MB
	var m runtime.MemStats
	ReadMemStats(&m)
	fmt.Println("heap size before push and pop operation:", m.HeapInuse)
	for i := 0; i < 5; i++ {

		Stack_push()
		arr := make([]byte, arraySize)
		_ = arr
		ReadMemStats(&m)
		fmt.Println("heap size after push:", m.HeapInuse)
		Stack_pop()
		ReadMemStats(&m)
		fmt.Println("heap size after pop: ", m.HeapInuse)
		if m.HeapInuse > maxHeapSize {
			fmt.Println("Heap size exceeds limit:", m.HeapInuse)
		}

	}

}
