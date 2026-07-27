// Topics + drain-status subcommands for `prompt-manager graph`.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"prompt-manager/cli/internal/appctx"

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
	// OwnerKey is the canonical surface owner (`team:<t>/<m>`,
	// `team:<t>`, `agent:<id>`, `skill:<id>`, `docs:<domain>`)
	// populated by Pillar 2 (`prose_topic_leak`). Other rules leave
	// it empty.
	OwnerKey string `json:"owner_key,omitempty"`
	Detail   string `json:"detail"`
	// Advisory marks a finding from a heuristic that cannot fully separate a
	// real defect from a lookalike — review material for this sweep, withheld
	// from surfaces that instruct an agent. Mirrors memberflow.Finding; the
	// field is decoded here so `--json` consumers see what the API sent.
	Advisory bool `json:"advisory,omitempty"`
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
//
// --findings-out=<path> writes a stable JSON artifact (see
// findings_artifact.go) for CI diff-against-previous-run telemetry. The
// artifact is opt-in: empty value (the default) leaves the filesystem
// untouched. CI scripts that consume the diff pass an explicit path.
//
// Artifact-write failures are surfaced as a stderr warning, not a fatal
// error: the validation result on stdout (and the exit code derived
// from it) is the primary contract; the artifact is telemetry that must
// not block CI from observing the validation outcome.
func cmdTopics(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("topics", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	findingsOut := fs.String("findings-out", "", "Write findings JSON artifact to this path (empty disables the write). CI uses this for diff-against-previous-run telemetry; see docs/agent-system/RUNTIME_ATTRIBUTION.md.")
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

	if err := writeFindingsArtifact(*findingsOut, resp, *team, time.Now()); err != nil {
		// Telemetry path: warn on stderr but do not change the
		// validation exit code. CI's diff step will notice the
		// missing/stale artifact on its own and surface a separate
		// signal.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	} else if !*jsonOut && *findingsOut != "" {
		// Echo the artifact write under human output only; --json
		// callers parse stdout strictly and get nothing extra.
		fmt.Printf("Findings artifact written to %s\n", *findingsOut)
	}

	if resp.Validation.Errors > 0 {
		// Returning a non-nil error makes the cliapp dispatcher exit
		// with a non-zero status, matching the contract in
		// docs/agent-system/TOPICS_SCHEMA.md.
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

	v := resp.Validation
	fmt.Println()
	if v.Errors == 0 && v.Warnings == 0 {
		fmt.Println("Validation: clean")
		return
	}
	fmt.Printf("Validation: %d error(s), %d warning(s)\n", v.Errors, v.Warnings)
	printTopicFindings(v.Findings)
}

const topicFindingExampleLimit = 3

// printTopicFindings presents validation findings by remediation cause rather
// than one unbounded line per finding. JSON output stays lossless; this is the
// concise operator-facing view.
func printTopicFindings(findings []topicFinding) {
	byRule := make(map[string][]topicFinding)
	for _, finding := range findings {
		byRule[finding.Rule] = append(byRule[finding.Rule], finding)
	}

	rules := make([]string, 0, len(byRule))
	for rule := range byRule {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		left, right := byRule[rules[i]], byRule[rules[j]]
		if topicSeverityRank(left[0].Severity) != topicSeverityRank(right[0].Severity) {
			return topicSeverityRank(left[0].Severity) < topicSeverityRank(right[0].Severity)
		}
		return rules[i] < rules[j]
	})

	fmt.Println("Finding summary:")
	for _, rule := range rules {
		group := byRule[rule]
		fmt.Printf("  %s [%s]: %d\n", rule, strings.ToUpper(group[0].Severity), len(group))
	}

	fmt.Println("Findings:")
	for _, rule := range rules {
		group := byRule[rule]
		sort.Slice(group, func(i, j int) bool { return topicFindingSortKey(group[i]) < topicFindingSortKey(group[j]) })
		fmt.Printf("  %s [%s] (%d):\n", rule, strings.ToUpper(group[0].Severity), len(group))
		for _, finding := range group[:min(topicFindingExampleLimit, len(group))] {
			fmt.Printf("    %s\n", formatTopicFinding(finding))
		}
		if suppressed := len(group) - topicFindingExampleLimit; suppressed > 0 {
			fmt.Printf("    ... %d more suppressed\n", suppressed)
		}
	}
}

func topicSeverityRank(severity string) int {
	if severity == "error" {
		return 0
	}
	return 1
}

func topicFindingSortKey(finding topicFinding) string {
	return strings.Join([]string{finding.Member.Team, finding.Member.Member, finding.OwnerKey, finding.Prefix, finding.Detail}, "\x00")
}

func formatTopicFinding(finding topicFinding) string {
	location := finding.OwnerKey
	if location == "" && (finding.Member.Team != "" || finding.Member.Member != "") {
		location = finding.Member.Team + "/" + finding.Member.Member
	}
	parts := make([]string, 0, 3)
	if location != "" {
		parts = append(parts, location)
	}
	if finding.Prefix != "" {
		parts = append(parts, finding.Prefix)
	}
	if finding.Detail != "" {
		parts = append(parts, finding.Detail)
	}
	return strings.Join(parts, "  ")
}

type drainStatusEntry struct {
	Member        topicMemberRef `json:"member"`
	Prefix        string         `json:"prefix"`
	UnroutedCount int            `json:"unrouted_count"`
	OldestAt      string         `json:"oldest_at,omitempty"`
	OldestAgeSecs int64          `json:"oldest_age_seconds,omitempty"`
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
