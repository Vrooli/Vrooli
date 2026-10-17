package cliapp

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/vrooli/envkit-go"
)

// PassthroughSpec describes an external command invocation owned by the
// passthrough primitive.
type PassthroughSpec struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
}

// Passthrough builds a renderer-separated external-delegation handler. The
// operation callback chooses the external command from parsed inputs, while
// cli-core owns the process wiring and exit-code mapping. Pair it with a command
// declaring the passthrough exception.
func Passthrough(resolve func(ctx OperationContext) (PassthroughSpec, error)) PrimitiveHandler {
	return PrimitiveHandler{primitive: PrimitivePassthrough, Run: func(ctx RunContext) error {
		spec, err := resolve(ctx)
		if err != nil {
			return err
		}
		return RunPassthrough(spec, ctx.Stdout(), ctx.Stderr())
	}}
}

// RunPassthrough streams an external command to the provided writers and returns
// the command's native error/exit status.
func RunPassthrough(spec PassthroughSpec, stdout, stderr io.Writer) error {
	name := strings.TrimSpace(spec.Command)
	if name == "" {
		return fmt.Errorf("passthrough command is empty")
	}
	cmd := exec.Command(name, spec.Args...)
	cmd.Dir = strings.TrimSpace(spec.Dir)
	if len(spec.Env) > 0 {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(spec.Env))
	}
	cmd.Stdin = os.Stdin
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// PassthroughLegacy tags an argv-parsing passthrough command with passthrough
// evidence. Use it when the command parses argv itself or delegates directly to
// another executable outside the ArgSchema/RunContext path.
func PassthroughLegacy(run func(args []string) error) LegacyPrimitiveHandler {
	return LegacyPrimitiveHandler{primitive: PrimitivePassthrough, Run: run}
}
