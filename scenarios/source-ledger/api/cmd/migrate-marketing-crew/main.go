// Command migrate-marketing-crew imports the durable, shared marketing-crew
// JSONL corpus into source-ledger. It is deliberately a Go command so the
// migration uses the same typed Connect contract as every other consumer.
//
// The command is content-addressed and replay-safe: every source line carries
// an import key made from its runtime, source path, and SHA-256 hash. Dry runs
// never call the ledger. Live runs never mutate the source files.
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"
	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
)

const (
	defaultScope = "team:marketing-crew"
	defaultRoot  = ".vrooli/data/vrooli/prompt-manager/teams/marketing-crew/shared"
)

type sourceSpec struct {
	Name  string
	Facet string
}

var sources = []sourceSpec{
	{Name: "handoff-history.jsonl", Facet: "handoff"},
	{Name: "knowledge.jsonl", Facet: "knowledge"},
	{Name: "audience-scans.jsonl", Facet: "audience-finding"},
	{Name: "campaign-drafts.jsonl", Facet: "campaign"},
	{Name: "decisions.jsonl", Facet: "decision"},
	{Name: "published-scenario-mentions.jsonl", Facet: "publication"},
}

type line struct {
	Source sourceSpec
	Path   string
	Body   string
	Hash   string
}

type fileReport struct {
	Path     string `json:"path"`
	Facet    string `json:"facet"`
	Lines    int    `json:"lines"`
	Checksum string `json:"checksum"`
	Skipped  bool   `json:"skipped"`
}

type report struct {
	DryRun        bool         `json:"dry_run"`
	Scope         string       `json:"scope"`
	Files         []fileReport `json:"files"`
	TotalLines    int          `json:"total_lines"`
	Imported      int          `json:"imported"`
	Existing      int          `json:"existing"`
	Skipped       int          `json:"skipped"`
	Compacted     int32        `json:"compacted,omitempty"`
	WakeItems     int          `json:"wake_items,omitempty"`
	WakeOverflow  bool         `json:"wake_overflow,omitempty"`
	DecisionMatch string       `json:"decision_match,omitempty"`
	RecallHits    int          `json:"recall_hits,omitempty"`
}

func main() {
	root := flag.String("root", "", "marketing-crew shared directory (default: ~/.vrooli/data/vrooli/prompt-manager/teams/marketing-crew/shared)")
	base := flag.String("api-base", os.Getenv("API_BASE_URL"), "source-ledger API base URL")
	scope := flag.String("scope", defaultScope, "destination scope")
	dryRun := flag.Bool("dry-run", false, "inspect and count source lines without writing")
	query := flag.String("query", "", "optional post-import recall query")
	flag.Parse()

	rootPath, err := resolveRoot(*root)
	if err != nil {
		fatal(err)
	}
	items, files, err := readSources(rootPath)
	if err != nil {
		fatal(err)
	}
	out := report{DryRun: *dryRun, Scope: *scope, Files: files, TotalLines: len(items)}
	for _, f := range files {
		if f.Skipped {
			out.Skipped += f.Lines
		}
	}
	if *dryRun {
		writeReport(out)
		return
	}
	if strings.TrimSpace(*base) == "" {
		fatal(errors.New("api-base or API_BASE_URL is required for a live import"))
	}

	ctx := context.Background()
	httpClient := &http.Client{}
	scopeClient := scopesconnect.NewScopesServiceClient(httpClient, strings.TrimRight(*base, "/"))
	journalClient := journalconnect.NewJournalServiceClient(httpClient, strings.TrimRight(*base, "/"))
	facetsClient := facetsconnect.NewFacetsServiceClient(httpClient, strings.TrimRight(*base, "/"))
	forestClient := forestconnect.NewForestServiceClient(httpClient, strings.TrimRight(*base, "/"))
	recallClient := recallconnect.NewRecallServiceClient(httpClient, strings.TrimRight(*base, "/"))

	if err := ensureScope(ctx, scopeClient, *scope); err != nil {
		fatal(err)
	}
	for _, item := range items {
		resp, err := journalClient.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{
			Body:    item.Body,
			Scope:   *scope,
			Kind:    "marketing-" + item.Source.Facet,
			FacetId: item.Source.Facet,
			ImportProvenance: &journalv1.ImportProvenance{
				Runtime:       "prompt-manager",
				SourceLocator: item.Path,
				ContentHash:   item.Hash,
			},
		}))
		if err != nil {
			fatal(fmt.Errorf("append %s: %w", item.Path, err))
		}
		if resp.Msg.GetExisting() {
			out.Existing++
		} else {
			out.Imported++
		}
		// Assignment is append-only and also repairs a replay performed by an
		// older server that classified an explicit source facet. The source
		// path remains the authority; the classifier never chooses this facet.
		if _, err := facetsClient.AssignFacet(ctx, connect.NewRequest(&facetsv1.AssignFacetRequest{
			EntryId: resp.Msg.GetEntry().GetId(), FacetId: item.Source.Facet, Scope: *scope,
		})); err != nil {
			fatal(fmt.Errorf("assign %s facet %s: %w", item.Path, item.Source.Facet, err))
		}
	}
	compact, err := forestClient.RunCompactionPass(ctx, connect.NewRequest(&forestv1.RunCompactionPassRequest{Scope: *scope}))
	if err != nil {
		fatal(fmt.Errorf("compact %s: %w", *scope, err))
	}
	out.Compacted = compact.Msg.GetCompactedCount()
	wake, err := recallClient.Wake(ctx, connect.NewRequest(&recallv1.WakeRequest{Scope: *scope}))
	if err != nil {
		fatal(fmt.Errorf("wake %s: %w", *scope, err))
	}
	out.WakeItems = len(wake.Msg.GetHits())
	out.WakeOverflow = wake.Msg.GetOverflow()
	if strings.TrimSpace(*query) != "" {
		recalled, err := recallClient.Recall(ctx, connect.NewRequest(&recallv1.RecallRequest{Scope: *scope, Query: *query, Limit: 10}))
		if err != nil {
			fatal(fmt.Errorf("recall %s: %w", *scope, err))
		}
		out.RecallHits = len(recalled.Msg.GetHits())
	}
	for _, item := range items {
		if item.Source.Facet == "decision" && strings.Contains(item.Body, "2026-04") {
			out.DecisionMatch = item.Hash
			break
		}
	}
	writeReport(out)
}

