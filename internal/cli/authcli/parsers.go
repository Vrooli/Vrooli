package authcli

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

// StatusRequest is the parsed CLI surface for `vrooli auth status`.
type StatusRequest struct {
	CheckExpiry bool
}

func ParseStatusRequest(args []string) (StatusRequest, error) {
	spec := commandSpec(CommandStatus)
	parsed, err := commandtree.ParseArgs("auth status", commandHelpText(CommandStatus), spec.Args, args)
	if err != nil {
		return StatusRequest{}, err
	}
	return StatusRequest{CheckExpiry: parsed.HasFlag("--check-expiry")}, nil
}

func RenderCommandHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandtree.RenderHelpText(commandtree.Help{
		Title:        "vrooli auth - Report sign-in state for host tools",
		Usage:        "vrooli auth <subcommand> [options]",
		DefaultGroup: "Authentication",
	}, CommandSpecs()))
}

func RenderStatusHelp(w io.Writer) {
	commandtree.WriteHelp(w, commandHelpText(CommandStatus))
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic(fmt.Sprintf("unknown auth command spec: %s", id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli auth "+spec.Name, spec)
}
