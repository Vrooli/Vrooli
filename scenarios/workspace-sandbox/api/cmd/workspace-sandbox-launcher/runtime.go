package main

import "runtime"

func runtimeGOOS() string {
	return runtime.GOOS
}
