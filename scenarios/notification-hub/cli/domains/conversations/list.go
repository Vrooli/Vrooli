package conversations

// `conversations list`: which asks, in what order, and what the operator's
// inbox looks like by urgency and sender.

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// listSorts are the values of --sort, in the order the help names them.
var listSorts = []struct {
	name  string
	order v1.ListAsksRequest_Order
}{
	{"attention", v1.ListAsksRequest_ORDER_ATTENTION},
	{"due", v1.ListAsksRequest_ORDER_DUE},
	{"newest", v1.ListAsksRequest_ORDER_NEWEST},
	{"oldest", v1.ListAsksRequest_ORDER_OLDEST},
}

// listStates are the values of --state: the Hub's state_filter.
var listStates = []string{"all", "awaiting", "pending", "escalated", "answered", "defaulted", "expired", "withdrawn", "dismissed"}

// listUrgencies are the values of --urgency.
var listUrgencies = []string{"low", "normal", "high", "critical"}

// listFilters is what the flags of `list` ask the Hub for.
type listFilters struct {
	order     v1.ListAsksRequest_Order
	state     string
	urgencies []string
	senders   []string
	limit     int32
	cursor    string
	query     string
}

func (s *service) list(ctx cliapp.RunContext) error {
	filters, err := parseListFilters(ctx)
	if err != nil {
		return err
	}
	resp, callErr := s.client.ListAsks(context.Background(), connect.NewRequest(filters.request()))
	if callErr != nil {
		return readError("list asks", callErr)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, listReport(resp.Msg))
}

func parseListFilters(ctx cliapp.RunContext) (listFilters, error) {
	var f listFilters
	var err error
	if f.limit, err = parsePageLimit(ctx.Flag("limit")); err != nil {
		return f, err
	}
	f.cursor = ctx.Flag("cursor")
	f.query = ctx.Flag("query")
	if f.order, err = parseSort(ctx.Flag("sort")); err != nil {
		return f, err
	}
	if f.state, err = parseState(ctx.Flag("state"), ctx.BoolFlag("open")); err != nil {
		return f, err
	}
	if f.urgencies, err = parseUrgencies(ctx.FlagValues("urgency")); err != nil {
		return f, err
	}
	if f.senders, err = parseSenders(ctx.FlagValues("sender")); err != nil {
		return f, err
	}
	return f, nil
}

func (f listFilters) request() *v1.ListAsksRequest {
	return &v1.ListAsksRequest{
		Limit:       f.limit,
		Cursor:      f.cursor,
		Query:       f.query,
		StateFilter: f.state,
		Order:       f.order,
		Urgencies:   f.urgencies,
		Senders:     f.senders,
	}
}

// parseSort reads --sort. Empty is the Hub's default, newest first.
func parseSort(value string) (v1.ListAsksRequest_Order, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return v1.ListAsksRequest_ORDER_UNSPECIFIED, nil
	}
	names := make([]string, 0, len(listSorts))
	for _, sort := range listSorts {
		if sort.name == value {
			return sort.order, nil
		}
		names = append(names, sort.name)
	}
	return 0, fmt.Errorf("--sort must be one of %s, got %q", strings.Join(names, ", "), value)
}

// parseState reads --state and its shortcut --open, which means --state
// awaiting. Naming both is fine when they agree and an error when they do not.
func parseState(value string, open bool) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "" && !contains(listStates, value) {
		return "", fmt.Errorf("--state must be one of %s, got %q", strings.Join(listStates, ", "), value)
	}
	switch {
	case !open:
		return value, nil
	case value == "" || value == "awaiting":
		return "awaiting", nil
	default:
		return "", fmt.Errorf("--open means --state awaiting, which conflicts with --state %s; drop one of them", value)
	}
}

// parseUrgencies reads every --urgency value, each of which may itself be a
// comma-separated list.
func parseUrgencies(values []string) ([]string, error) {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.ToLower(strings.TrimSpace(part))
			if !contains(listUrgencies, part) {
				return nil, fmt.Errorf("--urgency must be one or more of %s, got %q", strings.Join(listUrgencies, ", "), part)
			}
			if !contains(out, part) {
				out = append(out, part)
			}
		}
	}
	return out, nil
}

// parseSenders reads every --sender value. A sender is a name the sender
// chose, so it is matched whole and never split on commas; repeat the flag for
// several. The names to use are the ones the list's facets print.
func parseSenders(values []string) ([]string, error) {
	var out []string
	for _, value := range values {
		name := strings.Join(strings.Fields(value), " ")
		if name == "" {
			return nil, fmt.Errorf("--sender needs a sender name; a list prints the names it holds under \"By sender\" (or facets.senders with --json)")
		}
		if !contains(out, name) {
			out = append(out, name)
		}
	}
	return out, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// listReport is the text of a list: one row per ask, then what else the
// operator's inbox holds by urgency and sender.
func listReport(resp *v1.ListAsksResponse) cliapp.ListReport {
	lines := make([]string, 0, len(resp.GetAsks()))
	for _, ask := range resp.GetAsks() {
		lines = append(lines, askRow(ask))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d asks", len(lines))},
		ResultsHeading: "Asks",
		Results:        lines,
	}
	if facets := facetSummary(resp.GetFacets()); facets != "" {
		report.Summary = append(report.Summary, facets)
	}
	if resp.GetNextCursor() != "" {
		report.RetrievalHints = []string{"More asks follow: run the same command with --cursor " + resp.GetNextCursor()}
	}
	return report
}

// facetSummary is the facet counts as one line, "high 2, normal 5 | Research
// Desk 4, Agent run 3". It is empty when the Hub sent none.
func facetSummary(facets *v1.AskFacets) string {
	urgencies := facetCounts(facets.GetUrgencies())
	senders := facetCounts(facets.GetSenders())
	var parts []string
	if urgencies != "" {
		parts = append(parts, "By urgency: "+urgencies)
	}
	if senders != "" {
		parts = append(parts, "By sender: "+senders)
	}
	return strings.Join(parts, ". ")
}

func facetCounts(counts []*v1.AskFacetCount) string {
	parts := make([]string, 0, len(counts))
	for _, count := range counts {
		parts = append(parts, fmt.Sprintf("%s %d", count.GetValue(), count.GetCount()))
	}
	return strings.Join(parts, ", ")
}
