package repo

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	humancontrolv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control"
	humancontrolconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control/human_control_v1connect"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type statusResponse struct {
	RepoDir string `json:"repo_dir"`
	Branch  struct {
		Head     string `json:"head"`
		Upstream string `json:"upstream"`
		Ahead    int    `json:"ahead"`
		Behind   int    `json:"behind"`
	} `json:"branch"`
	Summary struct {
		Staged    int `json:"staged"`
		Unstaged  int `json:"unstaged"`
		Untracked int `json:"untracked"`
		Conflicts int `json:"conflicts"`
	} `json:"summary"`
}

type groupResponse struct {
	Key    string   `json:"key"`
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Source string   `json:"source"`
	Files  []string `json:"files"`
}

type groupsResponse struct {
	Groups []groupResponse `json:"groups"`
}

type diffResponse struct {
	RepoDir string `json:"repo_dir"`
	Path    string `json:"path"`
	Staged  bool   `json:"staged"`
	HasDiff bool   `json:"has_diff"`
	Stats   struct {
		Additions   int     `json:"additions"`
		Deletions   int     `json:"deletions"`
		Files       int     `json:"files"`
		NetLines    int     `json:"net_lines"`
		HunkCount   int     `json:"hunk_count"`
		LargestHunk int     `json:"largest_hunk"`
		Density     float64 `json:"density"`
	} `json:"stats"`
	Raw string `json:"raw"`
}

type stageRequest struct {
	Paths []string `json:"paths"`
	Scope string   `json:"scope,omitempty"`
}

type stageResponse struct {
	Success  bool     `json:"success"`
	Staged   []string `json:"staged"`
	Unstaged []string `json:"unstaged"`
	Failed   []string `json:"failed"`
	Errors   []string `json:"errors"`
}

type commitRequest struct {
	Message              string `json:"message"`
	IntentID             string `json:"intent_id,omitempty"`
	ValidateConventional bool   `json:"validate_conventional,omitempty"`
	Amend                bool   `json:"amend,omitempty"`
}

type commitResponse struct {
	Success          bool     `json:"success"`
	Hash             string   `json:"hash,omitempty"`
	Message          string   `json:"message,omitempty"`
	Amended          bool     `json:"amended,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
	Error            string   `json:"error,omitempty"`
}

var humanControlClientFactory = func(core *cliapp.ScenarioApp) humancontrolconnect.HumanControlServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return humancontrolconnect.NewHumanControlServiceClient(httpClient, baseURL)
}

var repoClientFactory = func(core *cliapp.ScenarioApp) repoconnect.RepoServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return repoconnect.NewRepoServiceClient(httpClient, baseURL)
}

type syncStatusResponse struct {
	Branch                string   `json:"branch"`
	Upstream              string   `json:"upstream,omitempty"`
	RemoteURL             string   `json:"remote_url,omitempty"`
	Ahead                 int      `json:"ahead"`
	Behind                int      `json:"behind"`
	HasUpstream           bool     `json:"has_upstream"`
	CanPush               bool     `json:"can_push"`
	CanPull               bool     `json:"can_pull"`
	NeedsPull             bool     `json:"needs_pull"`
	NeedsPush             bool     `json:"needs_push"`
	HasUncommittedChanges bool     `json:"has_uncommitted_changes"`
	SafetyWarnings        []string `json:"safety_warnings,omitempty"`
	Recommendations       []string `json:"recommendations,omitempty"`
	Fetched               bool     `json:"fetched"`
	FetchError            string   `json:"fetch_error,omitempty"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "repo",
		Description: "Inspect and change repository state",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Show repository status (branch + changed files)", Run: func(args []string) error { return runStatus(core, args) }},
			{Name: "groups", NeedsAPI: true, Description: "Show resolved repository change groups", Run: func(args []string) error { return runGroups(core, args) }},
			{Name: "diff", NeedsAPI: true, Description: "Show git diff (--path=FILE --staged)", Run: func(args []string) error { return runDiff(core, args) }},
			{Name: "blame", NeedsAPI: true, Description: "Show bounded native line attribution (--path=FILE [--revision=REV] [--start=N --end=N])", Run: func(args []string) error { return runBlame(core, args) }},
			{Name: "stage", NeedsAPI: true, Description: "Stage files (FILE... or --scope=scenario:name)", Run: func(args []string) error { return runStage(core, args) }},
			{Name: "unstage", NeedsAPI: true, Description: "Unstage files (FILE... or --scope=scenario:name)", Run: func(args []string) error { return runUnstage(core, args) }},
			{Name: "commit", NeedsAPI: true, Description: "Create a commit (-m MESSAGE [--conventional])", Run: func(args []string) error { return runCommit(core, args) }},
			{Name: "prepare-push-recovery", NeedsAPI: true, Description: "Review and prepare isolated recovery artifacts (--fingerprint=ID)", Run: func(args []string) error { return runPrepareRecovery(core, args) }},
			{Name: "push-safety", NeedsAPI: true, Description: "Inspect outgoing file sizes without changing the repository", Run: func(args []string) error { return runPushSafety(core, args) }},
			{Name: "push-recovery-status", NeedsAPI: true, Description: "Find latest retained preparation, or select --fingerprint=ID", Run: func(args []string) error { return runRecoveryStatus(core, args) }},
			{Name: "sync-status", NeedsAPI: true, Description: "Check push/pull status ([--fetch] [--remote=NAME])", Run: func(args []string) error { return runSyncStatus(core, args) }},
		},
	}
}

