package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Record mirrors the API shape (api/internal/records.Record).
type Record struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Scenario     string   `json:"scenario"`
	BacklogRef   string   `json:"backlog_ref,omitempty"`
	MilestoneID  string   `json:"milestone_id,omitempty"`
	Supersedes   string   `json:"supersedes,omitempty"`
	SupersededBy string   `json:"superseded_by,omitempty"`
	Trigger      string   `json:"trigger"`
	Approach     string   `json:"approach"`
	RuledOut     []string `json:"ruled_out,omitempty"`
	Evidence     string   `json:"evidence,omitempty"`
	Commit       string   `json:"commit,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
	Outcome      string   `json:"outcome"`
	Stub         bool     `json:"stub"`
	CreatedAt    string   `json:"created_at"`
	CreatedBy    string   `json:"created_by,omitempty"`
	NarrativeAt  string   `json:"narrative_at,omitempty"`
}

// RecordEnvelope wraps single-record endpoints. Warnings carry non-blocking
// server advice (e.g. off-registry scenario slug) on create.
type RecordEnvelope struct {
	Record   Record   `json:"record"`
	Warnings []string `json:"warnings,omitempty"`
}

type RecordCaptureResponse struct {
	Disposition string   `json:"disposition"`
	Record      Record   `json:"record"`
	Needs       []string `json:"needs"`
	Invalid     []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"invalid"`
	Warnings   []string `json:"warnings"`
	NextAction []string `json:"next_action"`
}

func (a *App) cmdRecordsCapture(args []string) error {
	fs := flag.NewFlagSet("records capture", flag.ContinueOnError)
	kind := fs.String("kind", "", "Record kind (or documented alias)")
	scenario := fs.String("scenario", "", "Target scenario slug")
	trigger := fs.String("trigger", "", "One-line goal or symptom")
	approach := fs.String("approach", "", "What was built or learned")
	evidence := fs.String("evidence", "", "Validation evidence")
	outcome := fs.String("outcome", "", "Outcome category")
	createdBy := fs.String("created-by", "", "Author identifier")
	idempotencyKey := fs.String("idempotency-key", "", "Optional retry-safe capture key")
	var ruledOut stringSlice
	fs.Var(&ruledOut, "ruled-out", "Rejected hypothesis (repeatable)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return recordsFlagError(fs, err)
	}
	payload, err := json.Marshal(map[string]any{"kind": *kind, "scenario": *scenario, "trigger": *trigger, "approach": *approach, "evidence": *evidence, "outcome": *outcome, "created_by": *createdBy, "idempotency_key": *idempotencyKey, "ruled_out": []string(ruledOut)})
	if err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/records/capture", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[RecordCaptureResponse](body)
	if err != nil {
		return err
	}
	printSection("Result")
	fmt.Printf("  %s: %s\n", strings.ToUpper(resp.Disposition), resp.Record.ID)
	if resp.Disposition == "draft" {
		fmt.Println("  NOT PUBLISHED — private draft saved.")
		if len(resp.Needs) > 0 {
			fmt.Printf("  Needs: %s\n", strings.Join(resp.Needs, ", "))
		}
		for _, v := range resp.Invalid {
			fmt.Printf("  Invalid %s: %s\n", v.Field, v.Message)
		}
		if len(resp.NextAction) > 0 {
			printCommandListSection("Repair", []string{strings.Join(resp.NextAction, " ")})
		}
	}
	return nil
}

// ListRecordsResponse wraps GET /records.
type ListRecordsResponse struct {
	Records []Record `json:"records"`
}

// RecordSearchHit wraps a semantic-search match.
type RecordSearchHit struct {
	Record Record  `json:"record"`
	Score  float64 `json:"score"`
}

// RecordSearchResponse wraps POST /records/search.
type RecordSearchResponse struct {
	Hits []RecordSearchHit `json:"hits"`
}

