package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/config"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

func runProjectSetupCommand(root string, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	return projectsetup.RunSetup(root, home, args, stdout, stderr)
}

func runProjectDevelopCommand(root string, args []string, stdout, stderr io.Writer) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	return projectsetup.RunDevelop(root, home, args, stdout, stderr)
}
