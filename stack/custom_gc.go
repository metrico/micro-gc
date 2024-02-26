//go:build gc.custom

package stack

import (
	"runtime"
	"unsafe"
)

const (
	wasmMemoryIndex = 0
	wasmPageSize    = 64 * 1024
)

// Context represents the context of the stack
type Context struct {
	start uintptr
}

//go:extern __heap_base
var heapStartSymbol [0]byte
var (
	context      [1024]Context
	contextIndex int
	heapStart    = uintptr(unsafe.Pointer(&heapStartSymbol))
	heapEnd      = uintptr(wasm_memory_size(wasmMemoryIndex) * wasmPageSize)
	heapptr      = heapStart
	gcMallocs    uint64
	gcTotalAlloc uint64
	gcFrees      uint64
	heapReleased uint64
	//allocatedBlocks = make(map[uintptr]uintptr)
)

// PushStack pushes a new context onto the stack
func PushStack() {
	var c Context
	c.start = heapptr
	contextIndex++
	context[contextIndex] = c
}

// PopStack pops a context from the stack
func PopStack() {
	if contextIndex > 0 {
		c := context[contextIndex]
		_heapptr := c.start
		contextIndex--
		heapReleased += uint64(heapptr - _heapptr)
		heapptr = _heapptr
	} else {
		_heapptr := heapStart
		heapReleased += uint64(heapptr - _heapptr)
		heapptr = _heapptr
	}
	gcFrees++
}

//go:linkname alloc runtime.alloc
func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	size = align(size)
	addr := heapptr
	gcTotalAlloc += uint64(size)
	gcMallocs++
	heapptr += size
	for heapptr >= heapEnd {
		if !growHeap() {
			return nil
		}
	}
	pointer := unsafe.Pointer(addr)
	memzero(pointer, size)
	return pointer
}

//go:linkname realloc runtime.realloc
func realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	newAlloc := alloc(size, nil)
	if ptr == nil {
		return newAlloc
	}
	memcpy(newAlloc, ptr, size)
	return newAlloc
}

//go:linkname free runtime.free
func free(ptr unsafe.Pointer, size uintptr) {

}

//go:linkname GC runtime.GC
func GC() {
}

//go:linkname SetFinalizer runtime.SetFinalizer
func SetFinalizer(obj interface{}, finalizer interface{}) {
}

//go:linkname ReadMemStats runtime.ReadMemStats
func ReadMemStats(m *runtime.MemStats) {
	if m == nil {
		return
	}
	m.HeapIdle = uint64(heapEnd - heapptr)
	m.HeapInuse = uint64(heapptr - heapStart)
	m.HeapReleased = heapReleased
	m.HeapSys = uint64(heapEnd - heapStart)
	m.GCSys = uint64(unsafe.Sizeof(context)) + 96 //[1024]Context + all the vars on the top of the file
	m.TotalAlloc = gcTotalAlloc
	m.Mallocs = gcMallocs
	m.Frees = gcFrees
	m.Sys = uint64(wasm_memory_size(wasmMemoryIndex) * wasmPageSize)
}

//go:linkname initHeap runtime.initHeap
func initHeap() {
}

//go:linkname markRoots runtime.markRoots
func markRoots(start, end uintptr) {}

// setHeapEnd sets a new (larger) heapEnd pointer.
func setHeapEnd(newHeapEnd uintptr) {
	heapEnd = newHeapEnd
}

func growHeap() bool {
	memorySize := wasm_memory_size(wasmMemoryIndex)
	result := wasm_memory_grow(wasmMemoryIndex, memorySize)
	if result == -1 {
		return false
	}
	setHeapEnd(uintptr(wasm_memory_size(wasmMemoryIndex) * wasmPageSize))
	return true
}
