package main

import "github.com/vrooli/vrooli/internal/shell"

func runExternalCommand(spec commandSpec) error {
	cmd := shell.CommandWithDefaults(shell.Spec{
		Name: spec.name,
		Args: spec.args,
		Dir:  spec.dir,
		Env:  spec.env,
	})
	return cmd.Run()
}
