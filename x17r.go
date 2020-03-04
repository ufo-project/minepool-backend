package main

// #include <stdio.h>
// #include <stdlib.h>
// #include "./x17r/x17r.h"
// #cgo CFLAGS: -std=gnu99 -Wall -I.
// #cgo LDFLAGS: -L./x17r/ -lx17r
import "C"

import (
	"encoding/hex"
	"unsafe"
)

func X17r_Sum256(input string) []byte {
	in, err := hex.DecodeString(input)
	if err != nil {
		Warning.Println("X17r_Sum256 DecodeString error:", err)
		return nil
	}
	in1 := (*C.char)(unsafe.Pointer(&in[0]))

	output := make([]byte, 32)
	out := (*C.char)(unsafe.Pointer(&output[0]))

	C.x17r_hash(unsafe.Pointer(out), unsafe.Pointer(in1), C.int(len(input)/2))

	return output
}
