package main

import "github.com/vrooli/vrooli/internal/shell"

func runExternalCommand(spec commandSpec) error {
	return shell.Run(shell.Spec{
		Name: spec.name,
		Args: spec.args,
		Dir:  spec.dir,
		Env:  spec.env,
	})
}
