// Package sourceledger is Swarm Manager's client boundary to the shared
// durable Source Ledger. It owns no memory semantics; it only translates the
// Source Ledger Connect API into the small operations sessions need.
package sourceledger

import (
	"context"
	"encoding/json"
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

const DefaultWakeBudget = 128

const EntryKindSessionResolution = "session-resolution"

type Resolution struct {
	Scope        string `json:"-"`
	Decision     string `json:"decision"`
	Reason       string `json:"reason"`
	SessionID    string `json:"session_id"`
	ProposalID   string `json:"proposal_id"`
	ProposalKind string `json:"proposal_kind"`
}

func SessionScopeFacets(scopeID string) []*scopesv1.FacetSpec {
	clean := strings.NewReplacer(":", "-", "_", "-").Replace(strings.TrimSpace(scopeID))
	return []*scopesv1.FacetSpec{
		{Id: clean + "-knowledge", Label: "Session knowledge", Guidance: "Durable session context and operating lessons", CompactionEligible: true, ResidentBudget: 32},
		{Id: clean + "-evidence", Label: "Session evidence", Guidance: "Evidence supporting a session decision", CompactionEligible: true, ResidentBudget: 16},
	}
}

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
	_, err = c.Scopes.CreateScope(ctx, connect.NewRequest(&scopesv1.CreateScopeRequest{Scope: &scopesv1.Scope{
		Id: id, Label: label, FrontierTarget: 16, WakeBudget: DefaultWakeBudget, MaxEntryLines: 2, Facets: facets,
	}}))
	if err != nil {
		return &UnavailableError{Operation: fmt.Sprintf("create scope %q", id), Err: err}
	}
	return nil
}

func (c *Client) Append(ctx context.Context, scope, body, kind string) (Entry, error) {
	response, err := c.Journal.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Scope: scope, Body: body, Kind: kind}))
	if err != nil {
		return Entry{}, fmt.Errorf("append source-ledger entry: %w", err)
	}
	return fromProto(response.Msg.GetEntry()), nil
}

func (c *Client) WriteResolution(ctx context.Context, resolution Resolution) error {
	payload, err := json.Marshal(resolution)
	if err != nil {
		return fmt.Errorf("encode session resolution: %w", err)
	}
	if _, err := c.Append(ctx, strings.TrimSpace(resolution.Scope), string(payload), EntryKindSessionResolution); err != nil {
		return fmt.Errorf("write session resolution: %w", err)
	}
	return nil
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
