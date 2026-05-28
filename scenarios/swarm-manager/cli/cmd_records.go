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
	Supersedes   string   `json:"supersedes,omitempty"`
	SupersededBy string   `json:"superseded_by,omitempty"`
	Trigger      string   `json:"trigger"`
	Approach     string   `json:"approach"`
	RuledOut     []string `json:"ruled_out,omitempty"`
	Commit       string   `json:"commit,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
	Outcome      string   `json:"outcome"`
	Stub         bool     `json:"stub"`
	CreatedAt    string   `json:"created_at"`
	CreatedBy    string   `json:"created_by,omitempty"`
	NarrativeAt  string   `json:"narrative_at,omitempty"`
}

// RecordEnvelope wraps single-record endpoints.
type RecordEnvelope struct {
	Record Record `json:"record"`
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
	commitFlag := fs.String("commit", "", "Commit SHA")
	var files stringSlice
	fs.Var(&files, "files", "Repo-relative file path (repeatable)")
	backlogRefFlag := fs.String("backlog-ref", "", "Backlog reference (kind/name)")
	supersedesFlag := fs.String("supersedes", "", "Record ID this record supersedes")
	outcomeFlag := fs.String("outcome", "shipped", "Outcome (shipped|partial|abandoned|duplicate)")
	createdByFlag := fs.String("created-by", "", "Author identifier (agent id or human)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "scenario", *scenarioFlag, "trigger", *triggerFlag); err != nil {
		return fmt.Errorf("usage: records create --kind K --scenario X --trigger '...' [--approach '...'] [--ruled-out '...']... [--commit SHA] [--files PATH]... [--backlog-ref kind/name] [--supersedes ID] [--outcome ...] [--json]\n\n%s", err)
	}

	payload, err := json.Marshal(map[string]any{
		"kind":          *kindFlag,
		"scenario":      *scenarioFlag,
		"backlog_ref":   *backlogRefFlag,
		"supersedes":    *supersedesFlag,
		"trigger":       *triggerFlag,
		"approach":      *approachFlag,
		"ruled_out":     []string(ruledOut),
		"commit":        *commitFlag,
		"files_changed": []string(files),
		"outcome":       *outcomeFlag,
		"created_by":    *createdByFlag,
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
	printCommandListSection("Next Steps", []string{
		cliCommand("records", "get", "--id", r.ID),
		cliCommand("records", "search", fmt.Sprintf("'%s'", truncate(r.Trigger, 40))),
	})
	return nil
}

func (a *App) cmdRecordsEdit(args []string) error {
	fs := flag.NewFlagSet("records edit", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Record ID (must be a stub) [required]")
	triggerFlag := fs.String("trigger", "", "One-line symptom/goal/smell")
	approachFlag := fs.String("approach", "", "What was understood / built")
	var ruledOut stringSlice
	fs.Var(&ruledOut, "ruled-out", "Hypothesis considered and rejected (repeatable)")
	commitFlag := fs.String("commit", "", "Commit SHA")
	var files stringSlice
	fs.Var(&files, "files", "Repo-relative file path (repeatable)")
	outcomeFlag := fs.String("outcome", "", "Outcome (shipped|partial|abandoned|duplicate)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: records edit --id ID --trigger '...' --approach '...' [--ruled-out '...']... [--commit SHA] [--files PATH]... [--outcome ...]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	payload, err := json.Marshal(map[string]any{
		"trigger":       *triggerFlag,
		"approach":      *approachFlag,
		"ruled_out":     []string(ruledOut),
		"commit":        *commitFlag,
		"files_changed": []string(files),
		"outcome":       *outcomeFlag,
	})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	body, err := a.core.Request("PATCH", "/records/"+id+"/narrative", nil, json.RawMessage(payload))
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

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
