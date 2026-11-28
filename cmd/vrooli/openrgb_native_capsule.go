package main

import (
	"context"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/resources"
)

func runOpenRGBNativeCapsule(args []string, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "__vrooli-openrgb-native-capsule" {
		return false, 0
	}
	if len(args) != 2 {
		fmt.Fprintln(stderr, "native lighting capsule requires exactly one admitted capsule path")
		return true, 2
	}
	if err := resources.RunOpenRGBNativeCapsule(context.Background(), args[1]); err != nil {
		fmt.Fprintln(stderr, err)
		return true, 1
	}
	return true, 0
}
