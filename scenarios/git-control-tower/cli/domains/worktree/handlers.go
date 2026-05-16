package worktree

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"
	worktreeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree/worktree_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// clientFactory builds a generated WorktreeServiceClient from the
// scenario app. Replaced in tests to inject an httptest-backed client.
var clientFactory = func(core *cliapp.ScenarioApp) worktreeconnect.WorktreeServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return worktreeconnect.NewWorktreeServiceClient(httpClient, baseURL)
}

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	return &handlers{core: core}
}

// flagBag is a minimal positional/flag parser for the worktree
// commands. Sticks to the existing GCT CLI conventions (--key=value)
// to keep parity with the rest of the codebase.
type flagBag struct {
	values map[string]string
	flags  map[string]bool
}

func parse(args []string) flagBag {
	b := flagBag{values: map[string]string{}, flags: map[string]bool{}}
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			continue
		}
		a = strings.TrimPrefix(a, "--")
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			b.values[a[:eq]] = a[eq+1:]
		} else {
			b.flags[a] = true
		}
	}
	return b
}

func (f flagBag) Get(k string) string  { return f.values[k] }
func (f flagBag) Has(k string) bool    { return f.flags[k] || f.values[k] != "" }
func (f flagBag) Flag(k string) bool   { return f.flags[k] }

func (h *handlers) list(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	if repo == "" {
		return fmt.Errorf("usage: worktree list --repo=PATH")
	}
	client := clientFactory(h.core)
	resp, err := client.ListWorktrees(context.Background(), connect.NewRequest(&worktreev1.ListWorktreesRequest{RepoPath: repo}))
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}
	if len(resp.Msg.Worktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}
	fmt.Printf("Worktrees (%d):\n", len(resp.Msg.Worktrees))
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
		fmt.Printf("%s%s\n", marker, summary)
	}
	return nil
}

func (h *handlers) get(args []string) error {
	flags := parse(args)
	if flags.Get("repo") == "" || flags.Get("path") == "" {
		return fmt.Errorf("usage: worktree get --repo=PATH --path=PATH")
	}
	client := clientFactory(h.core)
	resp, err := client.GetWorktree(context.Background(), connect.NewRequest(&worktreev1.GetWorktreeRequest{
		RepoPath: flags.Get("repo"), WorktreePath: flags.Get("path"),
	}))
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	w := resp.Msg.Worktree
	fmt.Printf("Path:   %s\n", w.Path)
	fmt.Printf("Name:   %s\n", w.Name)
	if w.Branch != "" {
		fmt.Printf("Branch: %s\n", w.Branch)
	}
	if w.Detached {
		fmt.Printf("Branch: (detached at %s)\n", shortSHA(w.HeadCommit))
	}
	fmt.Printf("HEAD:   %s\n", w.HeadCommit)
	if w.IsMain {
		fmt.Println("Main:   yes")
	}
	if w.Locked {
		fmt.Printf("Locked: %s\n", w.LockReason)
	}
	if w.Prunable {
		fmt.Printf("Prunable: %s\n", w.PrunableReason)
	}
	return nil
}

func (h *handlers) create(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	path := flags.Get("path")
	if repo == "" || path == "" {
		return fmt.Errorf("usage: worktree create --repo=PATH --path=PATH (--branch=NAME | --new-branch=NAME [--start=REF] [--track] | --commit=SHA) [--force]")
	}
	req := &worktreev1.CreateWorktreeRequest{
		RepoPath:        repo,
		NewWorktreePath: path,
		Force:           flags.Flag("force"),
		Track:           flags.Flag("track"),
	}
	switch {
	case flags.Get("branch") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_ExistingBranch{ExistingBranch: flags.Get("branch")}
	case flags.Get("new-branch") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_NewBranch{NewBranch: &worktreev1.NewBranchSpec{
			Name:       flags.Get("new-branch"),
			StartPoint: flags.Get("start"),
		}}
	case flags.Get("commit") != "":
		req.Source = &worktreev1.CreateWorktreeRequest_Commit{Commit: flags.Get("commit")}
	default:
		return fmt.Errorf("create requires one of --branch, --new-branch, or --commit")
	}
	client := clientFactory(h.core)
	resp, err := client.CreateWorktree(context.Background(), connect.NewRequest(req))
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	w := resp.Msg.Worktree
	if resp.Msg.DryRun {
		fmt.Printf("(dry-run) would create worktree %s\n", w.Path)
		return nil
	}
	fmt.Printf("Created worktree %s\n", w.Path)
	if w.Branch != "" {
		fmt.Printf("  Branch: %s\n", w.Branch)
	}
	return nil
}

