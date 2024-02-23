package main

import (
	"encoding/json"
	"fmt"
	"github.com/metrico/micro-gc/stack"
	"runtime"
)

//Encode a huge JSON object

func main() {
	json.Marshal(
		map[string]any{"smallStr": "", "hugeArr": []string{("")}},
	) //Marshal an example to init all the sync.pools encoding/json uses inside
	for i := 0; i < 10; i++ {
		fmt.Println("PushStack")
		stack.PushStack()
		hugeArr := make([]byte, 10*1024*1024)
		for i := range hugeArr {
			hugeArr[i] = byte('0')
		}
		hugeObject := map[string]any{
			"smallStr": "123456",
			"hugeArr": []string{
				string(hugeArr),
				string(hugeArr),
				string(hugeArr),
			},
		}
		buf, err := json.Marshal(hugeObject)
		if err != nil {
			panic(err)
		}
		fmt.Println("generated a JSON object of size ", len(buf)/1024/1024, "MiB")
		fmt.Println("PopStack")
		stack.PopStack()
	}
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	fmt.Println("total heap used after the operation:", ms.HeapSys/1024/1024, "MiB")
	fmt.Println("total heap released during the operation:", ms.HeapReleased/1024/1024, "MiB")
}