func runBlame(core *cliapp.ScenarioApp, args []string) error {
	request, jsonOutput, err := parseBlameFlags(args)
	if err != nil {
		return err
	}

	response, err := repoClientFactory(core).GetBlame(context.Background(), connect.NewRequest(request))
	if err != nil {
		return err
	}
	if jsonOutput {
		body, err := (protojson.MarshalOptions{Multiline: true, UseProtoNames: true}).Marshal(response.Msg)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}
	for _, file := range response.Msg.GetFiles() {
		fmt.Printf("%s [%s] lines=%d\n", file.GetPath(), file.GetStatus(), len(file.GetLines()))
		if file.GetReason() != "" {
			fmt.Printf("  reason: %s\n", file.GetReason())
		}
		for _, line := range file.GetLines() {
			fmt.Printf("  %d %s %s %s\n", line.GetLine(), line.GetCommit(), line.GetAuthor(), line.GetContent())
		}
	}
	for _, warning := range response.Msg.GetWarnings() {
		fmt.Printf("Warning: %s\n", warning)
	}
	if response.Msg.GetTruncated() {
		fmt.Println("Results truncated by safety bounds.")
	}
	return nil
}

func parseBlameFlags(args []string) (*repov1.GetBlameRequest, bool, error) {
	request := &repov1.GetBlameRequest{}
	jsonOutput := false
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--path="):
			request.Paths = append(request.Paths, strings.TrimPrefix(arg, "--path="))
		case strings.HasPrefix(arg, "--revision="):
			request.Revision = strings.TrimPrefix(arg, "--revision=")
		case strings.HasPrefix(arg, "--start="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--start="))
			if err != nil {
				return nil, false, fmt.Errorf("invalid --start value: %w", err)
			}
			request.StartLine = int32(value)
		case strings.HasPrefix(arg, "--end="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--end="))
			if err != nil {
				return nil, false, fmt.Errorf("invalid --end value: %w", err)
			}
			request.EndLine = int32(value)
		case arg == "--enrich":
			request.Enrich = true
		case arg == "--json":
			jsonOutput = true
		case !strings.HasPrefix(arg, "--"):
			request.Paths = append(request.Paths, arg)
		}
	}
	return request, jsonOutput, nil
}

func runGroups(core *cliapp.ScenarioApp, _ []string) error {
	response, err := repoClientFactory(core).GetRepoGroups(context.Background(), connect.NewRequest(&repov1.GetRepoGroupsRequest{}))
	if err != nil {
		return err
	}
	for _, group := range response.Msg.GetGroups() {
		fmt.Printf("%s kind=%s label=%s source=%s files=%d\n", group.GetKey(), group.GetKind(), group.GetLabel(), group.GetSource(), len(group.GetFiles()))
	}
	return nil
}

type diffFlags struct {
	path   string
	staged bool
}

func parseDiffFlags(args []string) diffFlags {
	var f diffFlags
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--path="):
			f.path = strings.TrimPrefix(arg, "--path=")
		case arg == "--staged":
			f.staged = true
		}
	}
	return f
}

