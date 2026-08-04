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
		return fmt.Errorf("usage: agent-manager run mine-self-report-vocabulary [--output review.json] <transcript.jsonl> [more files]")
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
	cohort := fs.String("cohort", "", "durable cohort name")
	limit := fs.Int("limit", 100, "maximum runs")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := url.Values{"limit": []string{fmt.Sprint(*limit)}}
	if *tagPrefix != "" {
		values.Set("tag_prefix", *tagPrefix)
	}
	if *cohort != "" {
		values.Set("cohort", *cohort)
	}
	body, err := a.services.Runs.EpisodeCohort(values)
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

func invocationQueryFlags(fs *flag.FlagSet) (dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus, cohort *string, limit *int) {
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
	cohort = fs.String("cohort", "", "durable cohort name")
	limit = fs.Int("limit", 100, "maximum results")
	return
}

func invocationQueryValues(dimension, ownership, outcome, executable, fingerprint, from, to, profileID, runnerType, model, tagPrefix, runStatus, cohort string, limit int) url.Values {
	values := url.Values{"limit": []string{fmt.Sprint(limit)}}
	for key, value := range map[string]string{"dimension": dimension, "ownership": ownership, "outcome": outcome, "executable": executable, "fingerprint": fingerprint, "from": from, "to": to, "profile_id": profileID, "runner_type": runnerType, "model": model, "tag_prefix": tagPrefix, "run_status": runStatus, "cohort": cohort} {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func (a *App) runAggregateInvocationFacts(args []string) error {
	fs := flag.NewFlagSet("run invocation-aggregate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	d, o, u, e, f, from, to, p, r, m, t, s, c, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *d == "" {
		return fmt.Errorf("--dimension is required")
	}
	body, err := a.services.Runs.AggregateInvocationFacts(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *c, *l))
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
	d, o, u, e, f, from, to, p, r, m, t, s, c, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.SelectInvocationCohort(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *c, *l))
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
	d, o, u, e, f, from, to, p, r, m, t, s, c, l := invocationQueryFlags(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationMetrics(invocationQueryValues(*d, *o, *u, *e, *f, *from, *to, *p, *r, *m, *t, *s, *c, *l))
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

func (a *App) runCohort(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-manager run cohort <define|list|show|delete> [args]")
	}
	jsonOutput := false
	printBody := func(body []byte) {
		if jsonOutput {
			cliutil.PrintJSON(body)
		} else {
			fmt.Println(string(body))
		}
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("run cohort list", flag.ContinueOnError)
		jsonFlag := cliutil.JSONFlag(fs)
		if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
			return err
		}
		jsonOutput = *jsonFlag
		body, err := a.services.Runs.ListCohorts()
		if err != nil {
			return err
		}
		printBody(body)
		return nil
	case "define":
		fs := flag.NewFlagSet("run cohort define", flag.ContinueOnError)
		jsonFlag := cliutil.JSONFlag(fs)
		nameFlag := fs.String("name", "", "cohort name")
		filterJSON := fs.String("filter-json", "", "JSON invocation filter")
		changeBinding := fs.String("change-binding", "", "change or release binding")
		if len(args) < 2 && strings.TrimSpace(*nameFlag) == "" {
			return fmt.Errorf("usage: agent-manager run cohort define <name> --filter-json '{}' [--change-binding value] [--json]")
		}
		flagArgs := args[1:]
		name := ""
		if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
			name = args[1]
			flagArgs = args[2:]
		}
		if err := cliutil.ParseInterspersed(fs, flagArgs); err != nil {
			return err
		}
		if strings.TrimSpace(name) == "" {
			name = strings.TrimSpace(*nameFlag)
		}
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if strings.TrimSpace(*filterJSON) == "" {
			return fmt.Errorf("--filter-json is required")
		}
		var filter map[string]any
		if err := json.Unmarshal([]byte(*filterJSON), &filter); err != nil {
			return fmt.Errorf("--filter-json must be valid JSON: %w", err)
		}
		payload, err := json.Marshal(map[string]any{"name": name, "filter": filter, "changeBinding": *changeBinding})
		if err != nil {
			return err
		}
		jsonOutput = *jsonFlag
		body, err := a.services.Runs.DefineCohort(payload)
		if err != nil {
			return err
		}
		printBody(body)
		return nil
	case "show":
		fs := flag.NewFlagSet("run cohort show", flag.ContinueOnError)
		jsonFlag := cliutil.JSONFlag(fs)
		limit := fs.Int("limit", 100, "maximum matched runs")
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return fmt.Errorf("usage: agent-manager run cohort show <name> [--limit n] [--json]")
		}
		if err := cliutil.ParseInterspersed(fs, args[2:]); err != nil {
			return err
		}
		jsonOutput = *jsonFlag
		body, err := a.services.Runs.ShowCohort(args[1], *limit)
		if err != nil {
			return err
		}
		printBody(body)
		return nil
	case "delete":
		fs := flag.NewFlagSet("run cohort delete", flag.ContinueOnError)
		jsonFlag := cliutil.JSONFlag(fs)
		if len(args) < 2 || strings.HasPrefix(args[1], "-") {
			return fmt.Errorf("usage: agent-manager run cohort delete <name> [--json]")
		}
		if err := cliutil.ParseInterspersed(fs, args[2:]); err != nil {
			return err
		}
		if _, err := a.services.Runs.DeleteCohort(args[1]); err != nil {
			return err
		}
		if *jsonFlag {
			cliutil.PrintJSON([]byte(`{"deleted":true}`))
		} else {
			fmt.Println("deleted")
		}
		return nil
	default:
		return fmt.Errorf("unknown cohort operation %q", args[0])
	}
}

