//go:build !linux

package main

import "errors"

func publishRendererDirectory(string, string) error {
	return errors.New("managed renderer release staging requires Linux no-replace directory publication")
}
