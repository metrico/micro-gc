package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/metrico/micro-gc/context"
	"runtime"
)

type CtxBuffer struct {
	bytes.Buffer
	CtxID uint32
}

func (b *CtxBuffer) Write(p []byte) (int, error) {
	ctxId := context.GetContextID()
	context.SetContext(b.CtxID)
	n, err := b.Buffer.Write(p)
	context.SetContext(ctxId)
	return n, err
}

func main() {
	for i := 0; i < 10; i++ {
		context.SetContext(1)
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
		context.SetContext(0)
		buf := CtxBuffer{
			Buffer: bytes.Buffer{},
			CtxID:  1,
		}
		enc := json.NewEncoder(&buf)
		err := enc.Encode(hugeObject)
		if err != nil {
			panic(err)
		}
		fmt.Println("generated a JSON object of size ", buf.Len()/1024/1024, "MiB")
		context.ReleaseContext(1)
	}
	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	fmt.Println("total heap used after the operation:", ms.HeapSys/1024/1024, "MiB")
	fmt.Println("total heap released during the operation:", ms.HeapReleased/1024/1024, "MiB")
}