func runStatus(core *cliapp.ScenarioApp, _ []string) error {
	response, err := repoClientFactory(core).GetRepoStatus(context.Background(), connect.NewRequest(&repov1.GetRepoStatusRequest{}))
	if err != nil {
		return err
	}

	parsed := statusResponse{RepoDir: response.Msg.RepoDir}
	if branch := response.Msg.GetBranchStatus(); branch != nil {
		parsed.Branch.Head = branch.GetHead()
		parsed.Branch.Upstream = branch.GetUpstream()
		parsed.Branch.Ahead = int(branch.GetAhead())
		parsed.Branch.Behind = int(branch.GetBehind())
	}
	if summary := response.Msg.GetSummary(); summary != nil {
		parsed.Summary.Staged = int(summary.GetStaged())
		parsed.Summary.Unstaged = int(summary.GetUnstaged())
		parsed.Summary.Untracked = int(summary.GetUntracked())
		parsed.Summary.Conflicts = int(summary.GetConflicts())
	}
	if parsed.RepoDir != "" {
		report := cliapp.OperationalReport{
			Status: []string{
				"Repo: " + parsed.RepoDir,
			},
			NextSteps: []string{
				"git-control-tower repo diff",
				"git-control-tower branch list",
			},
		}
		if parsed.Branch.Head != "" {
			report.Status = append(report.Status, "Branch: "+parsed.Branch.Head)
		}
		if parsed.Branch.Upstream != "" {
			report.Triage = append(report.Triage, cliapp.TriageGroup{
				Heading: "Upstream",
				Items: []string{
					fmt.Sprintf("%s (ahead %d, behind %d)", parsed.Branch.Upstream, parsed.Branch.Ahead, parsed.Branch.Behind),
				},
			})
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Changes",
			Items: []string{
				fmt.Sprintf("Staged: %d", parsed.Summary.Staged),
				fmt.Sprintf("Unstaged: %d", parsed.Summary.Unstaged),
				fmt.Sprintf("Untracked: %d", parsed.Summary.Untracked),
				fmt.Sprintf("Conflicts: %d", parsed.Summary.Conflicts),
			},
		})
		return cliapp.RenderOperationalReport(os.Stdout, report)
	}

	body, marshalErr := json.Marshal(response.Msg)
	if marshalErr != nil {
		return marshalErr
	}
	cliutil.PrintJSON(body)
	return nil
}

func formatDiffOutput(parsed *diffResponse) {
	fmt.Printf("Diff for: %s\n", parsed.Path)
	if parsed.Staged {
		fmt.Println("(staged changes)")
	}
	statLine := fmt.Sprintf("Stats: +%d -%d (net %+d)", parsed.Stats.Additions, parsed.Stats.Deletions, parsed.Stats.NetLines)
	if parsed.Stats.HunkCount > 0 {
		statLine += fmt.Sprintf(" | %d hunks, largest: %d lines", parsed.Stats.HunkCount, parsed.Stats.LargestHunk)
	}
	fmt.Println(statLine)
	if parsed.Raw != "" {
		fmt.Println("---")
		fmt.Println(parsed.Raw)
	}
}

func runDiff(core *cliapp.ScenarioApp, args []string) error {
	f := parseDiffFlags(args)
	response, err := repoClientFactory(core).GetRepoDiff(context.Background(), connect.NewRequest(&repov1.GetRepoDiffRequest{
		Path:   f.path,
		Staged: f.staged,
	}))
	if err != nil {
		return err
	}

	if !response.Msg.GetHasDiff() {
		fmt.Println("No changes")
		return nil
	}
	parsed := diffResponse{RepoDir: response.Msg.GetRepoDir(), Path: response.Msg.GetPath(), Staged: response.Msg.GetStaged(), HasDiff: response.Msg.GetHasDiff(), Raw: response.Msg.GetRaw()}
	if stats := response.Msg.GetStats(); stats != nil {
		parsed.Stats.Additions = int(stats.GetAdditions())
		parsed.Stats.Deletions = int(stats.GetDeletions())
		parsed.Stats.Files = int(stats.GetFiles())
		parsed.Stats.NetLines = int(stats.GetNetLines())
		parsed.Stats.HunkCount = int(stats.GetHunkCount())
		parsed.Stats.LargestHunk = int(stats.GetLargestHunk())
		parsed.Stats.Density = stats.GetDensity()
	}
	formatDiffOutput(&parsed)
	return nil
}

