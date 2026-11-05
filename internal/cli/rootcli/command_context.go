package rootcli

import (
	"context"
	"io"
	"os"
)

// CommandContext is the CLI-owned transport shell shared by root command
// handlers. Application services receive its values as explicit arguments and
// never depend on this type.
type CommandContext struct {
	Root          string
	Globals       GlobalOptions
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	Context       context.Context
	HomeDirFn     func() (string, error)
	ResolveRootFn func() (string, error)
	Version       string
}

func (ctx *CommandContext) OperationContext() context.Context {
	if ctx != nil && ctx.Context != nil {
		return ctx.Context
	}
	return context.Background()
}

func (ctx *CommandContext) Input() io.Reader {
	if ctx != nil && ctx.Stdin != nil {
		return ctx.Stdin
	}
	return os.Stdin
}

func (ctx *CommandContext) HomeDir() (string, error) {
	if ctx == nil || ctx.HomeDirFn == nil {
		return "", io.ErrClosedPipe
	}
	return ctx.HomeDirFn()
}

func (ctx *CommandContext) ResolveRoot() (string, error) {
	if ctx == nil || ctx.ResolveRootFn == nil {
		return "", io.ErrClosedPipe
	}
	return ctx.ResolveRootFn()
}
