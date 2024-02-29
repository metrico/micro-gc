//go:build gc.custom

package context

import "unsafe"

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