func (a *App) cmdRecordsList(args []string) error {
	fs := flag.NewFlagSet("records list", flag.ContinueOnError)
	scenarioFlag := fs.String("scenario", "", "Filter by scenario slug")
	kindFlag := fs.String("kind", "", "Filter by kind (idea|research|fix|execute|chore)")
	backlogRefFlag := fs.String("backlog-ref", "", "Filter by backlog ref (kind/name)")
	includeStubsFlag := fs.Bool("include-stubs", false, "Include stub records (hidden by default)")
	limitFlag := fs.Int("limit", 0, "Max results")
	offsetFlag := fs.Int("offset", 0, "Pagination offset")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := url.Values{}
	if v := strings.TrimSpace(*scenarioFlag); v != "" {
		q.Set("scenario", v)
	}
	if v := strings.TrimSpace(*kindFlag); v != "" {
		q.Set("kind", v)
	}
	if v := strings.TrimSpace(*backlogRefFlag); v != "" {
		q.Set("backlog_ref", v)
	}
	if *includeStubsFlag {
		q.Set("include_stubs", "true")
	}
	if *limitFlag > 0 {
		q.Set("limit", strconv.Itoa(*limitFlag))
	}
	if *offsetFlag > 0 {
		q.Set("offset", strconv.Itoa(*offsetFlag))
	}
	path := "/records"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	body, err := a.core.Get(path, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[ListRecordsResponse](body)
	if err != nil {
		return err
	}

	if len(resp.Records) == 0 {
		printSection("Summary")
		fmt.Println("  No records found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("records", "create", "--kind", "fix", "--scenario", "<scenario>",
				"--trigger", "'<one-line symptom>'", "--approach", "'<root cause + fix>'", "--outcome", "shipped"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d record(s)\n", len(resp.Records))

	printSection("Results")
	for _, r := range resp.Records {
		stub := ""
		if r.Stub {
			stub = " [STUB]"
		}
		ref := ""
		if r.BacklogRef != "" {
			ref = "  backlog:" + r.BacklogRef
		}
		preview := r.Trigger
		if preview == "" {
			preview = "(no trigger)"
		}
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		fmt.Printf("  [%s/%s]%s  %s  %s%s\n", r.Kind, r.Scenario, stub, r.ID, preview, ref)
	}

	first := resp.Records[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("records", "get", "--id", first.ID),
		cliCommand("records", "search", "'<query>'"),
	})
	return nil
}

func (a *App) cmdRecordsGet(args []string) error {
	fs := flag.NewFlagSet("records get", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Record ID (ULID)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: records get --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Get("/records/"+id, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[RecordEnvelope](body)
	if err != nil {
		return err
	}
	r := resp.Record

	printSection("Summary")
	stub := ""
	if r.Stub {
		stub = " [STUB — fill via records edit]"
	}
	fmt.Printf("  %s  [%s/%s]%s\n", r.ID, r.Kind, r.Scenario, stub)

	printSection("Narrative")
	fmt.Printf("  Trigger:  %s\n", emptyDash(r.Trigger))
	fmt.Printf("  Approach: %s\n", emptyDash(r.Approach))
	if len(r.RuledOut) > 0 {
		fmt.Println("  Ruled out:")
		for _, ro := range r.RuledOut {
			fmt.Printf("    - %s\n", ro)
		}
	}
	if r.Evidence != "" {
		fmt.Printf("  Evidence: %s\n", r.Evidence)
	}
	fmt.Printf("  Outcome:  %s\n", r.Outcome)

	printSection("Provenance")
	if r.BacklogRef != "" {
		fmt.Printf("  Backlog: %s\n", r.BacklogRef)
	}
	if r.Commit != "" {
		fmt.Printf("  Commit: %s\n", r.Commit)
	}
	if len(r.FilesChanged) > 0 {
		fmt.Printf("  Files: %s\n", strings.Join(r.FilesChanged, ", "))
	}
	if r.Supersedes != "" {
		fmt.Printf("  Supersedes: %s\n", r.Supersedes)
	}
	if r.SupersededBy != "" {
		fmt.Printf("  Superseded by: %s\n", r.SupersededBy)
	}
	if r.CreatedBy != "" {
		fmt.Printf("  Created by: %s\n", r.CreatedBy)
	}
	if r.CreatedAt != "" {
		fmt.Printf("  Created at: %s\n", r.CreatedAt)
	}

	next := []string{}
	if r.Stub {
		next = append(next, cliCommand("records", "edit", "--id", r.ID,
			"--trigger", "'...'", "--approach", "'...'", "--outcome", "shipped"))
	} else if r.SupersededBy == "" {
		next = append(next, cliCommand("records", "create", "--kind", r.Kind, "--scenario", r.Scenario,
			"--supersedes", r.ID, "--trigger", "'amendment: ...'", "--approach", "'...'", "--outcome", "shipped"))
	}
	if len(next) > 0 {
		printCommandListSection("Next Steps", next)
	}
	return nil
}

func (a *App) cmdRecordsCreate(args []string) error {
	fs := flag.NewFlagSet("records create", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Record kind (idea|research|fix|execute|chore) [required]")
	scenarioFlag := fs.String("scenario", "", "Target scenario slug [required]")
	triggerFlag := fs.String("trigger", "", "One-line symptom/goal/smell [required]")
	approachFlag := fs.String("approach", "", "What was understood / built")
	var ruledOut stringSlice
	fs.Var(&ruledOut, "ruled-out", "Hypothesis considered and rejected (repeatable)")
	evidenceFlag := fs.String("evidence", "", "Validation results (test suites, baseline diffs, live checks)")
	commitFlag := fs.String("commit", "", "Commit SHA")
	var files stringSlice
	fs.Var(&files, "files", "Repo-relative file path (repeatable)")
	backlogRefFlag := fs.String("backlog-ref", "", "Backlog reference (kind/name)")
	milestoneIDFlag := fs.String("milestone-id", "", "Milestone this work belongs to (links the record back to the milestone)")
	supersedesFlag := fs.String("supersedes", "", "Record ID this record supersedes")
	outcomeFlag := fs.String("outcome", "shipped", "Outcome category (shipped|partial|abandoned|duplicate)")
	createdByFlag := fs.String("created-by", "", "Author identifier (agent id or human)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return recordsFlagError(fs, err)
	}

	in := recordsCreateInput{
		kind: *kindFlag, scenario: *scenarioFlag, trigger: *triggerFlag,
		approach: *approachFlag, evidence: *evidenceFlag, ruledOut: ruledOut,
		commit: *commitFlag, files: files, backlogRef: *backlogRefFlag,
		milestoneID: *milestoneIDFlag, supersedes: *supersedesFlag,
		outcome: *outcomeFlag, createdBy: *createdByFlag,
	}

	if extra := fs.Args(); len(extra) > 0 {
		return fmt.Errorf("unexpected positional arguments: %q\n"+
			"records create takes flag arguments only — stray words usually mean a quoting mistake in a long --trigger/--approach value, and silently dropping them would corrupt the record.\n\nTry:\n  %s",
			extra, in.suggestedCommand())
	}
	if outcomeLooksLikeProse(in.outcome) {
		prose := in.outcome
		in.outcome = ""
		if in.evidence == "" {
			in.evidence = prose
		}
		return fmt.Errorf("--outcome is a category (%s), not a summary — your validation story belongs in --evidence\n\nTry:\n  %s",
			recordsOutcomes, in.suggestedCommand())
	}
	if err := requireFlags("kind", in.kind, "scenario", in.scenario, "trigger", in.trigger); err != nil {
		return fmt.Errorf("%s\n\nTry:\n  %s\n\nRun '%s' for the full flag reference",
			err, in.suggestedCommand(), cliCommand("records", "create", "--help"))
	}

	payload, err := json.Marshal(map[string]any{
		"kind":          in.kind,
		"scenario":      in.scenario,
		"backlog_ref":   in.backlogRef,
		"milestone_id":  in.milestoneID,
		"supersedes":    in.supersedes,
		"trigger":       in.trigger,
		"approach":      in.approach,
		"ruled_out":     []string(in.ruledOut),
		"evidence":      in.evidence,
		"commit":        in.commit,
		"files_changed": []string(in.files),
		"outcome":       in.outcome,
		"created_by":    in.createdBy,
	})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	body, err := a.core.Request("POST", "/records", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[RecordEnvelope](body)
	if err != nil {
		return err
	}
	r := resp.Record

	printSection("Result")
	fmt.Printf("  Created record: %s\n", r.ID)
	fmt.Printf("  Kind/Scenario:  %s/%s\n", r.Kind, r.Scenario)
	if r.Supersedes != "" {
		fmt.Printf("  Supersedes: %s\n", r.Supersedes)
	}
	if len(resp.Warnings) > 0 {
		printSection("Warnings")
		for _, w := range resp.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("records", "get", "--id", r.ID),
		cliCommand("records", "search", fmt.Sprintf("'%s'", truncate(r.Trigger, 40))),
	})
	return nil
}

func (a *App) cmdRecordsEdit(args []string) error {
	fs := flag.NewFlagSet("records edit", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Record ID (stub or private draft) [required]")
	repairFlag := fs.Bool("repair", false, "Repair a private adaptive-capture draft instead of filling a backlog stub")
	triggerFlag := fs.String("trigger", "", "One-line symptom/goal/smell")
	approachFlag := fs.String("approach", "", "What was understood / built")
	var ruledOut stringSlice
	fs.Var(&ruledOut, "ruled-out", "Hypothesis considered and rejected (repeatable)")
	evidenceFlag := fs.String("evidence", "", "Validation results (test suites, baseline diffs, live checks)")
	commitFlag := fs.String("commit", "", "Commit SHA")
	var files stringSlice
	fs.Var(&files, "files", "Repo-relative file path (repeatable)")
	outcomeFlag := fs.String("outcome", "", "Outcome category (shipped|partial|abandoned|duplicate)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return recordsFlagError(fs, err)
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: records edit [--repair] --id ID --trigger '...' --approach '...' [--ruled-out '...']... [--evidence '...'] [--commit SHA] [--files PATH]... [--outcome %s]\n\n%s", recordsOutcomes, err)
	}
	if outcomeLooksLikeProse(*outcomeFlag) {
		return fmt.Errorf("--outcome is a category (%s), not a summary — your validation story belongs in --evidence", recordsOutcomes)
	}
	id := strings.TrimSpace(*idFlag)

	payload, err := json.Marshal(map[string]any{
		"trigger":       *triggerFlag,
		"approach":      *approachFlag,
		"ruled_out":     []string(ruledOut),
		"evidence":      *evidenceFlag,
		"commit":        *commitFlag,
		"files_changed": []string(files),
		"outcome":       *outcomeFlag,
	})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	path := "/records/" + id + "/narrative"
	if *repairFlag {
		path = "/records/" + id + "/capture"
	}
	body, err := a.core.Request("PATCH", path, nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	if *repairFlag {
		resp, err := decodeResponse[RecordCaptureResponse](body)
		if err != nil {
			return err
		}
		if resp.Disposition == "draft" {
			fmt.Printf("Private draft remains unpublished: %s\n", resp.Record.ID)
			if len(resp.Needs) > 0 {
				fmt.Printf("Needs: %s\n", strings.Join(resp.Needs, ", "))
			}
			if len(resp.NextAction) > 0 {
				fmt.Printf("Repair: %s\n", strings.Join(resp.NextAction, " "))
			}
			return nil
		}
		printSection("Result")
		fmt.Printf("  Published repaired record: %s\n", resp.Record.ID)
		return nil
	}
	resp, err := decodeResponse[RecordEnvelope](body)
	if err != nil {
		return err
	}
	r := resp.Record

	printSection("Result")
	fmt.Printf("  Filled narrative on record: %s\n", r.ID)
	printCommandListSection("Next Steps", []string{
		cliCommand("records", "get", "--id", r.ID),
	})
	return nil
}

func (a *App) cmdRecordsSearch(args []string) error {
	fs := flag.NewFlagSet("records search", flag.ContinueOnError)
	scenarioFlag := fs.String("scenario", "", "Filter by scenario slug")
	kindFlag := fs.String("kind", "", "Filter by kind")
	limitFlag := fs.Int("limit", 10, "Max results")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("usage: records search '<query>' [--kind K] [--scenario X] [--limit N] [--json]")
	}
	query := strings.Join(rest, " ")

	payload, err := json.Marshal(map[string]any{
		"query":    query,
		"kind":     *kindFlag,
		"scenario": *scenarioFlag,
		"limit":    *limitFlag,
	})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	body, err := a.core.Request("POST", "/records/search", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[RecordSearchResponse](body)
	if err != nil {
		return err
	}

	if len(resp.Hits) == 0 {
		printSection("Summary")
		fmt.Println("  No matching records.")
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d match(es) for %q\n", len(resp.Hits), query)

	printSection("Results")
	for _, h := range resp.Hits {
		r := h.Record
		preview := r.Trigger
		if preview == "" {
			preview = r.Approach
		}
		if len(preview) > 70 {
			preview = preview[:70] + "..."
		}
		fmt.Printf("  [%.2f] [%s/%s] %s  %s\n", h.Score, r.Kind, r.Scenario, r.ID, preview)
	}
	return nil
}

func (a *App) cmdRecordsSupersede(args []string) error {
	fs := flag.NewFlagSet("records supersede", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Record being superseded [required]")
	byFlag := fs.String("by", "", "Successor record ID [required]")
	reasonFlag := fs.String("reason", "", "Why this record is being superseded")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("id", *idFlag, "by", *byFlag); err != nil {
		return fmt.Errorf("usage: records supersede --id ID --by SUCCESSOR-ID [--reason '...']\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	payload, err := json.Marshal(map[string]any{
		"successor_id": *byFlag,
		"reason":       *reasonFlag,
	})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	body, err := a.core.Request("POST", "/records/"+id+"/supersede", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[RecordEnvelope](body)
	if err != nil {
		return err
	}
	r := resp.Record

	printSection("Result")
	fmt.Printf("  Marked %s as superseded by %s\n", r.ID, r.SupersededBy)
	return nil
}

// recordsOutcomes is the canonical outcome enumeration, kept in the usage
// strings so agents never have to guess what `[--outcome ...]` elides —
// transcript analysis showed that elision was the single largest source of
// failed create attempts.
const recordsOutcomes = "shipped|partial|abandoned|duplicate"

// recordsCreateInput carries the parsed create flags so validation failures
// can echo back a corrected, fully-quoted command instead of a bare usage line.
type recordsCreateInput struct {
	kind, scenario, trigger, approach, evidence string
	ruledOut, files                             stringSlice
	commit, backlogRef, milestoneID             string
	supersedes, outcome, createdBy              string
}

// suggestedCommand renders the full create command with everything the agent
// already provided (shell-quoted) and <placeholders> for missing required
// flags, ready to paste after one correction.
func (in recordsCreateInput) suggestedCommand() string {
	parts := []string{appName, "records", "create"}
	addReq := func(name, val, placeholder string) {
		if strings.TrimSpace(val) == "" {
			parts = append(parts, "--"+name, placeholder)
		} else {
			parts = append(parts, "--"+name, shellQuote(val))
		}
	}
	addOpt := func(name, val string) {
		if strings.TrimSpace(val) != "" {
			parts = append(parts, "--"+name, shellQuote(val))
		}
	}
	addReq("kind", in.kind, "<idea|research|fix|execute|chore>")
	addReq("scenario", in.scenario, "<scenario>")
	addReq("trigger", in.trigger, "'<one-line symptom/goal>'")
	addOpt("approach", in.approach)
	for _, ro := range in.ruledOut {
		parts = append(parts, "--ruled-out", shellQuote(ro))
	}
	addOpt("evidence", in.evidence)
	addOpt("commit", in.commit)
	for _, f := range in.files {
		parts = append(parts, "--files", shellQuote(f))
	}
	addOpt("backlog-ref", in.backlogRef)
	addOpt("milestone-id", in.milestoneID)
	addOpt("supersedes", in.supersedes)
	if o := strings.TrimSpace(in.outcome); o != "" && o != "shipped" {
		parts = append(parts, "--outcome", shellQuote(o))
	}
	addOpt("created-by", in.createdBy)
	return strings.Join(parts, " ")
}

// shellQuote renders s as a single bash word, so echoed commands survive
// apostrophes and $(...) in narrative text.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-' || c == '_' || c == '.' || c == '/' || c == ':' || c == '@' || c == '%' || c == '+' || c == ',' || c == '=':
		default:
			return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		}
	}
	return s
}

// outcomeLooksLikeProse mirrors the server's heuristic so misplaced narrative
// fails fast and locally — before the full payload burns a network roundtrip.
func outcomeLooksLikeProse(outcome string) bool {
	o := strings.TrimSpace(outcome)
	return len(o) > 40 || strings.ContainsAny(o, " \t\n")
}

// recordsFlagError decorates a flag-parse failure with a targeted hint.
// Transcript analysis: --text (borrowed from `captures create`) and --title
// were the two dominant invented flags on records create.
func recordsFlagError(fs *flag.FlagSet, err error) error {
	const marker = "flag provided but not defined: -"
	msg := err.Error()
	i := strings.Index(msg, marker)
	if i < 0 {
		return err
	}
	name := strings.TrimLeft(strings.TrimSpace(msg[i+len(marker):]), "-")
	hint := ""
	switch name {
	case "text":
		hint = "--text belongs to `" + appName + " captures create`; records take narrative in --trigger (symptom/goal), --approach (what was built), and --evidence (validation results)"
	case "title", "summary", "subject":
		hint = "did you mean --trigger (one-line symptom/goal)?"
	case "description", "details", "content", "body", "notes":
		hint = "did you mean --approach (what was understood/built)?"
	case "file", "files-changed", "paths":
		hint = "did you mean --files (repeatable)?"
	case "validation", "tests", "proof":
		hint = "did you mean --evidence (validation results)?"
	default:
		known := make([]string, 0, 16)
		fs.VisitAll(func(f *flag.Flag) { known = append(known, f.Name) })
		if nearest := cliutil.NearestString(name, known, 2); nearest != "" {
			hint = fmt.Sprintf("did you mean --%s?", nearest)
		}
	}
	if hint == "" {
		return err
	}
	return fmt.Errorf("%s — %s", msg, hint)
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// truncate limits s to n characters. It counts runes rather than bytes:
// slicing a byte offset splits multi-byte characters and emits replacement
// glyphs, which shows up wherever a description contains an em dash or an
// accented name.
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
