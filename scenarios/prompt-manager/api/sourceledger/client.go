// Package sourceledger is the prompt-manager boundary to the shared durable
// team corpus. Runtime team memory is owned by source-ledger; prompt-manager
// only supplies a scope and translates the wire response into its heartbeat
// view models.
package sourceledger

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
)

const DefaultScope = "agent-memory"

// TeamScopeFacets returns the stable facet vocabulary shared by the
// prompt-manager teams. Facet ids are globally keyed by source-ledger, so the
// team id is part of each id even though the semantic labels stay consistent.
func TeamScopeFacets(teamID string) []*scopesv1.FacetSpec {
	teamID = strings.TrimPrefix(strings.TrimSpace(teamID), "team:")
	return []*scopesv1.FacetSpec{
		{Id: "prompt-manager-" + teamID + "-knowledge", Label: "Team knowledge", Guidance: "Durable team context and operating lessons", CompactionEligible: true, ResidentBudget: 32},
		{Id: "prompt-manager-" + teamID + "-handoff", Label: "Team handoff", Guidance: "Member handoff context", CompactionEligible: true, ResidentBudget: 16},
		{Id: "prompt-manager-" + teamID + "-work", Label: "Team work", Guidance: "Work context and evidence", CompactionEligible: true, ResidentBudget: 16},
	}
}

// UnavailableError identifies the typed dependency failure returned when the
// shared source-ledger cannot be resolved or used. Callers can classify this
// without matching fragile transport text.
type UnavailableError struct {
	Operation string
	Err       error
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("source-ledger unavailable during %s: %v", e.Operation, e.Err)
}

func (e *UnavailableError) Unwrap() error { return e.Err }

type Entry struct {
	ID, Body, FacetID, Kind, CreatedAt string
}

// WakeResult preserves the Source Ledger's bounded-view signal at the
// prompt-manager boundary. Entries are the view; the remaining fields explain
// whether the view is complete and which ceilings governed it.
type WakeResult struct {
	Entries     []Entry
	Overflow    bool
	Refused     int
	LinesUsed   int
	CharsUsed   int
	BudgetLines int
	BudgetChars int
}

type Client struct {
	Journal       journalconnect.JournalServiceClient
	RecallService recallconnect.RecallServiceClient
	Scopes        scopesconnect.ScopesServiceClient
}

func New(ctx context.Context) (*Client, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "source-ledger")
	if err != nil {
		return nil, &UnavailableError{Operation: "endpoint discovery", Err: err}
	}
	return NewAt(base), nil
}

func NewAt(base string) *Client {
	base = strings.TrimRight(base, "/")
	httpClient := http.DefaultClient
	return &Client{
		Journal:       journalconnect.NewJournalServiceClient(httpClient, base),
		RecallService: recallconnect.NewRecallServiceClient(httpClient, base),
		Scopes:        scopesconnect.NewScopesServiceClient(httpClient, base),
	}
}

func (c *Client) EnsureScope(ctx context.Context, id, label string, facets []*scopesv1.FacetSpec) error {
	response, err := c.Scopes.ListScopes(ctx, connect.NewRequest(&scopesv1.ListScopesRequest{}))
	if err != nil {
		return &UnavailableError{Operation: "list scopes", Err: err}
	}
	for _, scope := range response.Msg.GetScopes() {
		if scope.GetId() == id {
			return nil
		}
	}
	_, err = c.Scopes.CreateScope(ctx, connect.NewRequest(&scopesv1.CreateScopeRequest{Scope: &scopesv1.Scope{Id: id, Label: label, FrontierTarget: 16, WakeBudget: 128, MaxEntryLines: 2, Facets: facets}}))
	if err != nil {
		return &UnavailableError{Operation: fmt.Sprintf("create scope %q", id), Err: err}
	}
	return nil
}

// EnsureTeamScope provisions the scope required before a team member can
// build a heartbeat. It is intentionally idempotent and leaves unavailability
// as the typed UnavailableError instead of permitting a local fallback.
func (c *Client) EnsureTeamScope(ctx context.Context, teamID string) error {
	teamID = strings.TrimPrefix(strings.TrimSpace(teamID), "team:")
	return c.EnsureScope(ctx, "team:"+teamID, "prompt-manager team "+teamID, TeamScopeFacets(teamID))
}

func (c *Client) Append(ctx context.Context, scope, body, kind string) (Entry, error) {
	response, err := c.Journal.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Scope: scope, Body: body, Kind: kind}))
	if err != nil {
		return Entry{}, fmt.Errorf("append source-ledger entry: %w", err)
	}
	return fromProto(response.Msg.GetEntry()), nil
}

func (c *Client) List(ctx context.Context, scope string, limit int) ([]Entry, error) {
	response, err := c.Journal.ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{Scope: scope, Limit: int32(limit)}))
	if err != nil {
		return nil, fmt.Errorf("list source-ledger entries: %w", err)
	}
	entries := make([]Entry, 0, len(response.Msg.GetEntries()))
	for _, entry := range response.Msg.GetEntries() {
		entries = append(entries, fromProto(entry))
	}
	return entries, nil
}

func (c *Client) Recall(ctx context.Context, scope, query string, limit int) ([]Entry, error) {
	response, err := c.RecallService.Recall(ctx, connect.NewRequest(&recallv1.RecallRequest{Scope: scope, Query: query, Limit: int32(limit)}))
	if err != nil {
		return nil, fmt.Errorf("recall source-ledger entries: %w", err)
	}
	entries := make([]Entry, 0, len(response.Msg.GetHits()))
	for _, hit := range response.Msg.GetHits() {
		entries = append(entries, Entry{ID: hit.GetEntryId(), Body: hit.GetText(), FacetID: hit.GetFacetId()})
	}
	return entries, nil
}

func (c *Client) Wake(ctx context.Context, scope string, budget int) (WakeResult, error) {
	response, err := c.RecallService.Wake(ctx, connect.NewRequest(&recallv1.WakeRequest{Scope: scope, LineBudget: int32(budget)}))
	if err != nil {
		return WakeResult{}, fmt.Errorf("wake source-ledger scope: %w", err)
	}
	entries := make([]Entry, 0, len(response.Msg.GetHits()))
	for _, hit := range response.Msg.GetHits() {
		entries = append(entries, Entry{ID: hit.GetEntryId(), Body: hit.GetText(), FacetID: hit.GetFacetId()})
	}
	return WakeResult{
		Entries: entries, Overflow: response.Msg.GetOverflow(), Refused: int(response.Msg.GetRefused()),
		LinesUsed: int(response.Msg.GetLinesUsed()), CharsUsed: int(response.Msg.GetCharsUsed()),
		BudgetLines: int(response.Msg.GetBudgetLines()), BudgetChars: int(response.Msg.GetBudgetChars()),
	}, nil
}

func fromProto(entry *journalv1.Entry) Entry {
	if entry == nil {
		return Entry{}
	}
	created := ""
	if entry.GetCreatedAt() != nil {
		created = entry.GetCreatedAt().AsTime().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return Entry{ID: entry.GetId(), Body: entry.GetBody(), FacetID: entry.GetFacetId(), Kind: entry.GetKind(), CreatedAt: created}
}
