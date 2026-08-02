package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"unicode"

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
	strategy := fs.String("strategy", "stratified", "selection strategy: deterministic-per-month, stratified, recent, or all")
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

// runMineSelfReportVocabulary is deliberately offline. It reads assistant
// turns from transcript files and writes ranked review candidates; promotion
// into the embedded rule pack remains a human, versioned decision.
func (a *App) runMineSelfReportVocabulary(args []string) error {
	fs := flag.NewFlagSet("run mine-self-report-vocabulary", flag.ContinueOnError)
	output := fs.String("output", "", "review JSON path (default stdout)")
	limit := fs.Int("limit", 100, "maximum ranked candidates")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	paths := fs.Args()
	if len(paths) == 0 {
		return fmt.Errorf("usage: agent-manager run mine-self-report-vocabulary [--output review.json] <transcript.jsonl> ...")
	}
	type candidate struct {
		Phrase string `json:"phrase"`
		Count  int    `json:"count"`
	}
	counts := map[string]int{}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open transcript %s: %w", path, err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 4<<20)
		for scanner.Scan() {
			var value map[string]any
			if json.Unmarshal(scanner.Bytes(), &value) != nil || strings.ToLower(stringValue(value, "role", "type")) != "assistant" {
				continue
			}
			text := stringValue(value, "content", "text", "message")
			words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\'' })
			for n := 3; n <= 6; n++ {
				for i := 0; i+n <= len(words); i++ {
					phrase := strings.Join(words[i:i+n], " ")
					if hasStruggleVocabulary(phrase) {
						counts[phrase]++
					}
				}
			}
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read transcript %s: %w", path, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close transcript %s: %w", path, closeErr)
		}
	}
	items := make([]candidate, 0, len(counts))
	for phrase, count := range counts {
		items = append(items, candidate{Phrase: phrase, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Phrase < items[j].Phrase
	})
	if *limit < 1 {
		return fmt.Errorf("--limit must be positive")
	}
	if len(items) > *limit {
		items = items[:*limit]
	}
	body, err := json.MarshalIndent(map[string]any{"strategy": "frequency-and-struggle-vocabulary", "sourceCount": len(paths), "candidates": items}, "", "  ")
	if err != nil {
		return err
	}
	if *output != "" {
		if err := os.WriteFile(*output, append(body, '\n'), 0o600); err != nil {
			return fmt.Errorf("write review file: %w", err)
		}
		fmt.Printf("wrote %d review candidates to %s\n", len(items), *output)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func stringValue(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := value[key].(string); ok {
			return text
		}
	}
	return ""
}

func hasStruggleVocabulary(phrase string) bool {
	for _, term := range []string{"blocked", "stuck", "cannot", "can't", "unable", "failed", "wait", "confused", "instead", "wrong", "retry", "permission"} {
		if strings.Contains(phrase, term) {
			return true
		}
	}
	return false
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

func invocationQueryFlags(fs *flag.FlagSet) (dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus *string, limit *int) {
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
	limit = fs.Int("limit", 100, "maximum results")
	return
}

func invocationQueryValues(dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus string, limit int) url.Values {
	values := url.Values{"limit": []string{fmt.Sprint(limit)}}
	for key, value := range map[string]string{"dimension": dimension, "ownership": ownership, "outcome": outcome, "executable": executable, "fingerprint": fingerprint, "from": from, "to": to, "profile_id": profileID, "runner_type": runnerType, "model": model, "tag_prefix": tagPrefix, "run_status": runStatus} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func (a *App) runAggregateInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-aggregate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *d == "" {
		return fmt.Errorf("--dimension is required")
	}
	body, err := a.services.Runs.AggregateInvocationFacts(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
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
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.SelectInvocationCohort(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
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
	d, o, u, e, f, from, to, p, r, m, t, s, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationMetrics(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *l))
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
