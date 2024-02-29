//go:build gc.custom

package context

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

const (
	wasmMemoryIndex      = 0
	wasmPageSize         = 64 * 1024
	maxMemoryAllocations = 1024000
	maxHeapSize          = 32 * 1024 * 1024
)

type Context struct {
	ID      uint32
	Start   uintptr
	End     uintptr
	HeapPtr uintptr
}

//go:extern __heap_base
var heapStartSymbol [0]byte
var (
	memoryAllocRegistry    [maxMemoryAllocations]Context
	memoryAllocRegistryLen uint32
	contextMutex           sync.Mutex
	currentContextID       uint32
	heapStart              = uintptr(unsafe.Pointer(&heapStartSymbol))
	heapEnd                = uintptr(wasm_memory_size(wasmMemoryIndex) * wasmPageSize)
	heapptr                = heapStart
	gcMallocs              uint64
	gcTotalAlloc           uint64
	gcFrees                uint64
	heapReleased           uint64
)

// SetContext sets the current context ID.
func SetContext(ctxID uint32) {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	currentContextID = ctxID

	fmt.Println("Set context value", ctxID)
}

// ReleaseContext releases the memory associated with the given context ID.
func ReleaseContext(ctxID uint32) {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	var newMemoryAllocRegistry [maxMemoryAllocations]Context
	var newMemoryAllocRegistryLen uint32

	fmt.Println("release context value", ctxID)
	for i := uint32(0); i < memoryAllocRegistryLen; i++ {
		if memoryAllocRegistry[i].ID != ctxID {
			newMemoryAllocRegistry[newMemoryAllocRegistryLen] = memoryAllocRegistry[i]
			newMemoryAllocRegistryLen++
		}
	}
	if newMemoryAllocRegistryLen < memoryAllocRegistryLen {
		copy(memoryAllocRegistry[:newMemoryAllocRegistryLen], newMemoryAllocRegistry[:newMemoryAllocRegistryLen])
		memoryAllocRegistryLen = newMemoryAllocRegistryLen
	}

}

// GetContextID returns the current context ID.
func GetContextID() uint32 {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	return currentContextID

}

//go:linkname alloc runtime.alloc
func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	size = align(size)
	// Check if there's enough space available in the heap
	if heapptr+size > heapEnd {
		if !growHeap() {
			fmt.Println("Failed to allocate memory: heap size exceeded")
			return nil
		}
	}
	// Initialize start, end, and heapptr if it's the first allocation
	if memoryAllocRegistryLen == 0 {
		start := heapStart
		end := heapStart + size

		// Update memory allocation registry with the new block
		memoryAllocRegistry[0] = Context{
			ID:      currentContextID,
			Start:   start,
			End:     end,
			HeapPtr: start + size,
		}
		memoryAllocRegistryLen++

		// Update heap pointer and statistics
		heapptr = end
		gcTotalAlloc += uint64(size)
		gcMallocs++
		memzero(unsafe.Pointer(start), size)
		// Return pointer to allocated memory
		return unsafe.Pointer(start)
	}

	// Search for free space between contexts
	for i := 0; i < int(memoryAllocRegistryLen)-1; i++ {
		if memoryAllocRegistry[i+1].Start-memoryAllocRegistry[i].End >= size {
			// Found free space between contexts, insert new entry
			start := memoryAllocRegistry[i].End
			end := start + size

			// Shift existing entries to accommodate the new block
			for j := int(memoryAllocRegistryLen); j > i+1; j-- {
				memoryAllocRegistry[j] = memoryAllocRegistry[j-1]
			}

			memoryAllocRegistry[i+1] = Context{
				ID:      currentContextID,
				Start:   start,
				End:     end,
				HeapPtr: end, // Move heap pointer to end of allocated block
			}
			memoryAllocRegistryLen++

			// Update heap pointer and statistics
			heapptr = end
			gcTotalAlloc += uint64(size)
			gcMallocs++
			memzero(unsafe.Pointer(start), size)
			// Return pointer to allocated memory
			return unsafe.Pointer(start)
		}
	}

	// If no free space between contexts is found, allocate at the end
	start := memoryAllocRegistry[memoryAllocRegistryLen-1].End
	end := start + size
	// Check if the allocation exceeds the heapEnd
	if end > heapEnd {
		if !growHeap() {
			fmt.Println("Failed to allocate memory: heap size exceeded")
			return nil
		}
	}

	memoryAllocRegistry[memoryAllocRegistryLen] = Context{
		ID:      currentContextID,
		Start:   start,
		End:     end,
		HeapPtr: end, // Move heap pointer to end of allocated block
	}
	memoryAllocRegistryLen++

	// Update heap pointer and statistics
	heapptr = end
	gcTotalAlloc += uint64(size)
	gcMallocs++
	memzero(unsafe.Pointer(start), size)
	pointer := unsafe.Pointer(start)
	return pointer
}

//go:linkname free runtime.free
func free(ptr unsafe.Pointer, size uintptr) {

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
	m.GCSys = uint64(unsafe.Sizeof(memoryAllocRegistryLen)) + 96 //[1024]Context + all the vars on the top of the file
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
