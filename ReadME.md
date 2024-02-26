# MicroGC
> small and unhandy _(yet working)_ GC for tiny-go + wasm

## Leaking + Stack GC for tiny go + wasm

It's like the leaking GC, but with the stack management elements inside:

```golang
package main
import (
	"github.com/metrico/micro-gc/stack"
)

func main() {

	for i := 0; i < 5; i++ {
		stack.PushStack() // push current heap pointer to the stack 
		arr := make([]byte, 100 * 1024 * 1024) //allocate as much as you want
		arr[0] = byte(i)
		stack.PopStack() // release all the memory allocated since the last PushStack
	}
}
```

## Prerequisites

- tiny-go v0.30+

## How to use

1. Import the library `"github.com/metrico/micro-gc/stack"` into the go code
2. Add `stack.PushStack` and `stack.PopStack` where you want
3. Build the wasm application with the `-gc=custom` flag 

## Known limitations

1. Be very careful with `sync.Pool` and all the global variables. `PopStack` may corrupt the data.
2. Be very careful with all the libraries using `sync.Pool` as well. Apparently, `fmt.Println` uses it.
3. Be careful working with goroutines.
4. Definitely not for multicore environments.

## Examples

Feel free to get inspiration from the examples in the `examples` directory.

Build an example like this: 
- `tinygo build -gc=custom -target=wasm -o=test.wasm github.com/metrico/micro-gc/examples/array/`

Then run it using nodeJS and wasm_exec file: `node wasm_exec.js test.wasm`
