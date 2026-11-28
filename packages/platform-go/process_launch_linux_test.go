//go:build linux

package platform

import (
	"os"
	"reflect"
	"testing"
)

func TestProcessLaunchObservationReadsExactSelfArgvAndRunningImage(t *testing.T) {
	args, err := ProcessArguments(os.Getpid())
	if err != nil || !reflect.DeepEqual(args, os.Args) {
		t.Fatalf("native process argument boundaries differ: %v", err)
	}
	image, err := OpenProcessExecutable(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	info, err := image.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		t.Fatalf("running executable is not a retained readable file: %v", err)
	}
	for _, pid := range []int{-1, 0} {
		if _, err := ProcessArguments(pid); err == nil {
			t.Fatal("invalid process arguments inferred")
		}
		if file, err := OpenProcessExecutable(pid); err == nil {
			file.Close()
			t.Fatal("invalid process image inferred")
		}
	}
}
