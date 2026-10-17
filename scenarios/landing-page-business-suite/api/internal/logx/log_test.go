package logx

import "testing"

func TestSystemImplementsLogger(t *testing.T) {
	if _, ok := any(System{}).(Logger); !ok {
		t.Fatal("System must implement Logger")
	}
}
