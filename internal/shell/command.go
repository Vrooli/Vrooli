package shell

import (
	"context"
	"io"
	"os"
	"os/exec"
)

type Spec struct {
	Context context.Context
	Name    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func Command(spec Spec) *exec.Cmd {
	var cmd *exec.Cmd
	if spec.Context != nil {
		cmd = exec.CommandContext(spec.Context, spec.Name, spec.Args...)
	} else {
		cmd = exec.Command(spec.Name, spec.Args...)
	}
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

func BashCommand(command string, spec Spec) *exec.Cmd {
	spec.Name = "bash"
	spec.Args = []string{"-lc", command}
	return CommandWithDefaults(spec)
}

func Run(spec Spec) error {
	return CommandWithDefaults(spec).Run()
}

func Output(spec Spec) ([]byte, error) {
	return Command(spec).Output()
}

func CombinedOutput(spec Spec) ([]byte, error) {
	return Command(spec).CombinedOutput()
}

func LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
