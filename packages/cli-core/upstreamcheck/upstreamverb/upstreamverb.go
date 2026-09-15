// Package upstreamverb wires the shared upstreamcheck logic into a cliapp
// subcommand group. It is split from the core upstreamcheck package so that
// callers needing only the check logic / aggregate (e.g. the root vrooli CLI)
// do not pull in the cliapp framework and its transitive dependencies.
package upstreamverb

import (
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// Commands returns the `upstream-check` subgroup. The lone `check`
// subcommand is the default, so `resource-* upstream-check [--json]` works
// without naming it.
func Commands(h *upstreamcheck.Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = upstreamcheck.Default(upstreamcheck.Config{})
	}
	return cliapp.SubcommandGroup{
		Name:              "upstream-check",
		Description:       "Compare the installed CLI against the latest upstream release (read-only)",
		DefaultSubcommand: "check",
		Subcommands: []cliapp.Command{
			{Name: "check", Description: "Report installed-vs-upstream version state", Run: h.Check},
		},
	}
}
