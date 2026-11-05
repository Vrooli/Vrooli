package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
)

func (a *App) runImportTranscript(args []string) error {
	fs := flag.NewFlagSet("run import-transcript", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	runnerType := fs.String("runner", "", "runner type")
	label := fs.String("label", "", "import label")
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run import-transcript <path> [--runner type] [--label value] [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := domainconnect.NewEpisodesServiceClient(h, base).ImportTranscript(context.Background(), connect.NewRequest(&domainpb.ImportTranscriptRequest{Path: args[0], RunnerType: *runnerType, Label: *label}))
	if err != nil {
		return err
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func (a *App) runImportSessionCorpus(args []string) error {
	fs := flag.NewFlagSet("run import-session-corpus", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	runners := fs.String("runners", "codex,claude-code", "comma-separated governed runner types")
	strategy := fs.String("strategy", "stratified", "selection strategy: deterministic-per-month, stratified, recent, or all")
	from := fs.String("from", "", "RFC3339 inclusive session-update time")
	to := fs.String("to", "", "RFC3339 exclusive session-update time")
	perMonth := fs.Int("per-month", 1, "deterministic sessions per runner-month")
	limit := fs.Int("limit", 5000, "maximum selected sessions (the governed all-corpus ceiling)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	runnerTypes := make([]string, 0)
	for _, value := range strings.Split(*runners, ",") {
		if value = strings.TrimSpace(value); value != "" {
			runnerTypes = append(runnerTypes, value)
		}
	}
	payload, err := json.Marshal(map[string]any{"runnerTypes": runnerTypes, "strategy": *strategy, "from": *from, "to": *to, "perMonth": *perMonth, "limit": *limit})
	if err != nil {
		return err
	}
	body, err := a.services.Runs.ImportSessionCorpus(payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runBackfillLabels(args []string) error {
	fs := flag.NewFlagSet("maintenance backfill-labels", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.BackfillLabels()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runImportSweep(args []string) error {
	fs := flag.NewFlagSet("run import-sweep", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.ImportSweep()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runBackfillSubjects(args []string) error {
	fs := flag.NewFlagSet("maintenance backfill-subjects", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.BackfillSubjects()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println(string(body))
	}
	return nil
}

func (a *App) runLedger(args []string) error {
	fs := flag.NewFlagSet("run ledger", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	withProjections := fs.Bool("with-projections", false, "include bounded receipt projections")
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run ledger <id> [--with-projections] [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := domainconnect.NewEpisodesServiceClient(h, base).GetCrossScenarioLedger(context.Background(), connect.NewRequest(&domainpb.GetCrossScenarioLedgerRequest{RunId: args[0], WithProjections: *withProjections}))
	if err != nil {
		return err
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}
