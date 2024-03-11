//go:build gc.custom

package context

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	wasmMemoryIndex      = 0
	wasmPageSize         = 64 * 1024
	maxMemoryAllocations = 1024 * 1024
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
	atomic.StoreUint32(&currentContextID, ctxID)
}

// ReleaseContext releases the memory associated with the given context ID.
func ReleaseContext(ctxID uint32) {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	var (
		i uint32 = 0
	)

	for j := range memoryAllocRegistry[:memoryAllocRegistryLen] {
		if memoryAllocRegistry[j].ID != ctxID {
			memoryAllocRegistry[i] = memoryAllocRegistry[j]
			i++
		} else {
			heapReleased += uint64(memoryAllocRegistry[j].End - memoryAllocRegistry[j].Start)
		}
	}
	memoryAllocRegistryLen = i
	if i > 0 {
		heapptr = memoryAllocRegistry[i-1].End
	} else {
		heapptr = heapStart
	}
}

// GetContextID returns the current context ID.
func GetContextID() uint32 {
	return atomic.LoadUint32(&currentContextID)
}

//go:linkname alloc runtime.alloc
func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	size = align(size)
	ctxId := atomic.LoadUint32(&currentContextID)

	_size := (size + wasmPageSize - 1) / wasmPageSize * wasmPageSize

	if idx := getContextByIdAndFreeSize(ctxId, size); idx != -1 {
		res := memoryAllocRegistry[idx].HeapPtr
		memoryAllocRegistry[idx].HeapPtr += size
		return unsafe.Pointer(res)
	} else if freeIdx := getFreeSpace(size); freeIdx != -1 {
		copy(memoryAllocRegistry[freeIdx+1:], memoryAllocRegistry[freeIdx:])
		start := heapStart
		if freeIdx > 0 {
			start = memoryAllocRegistry[freeIdx-1].End
		}
		memoryAllocRegistry[freeIdx] = Context{
			ID:      ctxId,
			Start:   start,
			End:     start + _size,
			HeapPtr: start + size,
		}
		memoryAllocRegistryLen++

		gcTotalAlloc += uint64(_size)
		gcMallocs++
		memzero(unsafe.Pointer(memoryAllocRegistry[freeIdx].Start), _size)

		return unsafe.Pointer(memoryAllocRegistry[freeIdx].Start)
	}
	for heapptr+size > heapEnd {
		if !growHeap() {
			fmt.Println("Failed to allocate memory: heap size exceeded")
			return nil
		}
	}
	memoryAllocRegistry[memoryAllocRegistryLen] = Context{
		ID:      ctxId,
		Start:   heapptr,
		End:     heapptr + _size,
		HeapPtr: heapptr + size,
	}
	memoryAllocRegistryLen++
	heapptr += _size

	gcTotalAlloc += uint64(_size)
	gcMallocs++
	memzero(unsafe.Pointer(memoryAllocRegistry[memoryAllocRegistryLen-1].Start), _size)
	return unsafe.Pointer(memoryAllocRegistry[memoryAllocRegistryLen-1].Start)
}

func getContextByIdAndFreeSize(ctxID uint32, size uintptr) int32 {
	for i := range memoryAllocRegistry[:memoryAllocRegistryLen] {
		if memoryAllocRegistry[i].ID == ctxID && memoryAllocRegistry[i].End-memoryAllocRegistry[i].HeapPtr >= size {
			return int32(i)
		}
	}
	return -1
}

func getFreeSpace(size uintptr) int32 {
	if memoryAllocRegistryLen == 0 {
		return -1
	}
	if memoryAllocRegistryLen == 1 && memoryAllocRegistry[0].Start-heapStart >= size {
		return 0
	}
	for i := range memoryAllocRegistry[:memoryAllocRegistryLen-1] {
		if memoryAllocRegistry[i+1].Start-memoryAllocRegistry[i].End >= size {
			return int32(i + 1)
		}
	}
	return -1
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
	atomic.StoreUint32(&currentContextID, 0)
	alloc(1, nil)
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
