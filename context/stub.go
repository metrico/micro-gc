//go:build !gc.custom

package stack

func SetContext(ctxID uint32) {
}

func ReleaseContext(ctxID uint32) {
}

func GetContextID() uint32 {

}
