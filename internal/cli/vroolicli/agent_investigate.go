package vroolicli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/recovery"
)

type codingAgentInvestigationBroker interface {
	ReconcileNative(context.Context, int) (int, error)
	SubmitContext(context.Context, recovery.CodingAgentRequest) (recovery.CodingAgentRecord, error)
}

func newCodingAgentInvestigationBroker() (codingAgentInvestigationBroker, error) {
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	worker, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{UserHome: home})
	if err != nil {
		return nil, err
	}
	return recovery.NewNativeCodingAgentBroker(path, worker)
}

func runCodingAgentInvestigation(ctx *AppContext, args []string, open func() (codingAgentInvestigationBroker, error)) error {
	fs := commandtree.NewFlagSet("vrooli agent recover investigate")
	fs.SetOutput(ctx.Stderr)
	scenario := fs.String("scenario", "", "scenario to investigate")
	reason := fs.String("reason", "", "bounded health failure or a path to its evidence")
	asJSON := fs.Bool("json", ctx.Globals.JSON, "emit the admitted investigation receipt")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	target := strings.TrimSpace(*scenario)
	why := strings.TrimSpace(*reason)
	if fs.NArg() != 0 || target == "" || why == "" {
		return clipolicy.UsageErrorf("agent recover investigate", "--scenario and --reason are required; no positional arguments")
	}
	root := strings.TrimSpace(ctx.Root)
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return fmt.Errorf("coding-agent investigation requires the resolved repository root")
	}
	broker, err := open()
	if err != nil {
		return err
	}
	if _, err := broker.ReconcileNative(ctx.OperationContext(), 32); err != nil {
		return fmt.Errorf("native recovery reconciliation unavailable: %w", err)
	}
	record, submitErr := broker.SubmitContext(ctx.OperationContext(), recovery.CodingAgentRequest{
		Scenario: target, Reason: why, Requester: "vrooli-cli",
		WorkingDir: root, Claim: filepath.Join("scenarios", target),
	})
	// An ambiguous dispatch retains its admission identity. Emit it even when
	// returning an error so the operator can inspect it without resubmission.
	if record.ID != "" && ctx.Stdout != nil {
		var outputErr error
		if *asJSON {
			outputErr = json.NewEncoder(ctx.Stdout).Encode(record)
		} else {
			_, outputErr = fmt.Fprintf(ctx.Stdout, "coding-agent investigation %s: scenario=%s outcome=%s deadline=%s\n", record.ID, record.Scenario, record.Outcome, record.DeadlineAt.UTC().Format("2006-01-02T15:04:05Z"))
		}
		return errors.Join(submitErr, outputErr)
	}
	return submitErr
}