func resolveRoot(raw string) (string, error) {
	if strings.TrimSpace(raw) != "" {
		return filepath.Abs(raw)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultRoot), nil
}

func readSources(root string) ([]line, []fileReport, error) {
	var items []line
	files := make([]fileReport, 0, len(sources))
	for _, source := range sources {
		path := filepath.Join(root, source.Name)
		report := fileReport{Path: path, Facet: source.Facet}
		file, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				report.Skipped = true
				files = append(files, report)
				continue
			}
			return nil, nil, err
		}
		hash := sha256.New()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
		for scanner.Scan() {
			raw := scanner.Text()
			if strings.TrimSpace(raw) == "" {
				continue
			}
			var value json.RawMessage
			if err := json.Unmarshal([]byte(raw), &value); err != nil {
				_ = file.Close()
				return nil, nil, fmt.Errorf("%s line %d is not JSON: %w", path, report.Lines+1, err)
			}
			_, _ = hash.Write([]byte(raw))
			_, _ = hash.Write([]byte{'\n'})
			lineHash := sha256.Sum256([]byte(raw))
			items = append(items, line{Source: source, Path: path, Body: raw, Hash: hex.EncodeToString(lineHash[:])})
			report.Lines++
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return nil, nil, err
		}
		if err := file.Close(); err != nil {
			return nil, nil, err
		}
		report.Checksum = hex.EncodeToString(hash.Sum(nil))
		files = append(files, report)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return items, files, nil
}

func ensureScope(ctx context.Context, client scopesconnect.ScopesServiceClient, id string) error {
	listed, err := client.ListScopes(ctx, connect.NewRequest(&scopesv1.ListScopesRequest{}))
	if err != nil {
		return fmt.Errorf("list scopes: %w", err)
	}
	for _, item := range listed.Msg.GetScopes() {
		if item.GetId() == id {
			return nil
		}
	}
	_, err = client.CreateScope(ctx, connect.NewRequest(&scopesv1.CreateScopeRequest{Scope: &scopesv1.Scope{
		Id: id, Label: "Marketing Crew", FrontierTarget: 32, WakeBudget: 256, MaxEntryLines: 4,
		Facets: []*scopesv1.FacetSpec{
			{Id: "handoff", Label: "Handoff", Guidance: "Resolved and active work handoffs", RetentionPolicy: "expire_on_resolution", CompactionEligible: true, ResidentBudget: 8},
			{Id: "knowledge", Label: "Knowledge", Guidance: "Durable marketing knowledge", RetentionPolicy: "retain", CompactionEligible: true, ResidentBudget: 8},
			{Id: "audience-finding", Label: "Audience finding", Guidance: "Observed audience evidence", RetentionPolicy: "retain", CompactionEligible: true, ResidentBudget: 8},
			{Id: "campaign", Label: "Campaign", Guidance: "Campaign drafts and experiments", RetentionPolicy: "retain", CompactionEligible: true, ResidentBudget: 8},
			{Id: "decision", Label: "Decision", Guidance: "Marketing decisions and their rationale", RetentionPolicy: "pin_or_review", CompactionEligible: false, ResidentBudget: 16},
			{Id: "publication", Label: "Publication", Guidance: "Published scenario mentions", RetentionPolicy: "retain", CompactionEligible: true, ResidentBudget: 4},
		},
	}}))
	if err != nil {
		return fmt.Errorf("create scope %s: %w", id, err)
	}
	return nil
}

func writeReport(value report) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
