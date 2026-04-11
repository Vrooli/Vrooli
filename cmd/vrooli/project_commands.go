package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/config"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

var (
	runProjectSetupFn   = projectsetup.RunSetup
	runProjectDevelopFn = projectsetup.RunDevelop
)

func runProjectSetupCommand(root string, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	return runProjectSetupFn(root, home, args, stdout, stderr)
}

func runProjectDevelopCommand(root string, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	return runProjectDevelopFn(root, home, args, stdout, stderr)
}
