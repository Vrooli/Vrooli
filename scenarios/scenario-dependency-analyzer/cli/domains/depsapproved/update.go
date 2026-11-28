package depsapproved

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"scenario-dependency-analyzer/cli/internal/support"
)

const updateUsage = "usage: %s deps update (<ecosystem>/<package>... | --advisory <GHSA|CVE>) --scenario <name> --surface <surface> [--apply] [--json]"

func updateArgumentSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{
			Name:        "dependency",
			Repeated:    true,
			Description: "Locked JavaScript package to update, as npm/<package> with no version (direct or transitive); name several to update them with one resolution. Omit when selecting by --advisory",
		}},
		Flags: []cliapp.Flag{
			{Name: "scenario", Description: "Required target scenario"},
			{Name: "surface", Description: "Required target surface: ui, api, cli, tools/<package>, ..."},
			{Name: "advisory", Description: "Select the packages by security advisory (GHSA or CVE id; several separated by commas; \"all\" for every advisory) instead of by name: the packages Security Health reports as exposed to it on this surface"},
			{Name: "lock-only", Bool: true, Description: "With --apply, publish the lockfile without linking node_modules: nothing is downloaded and no package store is written, so a running consumer keeps serving and its next governed setup links the new lock"},
			{Name: "apply", Bool: true, Description: "Perform the update; default is a dry-run preview"},
		},
	}
}

// runUpdate moves already-locked packages to the newest version their declared
// ranges allow, through the governed gateway.
func runUpdate(core *cliapp.ScenarioApp, args cliapp.RunContext) error {
	scenario, surface := strings.TrimSpace(args.Flag("scenario")), strings.TrimSpace(args.Flag("surface"))
	if scenario == "" || surface == "" {
		return fmt.Errorf("--scenario and --surface are required; %s", fmt.Sprintf(updateUsage, support.AppName))
	}
	advisory := strings.TrimSpace(args.Flag("advisory"))
	ecosystem, names, err := updateSelection(advisory, args.Positionals("dependency"))
	if err != nil {
		return err
	}
	timeout := 3*time.Minute + 15*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := governanceClientWithTimeout(core, timeout).UpdateLockedDependency(ctx, connect.NewRequest(&governancev1.UpdateLockedDependencyRequest{
		Scenario: scenario, Surface: surface, Ecosystem: ecosystem, PackageNames: names, Advisory: advisory, LockOnly: args.BoolFlag("lock-only"), Apply: args.BoolFlag("apply"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("update locked dependency", err, nil)
	}
	if args.JSON() {
		if err := printProto(resp.Msg); err != nil {
			return err
		}
		return updateOutcome(resp.Msg)
	}
	msg := resp.Msg
	summary := []string{msg.GetMessage(), fmt.Sprintf("Verdict: %s", msg.GetVerdict()), fmt.Sprintf("Command: %s", msg.GetCommand())}
	summary = append(summary, msg.GetNotes()...)
	if err := support.PrintList(false, cliapp.ListReport{Summary: summary, ResultsHeading: "Locked Dependency Update", Results: msg.GetNextSteps()}, nil); err != nil {
		return err
	}
	return updateOutcome(msg)
}

// updateSelection reads what to update: package specs or one advisory, never
// both and never neither.
func updateSelection(advisory string, specs []string) (ecosystem string, names []string, err error) {
	ecosystem, names, err = parseRemoveSpecs(specs)
	if err != nil {
		return "", nil, err
	}
	if (advisory == "") == (len(names) == 0) {
		return "", nil, fmt.Errorf("name the packages to update or select them with --advisory, not both and not neither; %s", fmt.Sprintf(updateUsage, support.AppName))
	}
	return ecosystem, names, nil
}

// updateOutcome is nil for every result that leaves the surface consistent: a
// preview, an applied update, a lock-only update, or nothing newer to take.
func updateOutcome(msg *governancev1.UpdateLockedDependencyResponse) error {
	if !msg.GetBlocked() && (msg.GetDryRun() || msg.GetUpdated() || msg.GetSourceUpdated() || msg.GetVerdict() == "current" || msg.GetVerdict() == "not_affected") {
		return nil
	}
	return fmt.Errorf("locked dependency update did not succeed (verdict=%s, blocked=%t, updated=%t)", msg.GetVerdict(), msg.GetBlocked(), msg.GetUpdated())
}
