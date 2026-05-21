package main

import (
	"fmt"
	"react-component-library/internal/uimanifest"
)

func main() {
	l := uimanifest.NewFSLoader("/home/matthalloran8/Vrooli")
	mf, err := l.Load("ui-health")
	fmt.Println("err:", err)
	fmt.Println("slots:", len(mf.Slots))
	for k, v := range mf.Slots {
		fmt.Println(" ", k, "dir:", v.Dir)
	}
}
