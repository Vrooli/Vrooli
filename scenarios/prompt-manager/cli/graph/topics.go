// Topics + drain-status subcommands for `prompt-manager graph`.
//
// DOC: docs/agent-system/drafts/topics-schema.md
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Mirrors api/memberflow types. Kept local to avoid coupling the CLI to the
// API package directly.

type topicNode struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
	Ref   struct {
		Team   string `json:"team"`
		Member string `json:"member"`
	} `json:"ref,omitempty"`
}

type topicEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Prefix string `json:"prefix"`
	Kind   string `json:"kind"`
}

type topicMemberRef struct {
	Team   string `json:"team"`
	Member string `json:"member"`
}

type topicFinding struct {
	Rule     string         `json:"rule"`
	Severity string         `json:"severity"`
	Member   topicMemberRef `json:"member,omitempty"`
	Prefix   string         `json:"prefix,omitempty"`
	Detail   string         `json:"detail"`
}

type topicValidation struct {
	Findings []topicFinding `json:"findings"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
}

type topicsGraphResponse struct {
	Nodes      []topicNode     `json:"nodes"`
	Edges      []topicEdge     `json:"edges"`
	Validation topicValidation `json:"validation"`
}

// cmdTopics handles `prompt-manager graph topics`. Default output is
// human-readable; --json available for programmatic consumers. Exit code 1
// when any error-severity finding is present.
func cmdTopics(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topics", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := url.Values{}
	if *team != "" {
		q.Set("team", *team)
	}
	var resp topicsGraphResponse
	if err := ctx.GetWithQuery("/topics/graph", q, &resp); err != nil {
		return fmt.Errorf("failed to fetch topics graph: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(resp)
	} else {
		printTopicsHuman(resp, *team)
	}

	if resp.Validation.Errors > 0 {
		// Returning a non-nil error makes the cliapp dispatcher exit
		// with a non-zero status, matching the contract in
		// docs/agent-system/drafts/topics-schema.md.
		return fmt.Errorf("topics validation failed: %d error(s)", resp.Validation.Errors)
	}
	return nil
}

func printTopicsHuman(resp topicsGraphResponse, team string) {
	if team == "" {
		fmt.Println("Topic Flow Graph (all teams)")
	} else {
		fmt.Printf("Topic Flow Graph (team=%s)\n", team)
	}
	fmt.Println()

	// Summary counts
	memberCount := 0
	for _, n := range resp.Nodes {
		if n.Kind == "member" {
			memberCount++
		}
	}
	fmt.Printf("Members:  %d\n", memberCount)
	fmt.Printf("Nodes:    %d\n", len(resp.Nodes))
	fmt.Printf("Edges:    %d\n", len(resp.Edges))
	fmt.Println()

	// Group edges by source member for readability.
	byFrom := make(map[string][]topicEdge)
	for _, e := range resp.Edges {
		byFrom[e.From] = append(byFrom[e.From], e)
	}

	// Build node lookup
	nodeByID := make(map[string]topicNode)
	for _, n := range resp.Nodes {
		nodeByID[n.ID] = n
	}

	// Print member nodes with their edges
	memberIDs := make([]string, 0, len(resp.Nodes))
	for _, n := range resp.Nodes {
		if n.Kind == "member" {
			memberIDs = append(memberIDs, n.ID)
		}
	}
	sort.Strings(memberIDs)

	for _, id := range memberIDs {
		n := nodeByID[id]
		fmt.Printf("  %s/%s\n", n.Ref.Team, n.Ref.Member)
		// Outgoing edges
		edges := byFrom[id]
		sort.Slice(edges, func(i, j int) bool { return edges[i].Prefix < edges[j].Prefix })
		for _, e := range edges {
			dest := nodeByID[e.To]
			label := dest.Label
			if label == "" {
				label = dest.ID
			}
			fmt.Printf("    %s -> %s (%s)\n", e.Prefix, label, e.Kind)
		}
		// Incoming intake edges (group separately)
		var incoming []topicEdge
		for _, e := range resp.Edges {
			if e.To == id && (e.Kind == "intake" || e.Kind == "external_producer" || e.Kind == "decision_consumed") {
				incoming = append(incoming, e)
			}
		}
		sort.Slice(incoming, func(i, j int) bool {
			if incoming[i].Kind != incoming[j].Kind {
				return incoming[i].Kind < incoming[j].Kind
			}
			return incoming[i].Prefix < incoming[j].Prefix
		})
		for _, e := range incoming {
			src := nodeByID[e.From]
			label := src.Label
			if label == "" {
				label = src.ID
			}
			fmt.Printf("    <- %s (%s) from %s\n", e.Prefix, e.Kind, label)
		}
	}

	// Validation
	v := resp.Validation
	fmt.Println()
	if v.Errors == 0 && v.Warnings == 0 {
		fmt.Println("Validation: clean")
		return
	}
	fmt.Printf("Validation: %d error(s), %d warning(s)\n", v.Errors, v.Warnings)
	for _, f := range v.Findings {
		fmt.Printf("  [%s] %s  %s/%s  %s\n", strings.ToUpper(string(f.Severity[0])), f.Rule, f.Member.Team, f.Member.Member, f.Detail)
	}
}

type drainStatusEntry struct {
	Member          topicMemberRef `json:"member"`
	Prefix          string         `json:"prefix"`
	UnroutedCount   int            `json:"unrouted_count"`
	OldestAt        string         `json:"oldest_at,omitempty"`
	OldestAgeSecs   int64          `json:"oldest_age_seconds,omitempty"`
}

type drainStatusResponse struct {
	Entries []drainStatusEntry `json:"entries"`
	Note    string             `json:"note,omitempty"`
}

// cmdDrainStatus handles `prompt-manager graph drain-status`.
//
// Returns per-intake-prefix queue depth + oldest-entry age for every member's
// declared intake (or just one team when --team is set). The API returns a
// `note` instead of `entries` when the knowledge backend is not wired in.
func cmdDrainStatus(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("drain-status", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := url.Values{}
	if *team != "" {
		q.Set("team", *team)
	}
	var resp drainStatusResponse
	if err := ctx.GetWithQuery("/topics/drain-status", q, &resp); err != nil {
		return fmt.Errorf("failed to fetch drain-status: %w", err)
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if resp.Note != "" {
		fmt.Println(resp.Note)
	}
	if len(resp.Entries) == 0 {
		fmt.Println("(no drain-status entries)")
		return nil
	}

	if *team == "" {
		fmt.Println("Drain Status (all teams)")
	} else {
		fmt.Printf("Drain Status (team=%s)\n", *team)
	}
	fmt.Println()

	// Sort entries: team, then member, then prefix.
	sort.Slice(resp.Entries, func(i, j int) bool {
		a, b := resp.Entries[i], resp.Entries[j]
		if a.Member.Team != b.Member.Team {
			return a.Member.Team < b.Member.Team
		}
		if a.Member.Member != b.Member.Member {
			return a.Member.Member < b.Member.Member
		}
		return a.Prefix < b.Prefix
	})

	currentMember := ""
	for _, e := range resp.Entries {
		mid := e.Member.Team + "/" + e.Member.Member
		if mid != currentMember {
			fmt.Println()
			fmt.Printf("  %s\n", mid)
			currentMember = mid
		}
		age := ""
		if e.OldestAgeSecs > 0 {
			age = fmt.Sprintf(", oldest %s", formatAge(e.OldestAgeSecs))
		}
		fmt.Printf("    %-40s  unrouted=%d%s\n", e.Prefix, e.UnroutedCount, age)
	}
	return nil
}

func formatAge(seconds int64) string {
	switch {
	case seconds < 3600:
		return fmt.Sprintf("%dm", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%dh", seconds/3600)
	default:
		return fmt.Sprintf("%dd", seconds/86400)
	}
}
