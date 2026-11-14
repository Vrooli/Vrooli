//go:build !linux

package main

import (
	"errors"
	"os"
)

func publishDirectory(string, string) error { return errors.New("Stable Audio3 pilot requires Linux") }
func openRegular(string) (*os.File, error) {
	return nil, errors.New("Stable Audio3 pilot requires Linux")
}
