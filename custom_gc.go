//go:build gc.custom

package microgc

import (
	"fmt"
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
	//allocatedBlocks = make(map[uintptr]uintptr)
)

//export llvm.wasm.memory.size.i32
func wasm_memory_size(index int32) int32

//export llvm.wasm.memory.grow.i32
func wasm_memory_grow(index int32, delta int32) int32

//go:linkname align runtime.align
func align(p uintptr) uintptr

//go:linkname memzero runtime.memzero
func memzero(ptr unsafe.Pointer, size uintptr)

// like llvm.memcpy.p0.p0.i32(dst, src, size, false).
func memcpy(dst, src unsafe.Pointer, size uintptr)

// Map to store allocated memory blocks and their sizes
var allocatedBlocks = make(map[uintptr]uintptr)

// Stack_push pushes a new context onto the stack
func Stack_push() {
	var c Context
	c.start = heapptr
	contextIndex++
	context[contextIndex] = c
}

// Stack_pop pops a context from the stack
func Stack_pop() {
	if contextIndex >= 0 {
		c := context[contextIndex]
		heapptr = c.start
		contextIndex--

	} else {
		fmt.Println("Error: Stack underflow - cannot pop from an empty stack")
	}
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
	// according to POSIX everything beyond the previous pointer's
	// size will have indeterminate values so we can just copy garbage
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
	m.HeapIdle = 0 // These values are dummy values. Replace them with actual values if available.
	m.HeapInuse = uint64(heapptr - heapStart)
	m.HeapReleased = 0
	m.HeapSys = m.HeapInuse
	m.GCSys = 0
	m.TotalAlloc = gcTotalAlloc
	m.Mallocs = gcMallocs
	m.Frees = gcFrees
	m.Sys = uint64(heapEnd - heapStart)
}

//go:linkname initHeap runtime.initHeap
func initHeap() {
	wasm_memory_grow(wasmMemoryIndex, wasmPageSize)
}

//go:linkname markRoots runtime.markRoots
func markRoots(start, end uintptr) {
	panic("not implemented #7")
}

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
