package shell

import (
	"io"
	"os"
	"os/exec"
)

type Spec struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Command(spec Spec) *exec.Cmd {
	cmd := exec.Command(spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	return cmd
}

func CommandWithDefaults(spec Spec) *exec.Cmd {
	cmd := Command(spec)
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	return cmd
}

func BashScript(path string, args []string, spec Spec) *exec.Cmd {
	spec.Name = "bash"
	spec.Args = append([]string{path}, args...)
	return CommandWithDefaults(spec)
}

func BashCommand(command string, spec Spec) *exec.Cmd {
	spec.Name = "bash"
	spec.Args = []string{"-lc", command}
	return CommandWithDefaults(spec)
}
