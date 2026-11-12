//go:build !linux

package hostwireless

import (
	"context"
	"errors"
	"os"
)

func openBundleFile(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}

func runBuild(context.Context, string, string, string, []string) error {
	return errors.New("wireless module preparation requires Linux")
}