type stageFlags struct {
	scope string
	paths []string
}

func parseStageFlags(args []string) stageFlags {
	var f stageFlags
	for _, arg := range args {
		if strings.HasPrefix(arg, "--scope=") {
			f.scope = strings.TrimPrefix(arg, "--scope=")
		} else if !strings.HasPrefix(arg, "-") {
			f.paths = append(f.paths, arg)
		}
	}
	return f
}

func printStageResult(parsed *stageResponse) {
	if parsed.Success {
		fmt.Printf("Staged %d file(s)\n", len(parsed.Staged))
		for _, f := range parsed.Staged {
			fmt.Printf("  + %s\n", f)
		}
	} else {
		fmt.Println("Staging failed:")
		for _, e := range parsed.Errors {
			fmt.Printf("  ! %s\n", e)
		}
	}
}

func printUnstageResult(parsed *stageResponse) {
	if parsed.Success {
		fmt.Printf("Unstaged %d file(s)\n", len(parsed.Unstaged))
		for _, f := range parsed.Unstaged {
			fmt.Printf("  - %s\n", f)
		}
	} else {
		fmt.Println("Unstaging failed:")
		for _, e := range parsed.Errors {
			fmt.Printf("  ! %s\n", e)
		}
	}
}

func runStage(core *cliapp.ScenarioApp, args []string) error {
	f := parseStageFlags(args)
	if len(f.paths) == 0 && f.scope == "" {
		return fmt.Errorf("usage: repo stage FILE... or --scope=scenario:name")
	}
	intent, err := confirmRepoMutation(core, "repo.stage")
	if err != nil {
		return err
	}
	resp, err := repoClientFactory(core).StageFiles(context.Background(), connect.NewRequest(&repov1.StageFilesRequest{
		RepositoryId: intent.repositoryID, IntentId: intent.intentID, Paths: f.paths, Scope: f.scope,
	}))
	if err != nil {
		return err
	}
	printStageResult(&stageResponse{Success: resp.Msg.Success, Staged: resp.Msg.Staged, Failed: resp.Msg.Failed, Errors: resp.Msg.Errors})
	return nil
}

func runUnstage(core *cliapp.ScenarioApp, args []string) error {
	f := parseStageFlags(args)
	if len(f.paths) == 0 && f.scope == "" {
		return fmt.Errorf("usage: repo unstage FILE... or --scope=scenario:name")
	}
	intent, err := confirmRepoMutation(core, "repo.unstage")
	if err != nil {
		return err
	}
	resp, err := repoClientFactory(core).UnstageFiles(context.Background(), connect.NewRequest(&repov1.UnstageFilesRequest{
		RepositoryId: intent.repositoryID, IntentId: intent.intentID, Paths: f.paths, Scope: f.scope,
	}))
	if err != nil {
		return err
	}
	printUnstageResult(&stageResponse{Success: resp.Msg.Success, Unstaged: resp.Msg.Unstaged, Failed: resp.Msg.Failed, Errors: resp.Msg.Errors})
	return nil
}

type repoIntent struct {
	repositoryID string
	intentID     string
}

func confirmRepoMutation(core *cliapp.ScenarioApp, operation string) (repoIntent, error) {
	return confirmRepoMutationContext(core, operation, "")
}

