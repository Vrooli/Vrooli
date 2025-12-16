package api

import "testing"

func TestLifecycleProtectionSkipsCliEntrypoints(t *testing.T) {
	input := []byte(`package main

import "fmt"

func main() { fmt.Println("hello") }
`)
	if v := CheckLifecycleProtection(input, "cli/main.go", "demo-app"); v != nil {
		t.Fatalf("expected cli/main.go to be skipped, got violation: %+v", *v)
	}
}

