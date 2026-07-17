// Package onboard is the CLI's onboard-domain command surface (`onboard …`),
// a thin wrapper over the API's Connect-RPC OnboardService. It gives the owner
// the one-shot entrypoints — start an end-to-end onboard, watch its live step
// states, and inspect/cancel ops — that drive a raw SSH host from bare OS to a
// paired, ONLINE fleet agent. The manual pair issue/redeem verbs remain for
// advanced flows; this domain is the walk-away path.
//
// All five verbs are owner-gated. The SSH password is NEVER a flag value and
// `start` NEVER prompts unless asked: it is read from stdin (--password-stdin),
// a masked TTY prompt (--prompt-password, explicit opt-in), or
// $BRIDGE_SSH_PASSWORD, and sent once in the request body — never on argv,
// where `ps` could leak it to any local user. With no source provided the host
// is assumed to already trust the bridge key. The UI onboard form is the
// equivalent browser path.
package onboard

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "onboard"

// Register builds the onboard subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go. The node-facing
// progress arrives over the SSH channel the orchestrator owns, so there is no
// node-facing RPC to omit here — every OnboardService method is an operator verb.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"OnboardService.StartOnboarding":        h.start,
		"OnboardService.GetOnboarding":          h.status,
		"OnboardService.ListOnboardings":        h.list,
		"OnboardService.WaitOnboarding":         h.watch,
		"OnboardService.CancelOnboarding":       h.cancel,
		"OnboardService.RemoveFailedOnboarding": h.removeFailed,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("onboard: load from manifest: %w", err)
	}
	return group, nil
}