func confirmRepoMutationContext(core *cliapp.ScenarioApp, operation, subjectContext string) (repoIntent, error) {
	client := humanControlClientFactory(core)
	ctx := context.Background()
	authorityResp, err := client.GetAuthorityStatus(ctx, connect.NewRequest(&humancontrolv1.GetAuthorityStatusRequest{}))
	if err != nil {
		return repoIntent{}, err
	}
	if !authorityResp.Msg.CanMutate {
		return repoIntent{}, formatAuthorityRefusal(authorityResp.Msg.AuthSource, authorityResp.Msg.Reason, authorityResp.Msg.RecoveryUrl)
	}
	previewResp, err := client.PrepareMutation(ctx, connect.NewRequest(&humancontrolv1.PrepareMutationRequest{Operation: operation, SubjectContext: subjectContext}))
	if err != nil {
		return repoIntent{}, err
	}
	preview := previewResp.Msg
	fmt.Printf("Repository: %s\nBranch: %s\nRevision: %s\nSubject digest: %s\nStaged files (%d):\n", preview.RepositoryPath, preview.Branch, preview.ExpectedRevision, preview.SubjectDigest, preview.FileCount)
	for _, path := range preview.StagedFiles {
		fmt.Printf("  %s\n", path)
	}
	fmt.Print("Type 'confirm' to authorize this exact mutation: ")
	line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
	if readErr != nil && readErr != io.EOF {
		return repoIntent{}, readErr
	}
	if strings.TrimSpace(line) != "confirm" {
		return repoIntent{}, fmt.Errorf("mutation not confirmed")
	}
	intentResp, err := client.ConfirmMutation(ctx, connect.NewRequest(&humancontrolv1.ConfirmMutationRequest{
		Operation: operation, RepositoryId: preview.RepositoryId, SubjectContext: subjectContext,
		ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest,
	}))
	if err != nil {
		return repoIntent{}, err
	}
	if intentResp.Msg.IntentId == "" {
		return repoIntent{}, fmt.Errorf("mutation intent response did not contain an intent id")
	}
	return repoIntent{repositoryID: preview.RepositoryId, intentID: intentResp.Msg.IntentId}, nil
}

type commitFlags struct {
	message      string
	conventional bool
	amend        bool
	confirmed    bool
}

func parseCommitFlags(args []string) commitFlags {
	var f commitFlags
	for i, arg := range args {
		switch {
		case arg == "-m" && i+1 < len(args):
			f.message = args[i+1]
		case strings.HasPrefix(arg, "-m="):
			f.message = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--message="):
			f.message = strings.TrimPrefix(arg, "--message=")
		case arg == "--conventional":
			f.conventional = true
		case arg == "--amend":
			f.amend = true
		case arg == "--yes" || arg == "--confirm":
			f.confirmed = true
		}
	}
	return f
}

func printCommitResult(parsed *commitResponse) {
	if parsed.Success {
		action := "Committed"
		if parsed.Amended {
			action = "Amended"
		}
		fmt.Printf("%s: %s\n", action, parsed.Hash)
		fmt.Printf("Message: %s\n", parsed.Message)
	} else {
		fmt.Println("Commit failed:")
		if parsed.Error != "" {
			fmt.Printf("  Error: %s\n", parsed.Error)
		}
		for _, e := range parsed.ValidationErrors {
			fmt.Printf("  ! %s\n", e)
		}
	}
}

