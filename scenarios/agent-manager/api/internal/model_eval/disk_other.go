//go:build !linux

package model_eval

import "errors"

func readFilesystemBytes(string) (int64, error) { return 0, errors.New("filesystem gate is unsupported on this platform") }
