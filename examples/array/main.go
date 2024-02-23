package main

import (
	"fmt"
	"github.com/metrico/micro-gc/stack"
	"runtime"
)

const arraySize = 15 * 1024 * 1024 // 30MB
var staticArrToAppend [arraySize]byte

func main() {

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Println("total memory used before the operation:", m.Sys/1024/1024, "MiB")
	fmt.Println("total memory for GC metadata:", m.GCSys, "bytes")
	for i := 0; i < 5; i++ {
		fmt.Println("Stack Push")
		stack.PushStack()
		arr := make([]byte, arraySize)
		arr[0] = byte(i)
		arr = append(arr, staticArrToAppend[:]...)
		fmt.Println("Allocating array of cap ", cap(arr)/1024/1024, "MiB")
		_ = arr
		fmt.Println("Stack Pop")
		stack.PopStack()
		runtime.ReadMemStats(&m)
		fmt.Println("total memory used after one iteration:", m.Sys/1024/1024, "MiB")
	}
	fmt.Println("total memory used after the operation:", m.Sys/1024/1024, "MiB")
	fmt.Println("total heap used after the operation:", m.HeapSys/1024/1024, "MiB")
}
