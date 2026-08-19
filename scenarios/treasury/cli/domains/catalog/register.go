// Package catalog exposes read-only ownership summaries for Treasury domains
// whose records are reached through a workflow RPC rather than a standalone
// service. These commands remain useful during an API outage and render the
// same operational report shape in human and --json modes.
package catalog

import "github.com/vrooli/cli-core/cliapp"

var descriptions = map[string]string{
	"book":     "Operator-owned source-of-funds boundary and book registry",
	"rail":     "Registered payment adapters and their execution boundary",
	"evidence": "Append-only spend-attempt evidence and retention contract",
	"ledger":   "Exactly-once Money Ledger emission outbox",
	"identity": "Fail-closed Agent Manager identity verification boundary",
}

func Register(name string) cliapp.SubcommandGroup {
	description := descriptions[name]
	return cliapp.SubcommandGroup{
		Name: name, Description: description,
		Subcommands: []cliapp.Command{{
			Name: "describe", Description: "Describe this domain's owned financial boundary",
			RunCtx: Describe(name),
		}},
	}
}

func Describe(name string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		return ctx.RenderOperational(cliapp.OperationalReport{
			Status:    []string{descriptions[name]},
			NextSteps: []string{"Use `treasury safety invariants --json` to inspect the cross-domain safety contract."},
		})
	}
}
