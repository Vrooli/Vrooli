package journal

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type handlers struct {
	client journalLister
}

type journalLister interface {
	ListEntries(context.Context, *connect.Request[journalv1.ListEntriesRequest]) (*connect.Response[journalv1.ListEntriesResponse], error)
	GetEntry(context.Context, *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error)
	AppendEntry(context.Context, *connect.Request[journalv1.AppendEntryRequest]) (*connect.Response[journalv1.AppendEntryResponse], error)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: journalconnect.NewJournalServiceClient(http, base)}
}

func (h *handlers) noteCall(ctx cliapp.OperationContext) (*journalv1.AppendEntryResponse, error) {
	response, err := h.client.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: ctx.Positional("body"), Scope: ctx.Flag("scope"), Kind: ctx.Flag("kind"), Trigger: ctx.Flag("trigger"), Approach: ctx.Flag("approach"), Evidence: ctx.Flag("evidence"), Outcome: ctx.Flag("outcome")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("append source-ledger entry", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.Entry == nil {
		return nil, fmt.Errorf("server returned no source-ledger entry")
	}
	return response.Msg, nil
}

func (h *handlers) noteReport(_ cliapp.OperationContext, msg *journalv1.AppendEntryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %s.", msg.GetEntry().GetId())}, Changes: []string{fmt.Sprintf("[%s] %s", msg.GetEntry().GetFacetId(), msg.GetEntry().GetBody())}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*journalv1.ListEntriesResponse, error) {
	limit, err := strconv.Atoi(ctx.Flag("limit"))
	if err != nil || limit < 1 || limit > 500 {
		return nil, fmt.Errorf("limit must be 1-500")
	}
	newest := ctx.BoolFlag("newest-first")
	if ctx.Flag("kind") != "" && !newest {
		return nil, fmt.Errorf("kind requires --newest-first")
	}
	if newest && ctx.Flag("cursor") != "" {
		return nil, fmt.Errorf("newest-first does not accept a cursor")
	}
	response, err := h.client.ListEntries(context.Background(), connect.NewRequest(&journalv1.ListEntriesRequest{Scope: ctx.Flag("scope"), Limit: int32(limit), NewestFirst: newest, Kind: ctx.Flag("kind"), Cursor: ctx.Flag("cursor")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list journal", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *journalv1.ListEntriesResponse) cliapp.ListReport {
	lines := []string{}
	for _, e := range msg.GetEntries() {
		lines = append(lines, fmt.Sprintf("%s [%s] %s", e.GetId(), e.GetKind(), e.GetBody()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d journal entries", len(lines))}, Results: lines}
}

const exportPageSize = 500

type journalExport struct {
	Source            string            `json:"source"`
	Scope             string            `json:"scope"`
	SnapshotAt        string            `json:"snapshot_at"`
	Complete          bool              `json:"complete"`
	Degraded          bool              `json:"degraded"`
	DegradationReason string            `json:"degradation_reason,omitempty"`
	EntryCount        int               `json:"entry_count"`
	PageCount         int               `json:"page_count"`
	Entries           []json.RawMessage `json:"entries"`
}

func (h *handlers) export(ctx context.Context, scope string, pageSize int, snapshotTime time.Time) (out journalExport) {
	out = journalExport{
		Source:     "source-ledger.journal.list",
		Scope:      scope,
		SnapshotAt: snapshotTime.UTC().Format(time.RFC3339Nano),
		Entries:    []json.RawMessage{},
	}
	defer func() { out.EntryCount = len(out.Entries) }()
	seenCursors := map[string]struct{}{}
	seenIDs := map[string]struct{}{}
	cursor := ""
	for {
		response, err := h.client.ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{
			Scope:  scope,
			Limit:  int32(pageSize),
			Cursor: cursor,
		}))
		if err != nil {
			out.Degraded = true
			out.DegradationReason = fmt.Sprintf("page %d failed: %v", out.PageCount+1, err)
			return out
		}
		if response == nil || response.Msg == nil {
			out.Degraded = true
			out.DegradationReason = fmt.Sprintf("page %d returned no response", out.PageCount+1)
			return out
		}
		page := response.Msg
		out.PageCount++
		for _, entry := range page.GetEntries() {
			if entry == nil || entry.GetId() == "" {
				out.Degraded = true
				out.DegradationReason = fmt.Sprintf("page %d contained an entry without an ID", out.PageCount)
				return out
			}
			if _, exists := seenIDs[entry.GetId()]; exists {
				out.Degraded = true
				out.DegradationReason = fmt.Sprintf("page %d repeated entry ID %q", out.PageCount, entry.GetId())
				return out
			}
			encoded, err := protojson.Marshal(entry)
			if err != nil {
				out.Degraded = true
				out.DegradationReason = fmt.Sprintf("page %d entry %q could not be encoded: %v", out.PageCount, entry.GetId(), err)
				return out
			}
			seenIDs[entry.GetId()] = struct{}{}
			out.Entries = append(out.Entries, encoded)
		}
		if next := page.GetNextCursor(); next == "" {
			out.Complete = true
			out.EntryCount = len(out.Entries)
			return out
		} else {
			if next == cursor {
				out.Degraded = true
				out.DegradationReason = fmt.Sprintf("page %d returned a non-advancing cursor", out.PageCount)
				return out
			}
			if _, exists := seenCursors[next]; exists {
				out.Degraded = true
				out.DegradationReason = fmt.Sprintf("page %d returned a previously seen cursor", out.PageCount)
				return out
			}
			seenCursors[next] = struct{}{}
			cursor = next
		}
	}
}

func (h *handlers) exportRun(ctx cliapp.RunContext) error {
	result := h.export(context.Background(), ctx.Flag("scope"), exportPageSize, time.Now())
	if err := json.NewEncoder(ctx.Stdout()).Encode(result); err != nil {
		return fmt.Errorf("write journal export: %w", err)
	}
	if result.Degraded {
		return fmt.Errorf("journal export incomplete: %s", result.DegradationReason)
	}
	return nil
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*journalv1.GetEntryResponse, error) {
	response, err := h.client.GetEntry(context.Background(), connect.NewRequest(&journalv1.GetEntryRequest{Id: ctx.Positional("id"), Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get journal entry", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *journalv1.GetEntryResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{msg.GetEntry().GetId()}, Results: []string{msg.GetEntry().GetBody()}}
}
