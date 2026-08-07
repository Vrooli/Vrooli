package worktree

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"

	"git-control-tower/cli/internal/callerheader"

	"github.com/vrooli/cli-core/cliapp"
)

// clientFactory builds a generated WorktreeServiceClient from the
// scenario app. Replaced in tests to inject an httptest-backed client.
// The callerheader interceptor stamps every outbound request with
// X-Vrooli-Caller (and X-Vrooli-Authorized when the agent-override env
// var is set) so the server's policygate interceptor can decide.
var clientFactory = func(core *cliapp.ScenarioApp) worktreeconnect.WorktreeServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return worktreeconnect.NewWorktreeServiceClient(httpClient, baseURL,
		connect.WithInterceptors(callerheader.New()))
}

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	repo := ctx.Flag("repo")
	client := clientFactory(h.core)
	resp, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: repo}))
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}
	out := ctx.Stdout()
	if len(resp.Msg.Worktrees) == 0 {
		fmt.Fprintln(out, "No worktrees found.")
		return nil
	}
	fmt.Fprintf(out, "Worktrees (%d):\n", len(resp.Msg.Worktrees))
	for _, w := range resp.Msg.Worktrees {
		marker := "  "
		if w.IsMain {
			marker = "* "
		}
		summary := w.Path
		if w.Branch != "" {
			summary = fmt.Sprintf("%s [%s]", w.Path, w.Branch)
		} else if w.Detached {
			summary = fmt.Sprintf("%s [detached %s]", w.Path, shortSHA(w.HeadCommit))
		}
		flags := []string{}
		if w.Locked {
			if w.LockReason != "" {
				flags = append(flags, fmt.Sprintf("locked: %s", w.LockReason))
			} else {
				flags = append(flags, "locked")
			}
		}
		if w.Prunable {
			flags = append(flags, fmt.Sprintf("prunable: %s", w.PrunableReason))
		}
		if len(flags) > 0 {
			summary += " (" + strings.Join(flags, "; ") + ")"
		}
		fmt.Fprintf(out, "%s%s\n", marker, summary)
	}
	return nil
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	client := clientFactory(h.core)
	resp, err := client.GetWorktree(context.Background(), connect.NewRequest(&worktreev1.GetWorktreeRequest{
		RepoPath: ctx.Flag("repo"), WorktreePath: ctx.Flag("path"),
	}))
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	w := resp.Msg.Worktree
	out := ctx.Stdout()
	fmt.Fprintf(out, "Path:   %s\n", w.Path)
	fmt.Fprintf(out, "Name:   %s\n", w.Name)
	if w.Branch != "" {
		fmt.Fprintf(out, "Branch: %s\n", w.Branch)
	}
	if w.Detached {
		fmt.Fprintf(out, "Branch: (detached at %s)\n", shortSHA(w.HeadCommit))
	}
	fmt.Fprintf(out, "HEAD:   %s\n", w.HeadCommit)
	if w.IsMain {
		fmt.Fprintln(out, "Main:   yes")
	}
	if w.Locked {
		fmt.Fprintf(out, "Locked: %s\n", w.LockReason)
	}
	if w.Prunable {
		fmt.Fprintf(out, "Prunable: %s\n", w.PrunableReason)
	}
	return nil
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	repo := ctx.Flag("repo")
	path := ctx.Flag("path")
	req := &worktreev1.CreateWorktreeRequest{
		RepoPath:        repo,
		NewWorktreePath: path,
		Force:           ctx.BoolFlag("force"),
		Track:           ctx.BoolFlag("track"),
	}
	switch {
	case ctx.Flag("existing_branch") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: ctx.Flag("existing_branch")}
	case ctx.Flag("new-branch") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_NewBranch{NewBranch: &worktreev1.NewBranchSpec{
			Name:       ctx.Flag("new-branch"),
			StartPoint: ctx.Flag("start"),
		}}
	case ctx.Flag("commit") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_Commit{Commit: ctx.Flag("commit")}
	default:
		return fmt.Errorf("create requires one of --branch, --new-branch, or --commit")
	}
	client := clientFactory(h.core)
	resp, err := client.CreateWorktree(context.Background(), connect.NewRequest(req))
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	w := resp.Msg.Worktree
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintf(out, "(dry-run) would create worktree %s\n", w.Path)
		return nil
	}
	fmt.Fprintf(out, "Created worktree %s\n", w.Path)
	if w.Branch != "" {
		fmt.Fprintf(out, "  Branch: %s\n", w.Branch)
	}
	return nil
}

