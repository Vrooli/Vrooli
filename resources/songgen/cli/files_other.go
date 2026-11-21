//go:build !linux

package main

import (
	"errors"
	"os"
)

func publishSongDirectory(string, string) error {
	return errors.New("Songgen native export requires Linux")
}
func openSongRegular(string) (*os.File, error) {
	return nil, errors.New("Songgen native export requires Linux")
}
