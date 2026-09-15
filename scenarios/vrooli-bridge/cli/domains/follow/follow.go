// Package follow is the CLI for branch-follow policies: set a node to follow a
// git branch so Bridge provisions it to each new commit, stop, list, and check
// now. Like readiness it wraps an owner-only REST exception.
package follow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `vrooli-bridge follow` group.
func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	node := cliapp.Flag{Name: "node", Required: true, Description: "Node id"}
	return cliapp.SubcommandGroup{Name: "follow", Description: "Keep nodes on the tip of a git branch", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Description: "Show every node following a branch and its last check", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, RunCtx: list},
		{Name: "set", Description: "Follow a branch: Bridge provisions the node to each new commit", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			node,
			{Name: "branch", Required: true, Description: "Branch to follow, e.g. agi"},
			{Name: "repo-url", Description: "Source repository (default https://github.com/Vrooli/Vrooli.git)"},
			{Name: "now", Bool: true, Description: "Also update the node to the branch's current head now"},
		}}, RunCtx: set},
		{Name: "unset", Description: "Stop following; the node keeps its current revision", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{node}}, RunCtx: unset},
		{Name: "check", Description: "Check a followed node now and dispatch an update if the branch moved", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{node}}, RunCtx: check},
	}}, nil
}

type policy struct {
	NodeID        string    `json:"node_id"`
	Branch        string    `json:"branch"`
	RepoURL       string    `json:"repo_url"`
	LastSeenHead  string    `json:"last_seen_head"`
	LastCheckedAt time.Time `json:"last_checked_at"`
	LastResult    string    `json:"last_result"`
	LastOpID      string    `json:"last_op_id"`
}

func (p policy) lines() []string {
	out := []string{fmt.Sprintf("%s follows %s (%s)", p.NodeID, p.Branch, p.RepoURL), "last result: " + p.LastResult}
	if !p.LastCheckedAt.IsZero() {
		out = append(out, "last checked: "+p.LastCheckedAt.Format(time.RFC3339))
	}
	if p.LastOpID != "" {
		out = append(out, "last provisioning op: "+p.LastOpID+" (`provision wait "+p.LastOpID+"`)")
	}
	return out
}

func nodePath(ctx cliapp.RunContext, suffix string) string {
	return "/follow/" + url.PathEscape(strings.TrimSpace(ctx.Flag("node"))) + suffix
}

func list(ctx cliapp.RunContext) error {
	data, err := ctx.Core().Get("/follow", url.Values{})
	if err != nil {
		return fmt.Errorf("list branch-follow policies: %w", err)
	}
	var result struct {
		Policies []policy `json:"policies"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode branch-follow policies: %w", err)
	}
	status := []string{fmt.Sprintf("%d node(s) follow a branch", len(result.Policies))}
	for _, p := range result.Policies {
		status = append(status, strings.Join(p.lines(), "\n  "))
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: status, NextSteps: []string{"`follow set --node <id> --branch <branch>` — keep a node on a branch's newest commit"}})
}

func set(ctx cliapp.RunContext) error {
	payload := map[string]any{"branch": ctx.Flag("branch"), "repo_url": ctx.Flag("repo-url"), "update_now": ctx.BoolFlag("now")}
	data, err := ctx.Core().Request(http.MethodPut, nodePath(ctx, ""), nil, payload)
	if err != nil {
		return fmt.Errorf("set node to follow %s: %w", ctx.Flag("branch"), err)
	}
	var p policy
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("decode branch-follow policy: %w", err)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Node follows " + p.Branch + "."}, Changes: p.lines(), NextCommand: []string{"`follow check --node " + p.NodeID + "` — check now instead of waiting for the next scheduled check"}})
}

func unset(ctx cliapp.RunContext) error {
	if _, err := ctx.Core().Request(http.MethodDelete, nodePath(ctx, ""), nil, nil); err != nil {
		return fmt.Errorf("stop following: %w", err)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Node no longer follows a branch; it keeps its current revision."}})
}

func check(ctx cliapp.RunContext) error {
	data, err := ctx.Core().Request(http.MethodPost, nodePath(ctx, "/check"), nil, map[string]any{})
	if err != nil {
		return fmt.Errorf("check followed node: %w", err)
	}
	var p policy
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("decode branch-follow policy: %w", err)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: p.lines()})
}