func runCommit(core *cliapp.ScenarioApp, args []string) error {
	f := parseCommitFlags(args)
	if f.message == "" && !f.amend {
		return fmt.Errorf("usage: repo commit [-m MESSAGE] [--conventional] [--amend] [--yes]")
	}
	client := humanControlClientFactory(core)
	authorityResp, err := client.GetAuthorityStatus(context.Background(), connect.NewRequest(&humancontrolv1.GetAuthorityStatusRequest{}))
	if err != nil {
		return err
	}
	if !authorityResp.Msg.CanMutate {
		return formatAuthorityRefusal(authorityResp.Msg.AuthSource, authorityResp.Msg.Reason, authorityResp.Msg.RecoveryUrl)
	}
	previewResp, err := client.PrepareMutation(context.Background(), connect.NewRequest(&humancontrolv1.PrepareMutationRequest{Operation: "repo.commit"}))
	if err != nil {
		return err
	}
	preview := previewResp.Msg
	fmt.Printf("Repository: %s\nBranch: %s\nRevision: %s\nSubject digest: %s\nStaged files (%d):\n", preview.RepositoryPath, preview.Branch, preview.ExpectedRevision, preview.SubjectDigest, preview.FileCount)
	for _, path := range preview.StagedFiles {
		fmt.Printf("  %s\n", path)
	}
	if !f.confirmed {
		fmt.Print("Type 'confirm' to authorize this exact mutation: ")
		line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if strings.TrimSpace(line) != "confirm" {
			return fmt.Errorf("mutation not confirmed")
		}
	}
	intentResp, err := client.ConfirmMutation(context.Background(), connect.NewRequest(&humancontrolv1.ConfirmMutationRequest{Operation: preview.Operation, ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest}))
	if err != nil {
		return err
	}
	if intentResp.Msg.IntentId == "" {
		return fmt.Errorf("mutation intent response did not contain an intent id")
	}
	req := &repov1.CreateCommitRequest{RepositoryId: preview.RepositoryId, Message: f.message, IntentId: intentResp.Msg.IntentId, ValidateConventional: f.conventional, Amend: f.amend}
	commitResp, err := repoClientFactory(core).CreateCommit(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	printCommitResult(&commitResponse{Success: commitResp.Msg.Success, Hash: commitResp.Msg.Hash, Message: commitResp.Msg.Message, Amended: commitResp.Msg.Amended, ValidationErrors: commitResp.Msg.ValidationErrors, Error: commitResp.Msg.Error})
	return nil
}

func formatAuthorityRefusal(source, reason, recoveryURL string) error {
	if reason == "" {
		reason = "authenticate through the configured provider"
	}
	message := "mutation unavailable: " + reason
	if source != "" {
		message += " (source: " + source + ")"
	}
	if recoveryURL != "" {
		message += "; reauthenticate at " + recoveryURL
	}
	return fmt.Errorf("%s", message)
}

type syncStatusFlags struct {
	fetch  bool
	remote string
}

func parseSyncStatusFlags(args []string) syncStatusFlags {
	var f syncStatusFlags
	for _, arg := range args {
		switch {
		case arg == "--fetch":
			f.fetch = true
		case strings.HasPrefix(arg, "--remote="):
			f.remote = strings.TrimPrefix(arg, "--remote=")
		}
	}
	return f
}

func formatSyncBranchInfo(resp *syncStatusResponse) {
	fmt.Printf("Branch: %s\n", resp.Branch)
	if resp.Upstream != "" {
		fmt.Printf("Upstream: %s\n", resp.Upstream)
	}
	if resp.RemoteURL != "" {
		fmt.Printf("Remote: %s\n", resp.RemoteURL)
	}
	if resp.HasUpstream {
		fmt.Printf("Ahead: %d  Behind: %d\n", resp.Ahead, resp.Behind)
	} else {
		fmt.Println("No upstream configured")
	}
}

func formatSyncActions(resp *syncStatusResponse) {
	var actions []string
	if resp.CanPush {
		actions = append(actions, "can push")
	}
	if resp.CanPull {
		actions = append(actions, "can pull")
	}
	if resp.HasUncommittedChanges {
		actions = append(actions, "has uncommitted changes")
	}
	if len(actions) > 0 {
		fmt.Printf("Status: %s\n", strings.Join(actions, ", "))
	}
}

func formatSyncWarnings(resp *syncStatusResponse) {
	if len(resp.SafetyWarnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range resp.SafetyWarnings {
			fmt.Printf("  ! %s\n", w)
		}
	}
	if len(resp.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for _, r := range resp.Recommendations {
			fmt.Printf("  -> %s\n", r)
		}
	}
	if resp.Fetched {
		fmt.Println("\n(fetched fresh data from remote)")
	}
	if resp.FetchError != "" {
		fmt.Printf("\n! Fetch error: %s\n", resp.FetchError)
	}
}

func runSyncStatus(core *cliapp.ScenarioApp, args []string) error {
	f := parseSyncStatusFlags(args)
	response, err := repoClientFactory(core).GetSyncStatus(context.Background(), connect.NewRequest(&repov1.GetSyncStatusRequest{
		Fetch: f.fetch, Remote: f.remote,
	}))
	if err != nil {
		return err
	}
	msg := response.Msg
	resp := syncStatusResponse{
		Branch: msg.GetBranch(), Upstream: msg.GetUpstream(), RemoteURL: msg.GetRemoteUrl(),
		Ahead: int(msg.GetAhead()), Behind: int(msg.GetBehind()), HasUpstream: msg.GetHasUpstream(),
		CanPush: msg.GetCanPush(), CanPull: msg.GetCanPull(), NeedsPull: msg.GetNeedsPull(),
		NeedsPush: msg.GetNeedsPush(), HasUncommittedChanges: msg.GetHasUncommittedChanges(),
		SafetyWarnings: msg.GetSafetyWarnings(), Recommendations: msg.GetRecommendations(),
		Fetched: msg.GetFetched(), FetchError: msg.GetFetchError(),
	}
	formatSyncBranchInfo(&resp)
	formatSyncActions(&resp)
	formatSyncWarnings(&resp)
	return nil
}