func (a *App) runCrossScenario(args []string) error {
	fs := flag.NewFlagSet("run cross-scenario", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run cross-scenario <run-id> [--json]")
	}
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.InvocationFacts(args[0])
	if err != nil {
		return err
	}
	var response struct {
		Facts []crossScenarioFact `json:"facts"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode invocation facts: %w", err)
	}
	metrics := crossScenarioMetrics(response.Facts)
	encoded, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(encoded)
	} else {
		fmt.Println(string(encoded))
	}
	return nil
}

type crossScenarioFact struct {
	Executable    string `json:"executable"`
	CommandPath   string `json:"commandPath"`
	IntentClass   string `json:"intentClass"`
	SemanticsKind string `json:"semanticsKind"`
	Outcome       string `json:"outcome"`
}

type crossScenarioReport struct {
	Available        bool   `json:"available"`
	Reason           string `json:"reason,omitempty"`
	DeclaredCalls    int    `json:"declaredCalls"`
	ClassifiedBase   int    `json:"classifiedBase"`
	QueryRefinement  int    `json:"queryRefinement"`
	QueryAbandonment int    `json:"queryAbandonment"`
	VerifyCycle      int    `json:"verifyCycle"`
	VerifyRegression int    `json:"verifyRegression"`
	GuidanceAdoption int    `json:"guidanceAdoption"`
}

func crossScenarioMetrics(facts []crossScenarioFact) crossScenarioReport {
	result := crossScenarioReport{}
	families := map[string][]crossScenarioFact{}
	for _, fact := range facts {
		if fact.SemanticsKind == "" {
			continue
		}
		result.DeclaredCalls++
		families[fact.Executable+" "+fact.CommandPath] = append(families[fact.Executable+" "+fact.CommandPath], fact)
	}
	result.ClassifiedBase = result.DeclaredCalls
	if result.DeclaredCalls == 0 {
		result.Reason = "no invocation command declares cross-scenario semantics"
		return result
	}
	result.Available = true
	for _, calls := range families {
		for i := 1; i < len(calls); i++ {
			if calls[i].SemanticsKind == "query" && calls[i-1].SemanticsKind == "query" && calls[i].IntentClass != calls[i-1].IntentClass {
				result.QueryRefinement++
			}
		}
		seenPass := false
		for i, call := range calls {
			switch call.SemanticsKind {
			case "query":
				if i == len(calls)-1 && call.Outcome != "success" {
					result.QueryAbandonment++
				}
			case "verify":
				if call.Outcome != "success" {
					result.VerifyCycle++
				}
				if seenPass && call.Outcome == "failure" {
					result.VerifyRegression++
				}
				if call.Outcome == "success" {
					seenPass = true
				}
			}
		}
	}
	for i, fact := range facts {
		if fact.SemanticsKind != "guidance" {
			continue
		}
		before, after := map[string]bool{}, map[string]bool{}
		for _, candidate := range facts[:i] {
			if candidate.SemanticsKind != "guidance" {
				before[candidate.Executable+" "+candidate.CommandPath] = true
			}
		}
		for _, candidate := range facts[i+1:] {
			if candidate.SemanticsKind != "guidance" {
				after[candidate.Executable+" "+candidate.CommandPath] = true
			}
		}
		for family := range after {
			if !before[family] {
				result.GuidanceAdoption++
				break
			}
		}
	}
	return result
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
