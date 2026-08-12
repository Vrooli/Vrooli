package main

import (
	"fmt"
	"os"
	"os/exec"
)

// resource-android-sdk is intentionally small: lifecycle authority remains in
// the Vrooli resource driver while this binary provides a governed, inspectable
// host-tool preflight for operators and CI.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("resource-android-sdk 1")
		return
	}
	path, err := exec.LookPath("adb")
	if err != nil {
		fmt.Fprintln(os.Stderr, "adb unavailable: install/start the android-sdk resource")
		os.Exit(1)
	}
	fmt.Println(path)
}