func (h *handlers) remove(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	path := flags.Get("path")
	if repo == "" || path == "" {
		return fmt.Errorf("usage: worktree remove --repo=PATH --path=PATH [--force]")
	}
	client := clientFactory(h.core)
	resp, err := client.RemoveWorktree(context.Background(), connect.NewRequest(&worktreev1.RemoveWorktreeRequest{
		RepoPath: repo, WorktreePath: path, Force: flags.Flag("force"),
	}))
	if err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}
	if resp.Msg.DryRun {
		fmt.Printf("(dry-run) would remove worktree %s\n", path)
		return nil
	}
	fmt.Printf("Removed worktree %s\n", path)
	return nil
}

func (h *handlers) lock(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	path := flags.Get("path")
	if repo == "" || path == "" {
		return fmt.Errorf("usage: worktree lock --repo=PATH --path=PATH [--reason=TEXT]")
	}
	client := clientFactory(h.core)
	resp, err := client.LockWorktree(context.Background(), connect.NewRequest(&worktreev1.LockWorktreeRequest{
		RepoPath: repo, WorktreePath: path, Reason: flags.Get("reason"),
	}))
	if err != nil {
		return fmt.Errorf("lock worktree: %w", err)
	}
	if resp.Msg.DryRun {
		fmt.Printf("(dry-run) would lock worktree %s\n", path)
		return nil
	}
	fmt.Printf("Locked worktree %s\n", path)
	return nil
}

func (h *handlers) unlock(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	path := flags.Get("path")
	if repo == "" || path == "" {
		return fmt.Errorf("usage: worktree unlock --repo=PATH --path=PATH")
	}
	client := clientFactory(h.core)
	resp, err := client.UnlockWorktree(context.Background(), connect.NewRequest(&worktreev1.UnlockWorktreeRequest{
		RepoPath: repo, WorktreePath: path,
	}))
	if err != nil {
		return fmt.Errorf("unlock worktree: %w", err)
	}
	if resp.Msg.DryRun {
		fmt.Printf("(dry-run) would unlock worktree %s\n", path)
		return nil
	}
	fmt.Printf("Unlocked worktree %s\n", path)
	return nil
}

func (h *handlers) move(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	path := flags.Get("path")
	newPath := flags.Get("new-path")
	if repo == "" || path == "" || newPath == "" {
		return fmt.Errorf("usage: worktree move --repo=PATH --path=PATH --new-path=PATH")
	}
	client := clientFactory(h.core)
	resp, err := client.MoveWorktree(context.Background(), connect.NewRequest(&worktreev1.MoveWorktreeRequest{
		RepoPath: repo, WorktreePath: path, NewWorktreePath: newPath,
	}))
	if err != nil {
		return fmt.Errorf("move worktree: %w", err)
	}
	if resp.Msg.DryRun {
		fmt.Printf("(dry-run) would move worktree %s -> %s\n", path, newPath)
		return nil
	}
	fmt.Printf("Moved worktree %s -> %s\n", path, resp.Msg.Worktree.Path)
	return nil
}

func (h *handlers) prune(args []string) error {
	flags := parse(args)
	repo := flags.Get("repo")
	if repo == "" {
		return fmt.Errorf("usage: worktree prune --repo=PATH [--report-only] [--reason=TEXT]")
	}
	client := clientFactory(h.core)
	resp, err := client.PruneWorktrees(context.Background(), connect.NewRequest(&worktreev1.PruneWorktreesRequest{
		RepoPath: repo, Reason: flags.Get("reason"), ReportOnly: flags.Flag("report-only"),
	}))
	if err != nil {
		return fmt.Errorf("prune worktrees: %w", err)
	}
	if resp.Msg.DryRun {
		fmt.Println("(dry-run) prune would run")
		return nil
	}
	if len(resp.Msg.PrunedPaths) == 0 {
		fmt.Println("Nothing to prune.")
		return nil
	}
	fmt.Printf("Pruned %d worktree records:\n", len(resp.Msg.PrunedPaths))
	for _, p := range resp.Msg.PrunedPaths {
		fmt.Printf("  - %s\n", p)
	}
	return nil
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
