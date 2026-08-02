package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	domainconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
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
	from := fs.String("from", "", "RFC3339 inclusive session-update time")
	to := fs.String("to", "", "RFC3339 exclusive session-update time")
	perMonth := fs.Int("per-month", 1, "deterministic sessions per runner-month")
	limit := fs.Int("limit", 24, "maximum selected sessions")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	runnerTypes := make([]string, 0)
	for _, value := range strings.Split(*runners, ",") {
		if value = strings.TrimSpace(value); value != "" {
			runnerTypes = append(runnerTypes, value)
		}
	}
	payload, err := json.Marshal(map[string]any{"runnerTypes": runnerTypes, "from": *from, "to": *to, "perMonth": *perMonth, "limit": *limit})
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

func (a *App) runInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationFacts(args[0])
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

func (a *App) runEpisodes(args []string) error {
	fs := flag.NewFlagSet("run episodes", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run episodes <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := domainconnect.NewEpisodesServiceClient(h, base).GetEpisodes(context.Background(), connect.NewRequest(&domainpb.GetEpisodesRequest{RunId: args[0]}))
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

func (a *App) runMessageFriction(args []string) error {
	fs := flag.NewFlagSet("run messages-friction", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run messages-friction <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := domainconnect.NewEpisodesServiceClient(h, base).GetSelfReportSpans(context.Background(), connect.NewRequest(&domainpb.GetSelfReportSpansRequest{RunId: args[0]}))
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

func (a *App) runEpisodeCohort(args []string) error {
	fs := flag.NewFlagSet("run episode-cohort", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	tagPrefix := fs.String("tag-prefix", "", "tag prefix")
	limit := fs.Int("limit", 100, "maximum runs")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := measureconnect.NewMeasuresServiceClient(h, base).EpisodeCohort(context.Background(), connect.NewRequest(&measurepb.EpisodeCohortRequest{
		Filter: &measurepb.InvocationFilter{TagPrefix: *tagPrefix},
		Limit:  int32(*limit),
	}))
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

func (a *App) runRefreshInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run refresh-invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run refresh-invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.RefreshInvocationFacts(args[0])
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

func (a *App) runReplayInvocationCorpus(args []string) error {
	fs := flag.NewFlagSet("run replay-invocation-corpus", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	from := fs.String("from", "", "RFC3339 inclusive start")
	to := fs.String("to", "", "RFC3339 exclusive end")
	status := fs.String("status", "", "run status")
	profileID := fs.String("profile-id", "", "agent profile UUID")
	tagPrefix := fs.String("tag-prefix", "", "tag prefix")
	limit := fs.String("limit", "", "maximum runs")
	refresh := fs.Bool("refresh", false, "refresh only runs with events newer than watermark")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := url.Values{}
	for key, value := range map[string]string{"from": *from, "to": *to, "status": *status, "profile_id": *profileID, "tag_prefix": *tagPrefix, "limit": *limit} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if *refresh {
		values.Set("mode", "refresh")
	}
	body, err := a.services.Runs.ReplayInvocationCorpus(values)
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

func invocationQueryFlags(fs *flag.FlagSet) (dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus, goalStatus *string, limit *int) {
	dimension = fs.String("dimension", "", "aggregate dimension")
	ownership = fs.String("ownership", "", "ownership filter")
	outcome = fs.String("outcome", "", "outcome filter")
	executable = fs.String("executable", "", "executable filter")
	fingerprint = fs.String("fingerprint", "", "fingerprint filter")
	from = fs.String("from", "", "RFC3339 start")
	to = fs.String("to", "", "RFC3339 end")
	profileID = fs.String("profile-id", "", "profile filter")
	runnerType = fs.String("runner-type", "", "runner filter")
	model = fs.String("model", "", "model filter")
	tagPrefix = fs.String("tag-prefix", "", "tag prefix")
	runStatus = fs.String("run-status", "", "status filter")
	goalStatus = fs.String("goal-status", "", "external goal outcome filter")
	limit = fs.Int("limit", 100, "maximum results")
	return
}

func invocationQueryValues(dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus, goalStatus string, limit int) url.Values {
	values := url.Values{"limit": []string{fmt.Sprint(limit)}}
	for key, value := range map[string]string{"dimension": dimension, "ownership": ownership, "outcome": outcome, "executable": executable, "fingerprint": fingerprint, "from": from, "to": to, "profile_id": profileID, "runner_type": runnerType, "model": model, "tag_prefix": tagPrefix, "run_status": runStatus, "goal_status": goalStatus} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func (a *App) runAggregateInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-aggregate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, g, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *d == "" {
		return fmt.Errorf("--dimension is required")
	}
	body, err := a.services.Runs.AggregateInvocationFacts(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *g, *l))
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

func (a *App) runSelectInvocationCohort(args []string) error {
	fs := flag.NewFlagSet("run invocation-cohort", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, g, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.SelectInvocationCohort(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *g, *l))
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

func (a *App) runInvocationMetrics(args []string) error {
	fs := flag.NewFlagSet("run invocation-metrics", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, g, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationMetrics(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *g, *l))
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

func (a *App) runReplayInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run replay-invocation-facts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run replay-invocation-facts <id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.ReplayInvocationFacts(args[0])
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
