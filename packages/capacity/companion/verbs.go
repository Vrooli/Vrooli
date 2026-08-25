package companion

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// One broker contract had four call signatures across the fleet: `capacity
// degrade --to`, `capacity-degrade --to`, `models activate --model`, and a bare
// `stop`. The broker had to know which resource it was talking to in order to
// talk to it at all, which is the opposite of a contract.
//
// This file is the one shape: every accelerated resource exposes
//
//	<resource-cli> capacity degrade  --to <label>
//	<resource-cli> capacity upshift  --to <label>
//	<resource-cli> capacity activity --state <active|idle>
//
// A resource supplies what each step means for it. Everything else — the flag
// names, the argument validation, the exit contract — comes from here.

// ErrUnknownStep is returned when the broker asks for a rung the resource does
// not declare. It is an error rather than a silent no-op: the broker believes
// the resource moved, and a resource that quietly ignored the request would
// leave the ledger describing a state that does not exist.
var ErrUnknownStep = errors.New("unknown capacity step")

// StepHandler applies one profile rung. The label is the step's declared name.
type StepHandler func(ctx context.Context, label string) error

// ActivityHandler reports the resource's own view of whether it is working.
// Most resources do not need one: idleness is the work-owner's truth, and a
// resource that cannot tell must not guess.
type ActivityHandler func(ctx context.Context, state string) error

// Verbs declares what a resource's capacity subcommands do.
type Verbs struct {
	// Resource is the resource name, used only in messages.
	Resource string
	// Degrade steps the resource down to a named rung. Required.
	Degrade StepHandler
	// Upshift steps the resource back up. nil means Degrade handles both
	// directions, which is true for any resource whose rungs are just "load
	// this model instead".
	Upshift StepHandler
	// Activity reports an activity state. nil means the resource does not
	// report activity.
	Activity ActivityHandler
	// Stdout and Stderr receive confirmations and usage. nil means the process
	// defaults.
	Stdout io.Writer
	Stderr io.Writer
}

// Run dispatches `capacity <subcommand>` for a resource CLI.
func (v Verbs) Run(args []string) error {
	stdout := v.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := v.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	if len(args) == 0 {
		v.usage(stderr)
		return fmt.Errorf("capacity requires a subcommand")
	}

	subcommand, rest := args[0], args[1:]
	switch subcommand {
	case "degrade", "upshift":
		label, err := parseStepFlags(subcommand, rest, stderr)
		if err != nil {
			return err
		}
		handler := v.Degrade
		if subcommand == "upshift" && v.Upshift != nil {
			handler = v.Upshift
		}
		if handler == nil {
			return fmt.Errorf("resource %s declares no capacity %s handler", v.Resource, subcommand)
		}
		if err := handler(context.Background(), label); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%s capacity %s: now at %q\n", v.Resource, subcommand, label)
		return nil

	case "activity":
		state, err := parseActivityFlags(rest, stderr)
		if err != nil {
			return err
		}
		if v.Activity == nil {
			return fmt.Errorf("resource %s does not report capacity activity", v.Resource)
		}
		if err := v.Activity(context.Background(), state); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%s capacity activity: %s\n", v.Resource, state)
		return nil
	}

	v.usage(stderr)
	return fmt.Errorf("unknown capacity subcommand %q", subcommand)
}

func (v Verbs) usage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  %s capacity degrade  --to <label>
  %s capacity upshift  --to <label>
  %s capacity activity --state <active|idle>
`, v.Resource, v.Resource, v.Resource)
}

func parseStepFlags(name string, args []string, stderr io.Writer) (string, error) {
	set := flag.NewFlagSet("capacity "+name, flag.ContinueOnError)
	set.SetOutput(stderr)
	to := set.String("to", "", "target profile step label")
	if err := set.Parse(args); err != nil {
		return "", err
	}
	label := strings.TrimSpace(*to)
	if label == "" {
		return "", fmt.Errorf("capacity %s requires --to <label>", name)
	}
	return label, nil
}

func parseActivityFlags(args []string, stderr io.Writer) (string, error) {
	set := flag.NewFlagSet("capacity activity", flag.ContinueOnError)
	set.SetOutput(stderr)
	state := set.String("state", "", "active|idle")
	if err := set.Parse(args); err != nil {
		return "", err
	}
	value := strings.ToLower(strings.TrimSpace(*state))
	if value != ActivityActive && value != ActivityIdle {
		return "", fmt.Errorf("capacity activity requires --state %s or --state %s", ActivityActive, ActivityIdle)
	}
	return value, nil
}

// StepsFromLabels builds a StepHandler that accepts only the declared labels,
// so a broker asking for a rung the resource does not have gets an error rather
// than a silent no-op.
func StepsFromLabels(labels []string, apply StepHandler) StepHandler {
	allowed := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		allowed[label] = struct{}{}
	}
	return func(ctx context.Context, label string) error {
		if _, ok := allowed[label]; !ok {
			return fmt.Errorf("%w %q (declared steps: %s)", ErrUnknownStep, label, strings.Join(labels, ", "))
		}
		return apply(ctx, label)
	}
}