func (h *handlers) remove(ctx cliapp.RunContext) error {
	path := ctx.Flag("path")
	client := clientFactory(h.core)
	resp, err := client.RemoveWorktree(context.Background(), connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: ctx.Flag("repo"), WorktreePath: path, Force: ctx.BoolFlag("force"),
	}))
	if err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintf(out, "(dry-run) would remove worktree %s\n", path)
		return nil
	}
	fmt.Fprintf(out, "Removed worktree %s\n", path)
	return nil
}

func (h *handlers) lock(ctx cliapp.RunContext) error {
	path := ctx.Flag("path")
	client := clientFactory(h.core)
	resp, err := client.LockWorktree(context.Background(), connect.NewRequest(&worktreev1.LockWorktreeRequest{
		RepoPath: ctx.Flag("repo"), WorktreePath: path, Reason: ctx.Flag("reason"),
	}))
	if err != nil {
		return fmt.Errorf("lock worktree: %w", err)
	}
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintf(out, "(dry-run) would lock worktree %s\n", path)
		return nil
	}
	fmt.Fprintf(out, "Locked worktree %s\n", path)
	return nil
}

func (h *handlers) unlock(ctx cliapp.RunContext) error {
	path := ctx.Flag("path")
	client := clientFactory(h.core)
	resp, err := client.UnlockWorktree(context.Background(), connect.NewRequest(&worktreev1.UnlockWorktreeRequest{
		RepoPath: ctx.Flag("repo"), WorktreePath: path,
	}))
	if err != nil {
		return fmt.Errorf("unlock worktree: %w", err)
	}
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintf(out, "(dry-run) would unlock worktree %s\n", path)
		return nil
	}
	fmt.Fprintf(out, "Unlocked worktree %s\n", path)
	return nil
}

func (h *handlers) move(ctx cliapp.RunContext) error {
	path := ctx.Flag("path")
	newPath := ctx.Flag("new-path")
	client := clientFactory(h.core)
	resp, err := client.MoveWorktree(context.Background(), connect.NewRequest(&worktreev1.MoveWorktreeRequest{
		RepoPath: ctx.Flag("repo"), WorktreePath: path, NewWorktreePath: newPath,
	}))
	if err != nil {
		return fmt.Errorf("move worktree: %w", err)
	}
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintf(out, "(dry-run) would move worktree %s -> %s\n", path, newPath)
		return nil
	}
	fmt.Fprintf(out, "Moved worktree %s -> %s\n", path, resp.Msg.Worktree.Path)
	return nil
}

func (h *handlers) prune(ctx cliapp.RunContext) error {
	client := clientFactory(h.core)
	resp, err := client.PruneWorktrees(context.Background(), connect.NewRequest(&worktreev1.PruneWorktreesRequest{
		RepoPath: ctx.Flag("repo"), Reason: ctx.Flag("reason"), ReportOnly: ctx.BoolFlag("report-only"),
	}))
	if err != nil {
		return fmt.Errorf("prune worktrees: %w", err)
	}
	out := ctx.Stdout()
	if resp.Msg.DryRun {
		fmt.Fprintln(out, "(dry-run) prune would run")
		return nil
	}
	if len(resp.Msg.PrunedPaths) == 0 {
		fmt.Fprintln(out, "Nothing to prune.")
		return nil
	}
	fmt.Fprintf(out, "Pruned %d worktree records:\n", len(resp.Msg.PrunedPaths))
	for _, p := range resp.Msg.PrunedPaths {
		fmt.Fprintf(out, "  - %s\n", p)
	}
	return nil
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
