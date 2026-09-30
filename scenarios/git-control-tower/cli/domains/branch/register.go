package branch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"
	branchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch"
	branchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/branch/branch_v1connect"
	humancontrolv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control"
	humancontrolconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control/human_control_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type info struct {
	Name      string `json:"name"`
	Upstream  string `json:"upstream,omitempty"`
	OID       string `json:"oid,omitempty"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

type listResponse struct {
	Current string `json:"current"`
	Locals  []info `json:"locals"`
	Remotes []info `json:"remotes"`
}

type warning struct {
	Message              string `json:"message"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	RequiresTracking     bool   `json:"requires_tracking,omitempty"`
	RequiresFetch        bool   `json:"requires_fetch,omitempty"`
}

type createRequest struct {
	Name       string `json:"name"`
	From       string `json:"from,omitempty"`
	Checkout   bool   `json:"checkout,omitempty"`
	AllowDirty bool   `json:"allow_dirty,omitempty"`
}

type createResponse struct {
	Success          bool     `json:"success"`
	Branch           *info    `json:"branch,omitempty"`
	Warning          *warning `json:"warning,omitempty"`
	Error            string   `json:"error,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
}

type switchRequest struct {
	Name        string `json:"name"`
	AllowDirty  bool   `json:"allow_dirty,omitempty"`
	TrackRemote bool   `json:"track_remote,omitempty"`
}

type switchResponse struct {
	Success bool     `json:"success"`
	Branch  *info    `json:"branch,omitempty"`
	Warning *warning `json:"warning,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type publishRequest struct {
	Remote string `json:"remote,omitempty"`
	Branch string `json:"branch,omitempty"`
	Fetch  bool   `json:"fetch,omitempty"`
}

type publishResponse struct {
	Success bool     `json:"success"`
	Remote  string   `json:"remote"`
	Branch  string   `json:"branch"`
	Warning *warning `json:"warning,omitempty"`
	Error   string   `json:"error,omitempty"`
}

var humanControlClientFactory = func(core *cliapp.ScenarioApp) humancontrolconnect.HumanControlServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return humancontrolconnect.NewHumanControlServiceClient(httpClient, baseURL)
}

var branchClientFactory = func(core *cliapp.ScenarioApp) branchconnect.BranchServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return branchconnect.NewBranchServiceClient(httpClient, baseURL)
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "branch",
		Description: "Manage repository branches",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List branches", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create branch NAME [--from=BASE] [--no-checkout] [--allow-dirty]", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "switch", NeedsAPI: true, Description: "Switch branch NAME [--allow-dirty] [--track-remote]", Run: func(args []string) error { return runSwitch(core, args) }},
			{Name: "publish", NeedsAPI: true, Description: "Publish current branch ([--remote=NAME] [--branch=NAME] [--fetch])", Run: func(args []string) error { return runPublish(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, _ []string) error {
	resp, err := branchClientFactory(core).ListBranches(context.Background(), connect.NewRequest(&branchv1.ListBranchesRequest{}))
	if err != nil {
		return err
	}
	if resp.Msg.Current != "" {
		fmt.Printf("Current: %s\n", resp.Msg.Current)
		fmt.Println("Local branches:")
		for _, branch := range resp.Msg.Locals {
			prefix := "  "
			if branch.IsCurrent {
				prefix = "* "
			}
			if branch.Upstream != "" {
				fmt.Printf("%s%s -> %s\n", prefix, branch.Name, branch.Upstream)
			} else {
				fmt.Printf("%s%s\n", prefix, branch.Name)
			}
		}
		if len(resp.Msg.Remotes) > 0 {
			fmt.Println("Remote branches:")
			for _, branch := range resp.Msg.Remotes {
				fmt.Printf("  %s\n", branch.Name)
			}
		}
		return nil
	}
	fmt.Println("No current branch")
	return nil
}

type createFlags struct {
	name       string
	from       string
	checkout   bool
	allowDirty bool
}

func parseCreateFlags(args []string) createFlags {
	f := createFlags{checkout: true}
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--from="):
			f.from = strings.TrimPrefix(arg, "--from=")
		case arg == "--no-checkout":
			f.checkout = false
		case arg == "--checkout":
			f.checkout = true
		case arg == "--allow-dirty":
			f.allowDirty = true
		case !strings.HasPrefix(arg, "-") && f.name == "":
			f.name = arg
		}
	}
	return f
}

func printCreateResult(resp *createResponse, name string, allowDirty bool) {
	if resp.Success {
		fmt.Printf("Created branch: %s\n", name)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresConfirmation && !allowDirty {
			fmt.Println("Retry with --allow-dirty to force checkout")
		}
		return
	}
	if len(resp.ValidationErrors) > 0 {
		fmt.Println("Validation errors:")
		for _, e := range resp.ValidationErrors {
			fmt.Printf("  ! %s\n", e)
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	f := parseCreateFlags(args)
	if f.name == "" {
		return fmt.Errorf("usage: branch create NAME [--from=BASE] [--no-checkout] [--allow-dirty]")
	}
	intent, err := confirmBranchMutation(core, "repo.branch.create")
	if err != nil {
		return err
	}
	resp, err := branchClientFactory(core).CreateBranch(context.Background(), connect.NewRequest(&branchv1.CreateBranchRequest{
		RepositoryId: intent.repositoryID,
		IntentId:     intent.intentID,
		Name:         f.name,
		From:         f.from,
		Checkout:     f.checkout,
		AllowDirty:   f.allowDirty,
	}))
	if err != nil {
		return err
	}
	printCreateResult(&createResponse{
		Success:          resp.Msg.Success,
		Error:            resp.Msg.Error,
		ValidationErrors: resp.Msg.ValidationErrors,
		Warning:          warningFromProto(resp.Msg.Warning),
	}, f.name, f.allowDirty)
	return nil
}

type switchFlags struct {
	name        string
	allowDirty  bool
	trackRemote bool
}

func parseSwitchFlags(args []string) switchFlags {
	var f switchFlags
	for _, arg := range args {
		switch {
		case arg == "--allow-dirty":
			f.allowDirty = true
		case arg == "--track-remote":
			f.trackRemote = true
		case !strings.HasPrefix(arg, "-") && f.name == "":
			f.name = arg
		}
	}
	return f
}

func printSwitchResult(resp *switchResponse, f switchFlags) {
	if resp.Success {
		fmt.Printf("Switched to: %s\n", f.name)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresTracking && !f.trackRemote {
			fmt.Println("Retry with --track-remote to track and switch")
		}
		if resp.Warning.RequiresConfirmation && !f.allowDirty {
			fmt.Println("Retry with --allow-dirty to force switch")
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func runSwitch(core *cliapp.ScenarioApp, args []string) error {
	f := parseSwitchFlags(args)
	if f.name == "" {
		return fmt.Errorf("usage: branch switch NAME [--allow-dirty] [--track-remote]")
	}
	intent, err := confirmBranchMutation(core, "repo.branch.switch")
	if err != nil {
		return err
	}
	resp, err := branchClientFactory(core).SwitchBranch(context.Background(), connect.NewRequest(&branchv1.SwitchBranchRequest{
		RepositoryId: intent.repositoryID,
		IntentId:     intent.intentID,
		Name:         f.name,
		AllowDirty:   f.allowDirty,
		TrackRemote:  f.trackRemote,
	}))
	if err != nil {
		return err
	}
	printSwitchResult(&switchResponse{Success: resp.Msg.Success, Error: resp.Msg.Error, Warning: warningFromProto(resp.Msg.Warning)}, f)
	return nil
}

type publishFlags struct {
	remote string
	branch string
	fetch  bool
}

func parsePublishFlags(args []string) publishFlags {
	var f publishFlags
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--remote="):
			f.remote = strings.TrimPrefix(arg, "--remote=")
		case strings.HasPrefix(arg, "--branch="):
			f.branch = strings.TrimPrefix(arg, "--branch=")
		case arg == "--fetch":
			f.fetch = true
		}
	}
	return f
}

func printPublishResult(resp *publishResponse, fetch bool) {
	if resp.Success {
		fmt.Printf("Published: %s to %s\n", resp.Branch, resp.Remote)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresFetch && !fetch {
			fmt.Println("Retry with --fetch to refresh remote status")
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func runPublish(core *cliapp.ScenarioApp, args []string) error {
	f := parsePublishFlags(args)
	intent, err := confirmBranchMutation(core, "repo.branch.publish")
	if err != nil {
		return err
	}
	resp, err := branchClientFactory(core).PublishBranch(context.Background(), connect.NewRequest(&branchv1.PublishBranchRequest{
		RepositoryId: intent.repositoryID,
		IntentId:     intent.intentID,
		Remote:       f.remote,
		Branch:       f.branch,
		Fetch:        f.fetch,
	}))
	if err != nil {
		return err
	}
	printPublishResult(&publishResponse{Success: resp.Msg.Success, Remote: resp.Msg.Remote, Branch: resp.Msg.Branch, Error: resp.Msg.Error, Warning: warningFromProto(resp.Msg.Warning)}, f.fetch)
	return nil
}

type branchIntent struct {
	repositoryID string
	intentID     string
}

func confirmBranchMutation(core *cliapp.ScenarioApp, operation string) (branchIntent, error) {
	client := humanControlClientFactory(core)
	ctx := context.Background()
	authorityResp, err := client.GetAuthorityStatus(ctx, connect.NewRequest(&humancontrolv1.GetAuthorityStatusRequest{}))
	if err != nil {
		return branchIntent{}, err
	}
	if !authorityResp.Msg.CanMutate {
		reason := authorityResp.Msg.Reason
		if reason == "" {
			reason = "authenticate through the configured provider"
		}
		return branchIntent{}, fmt.Errorf("mutation unavailable: %s (source: %s); reauthenticate at %s", reason, authorityResp.Msg.AuthSource, authorityResp.Msg.RecoveryUrl)
	}
	previewResp, err := client.PrepareMutation(ctx, connect.NewRequest(&humancontrolv1.PrepareMutationRequest{Operation: operation}))
	if err != nil {
		return branchIntent{}, err
	}
	preview := previewResp.Msg
	fmt.Printf("Repository: %s\nBranch: %s\nRevision: %s\nSubject digest: %s\nStaged files (%d):\n", preview.RepositoryPath, preview.Branch, preview.ExpectedRevision, preview.SubjectDigest, preview.FileCount)
	for _, path := range preview.StagedFiles {
		fmt.Printf("  %s\n", path)
	}
	fmt.Print("Type 'confirm' to authorize this exact mutation: ")
	line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
	if readErr != nil && readErr != io.EOF {
		return branchIntent{}, readErr
	}
	if strings.TrimSpace(line) != "confirm" {
		return branchIntent{}, fmt.Errorf("mutation not confirmed")
	}
	intentResp, err := client.ConfirmMutation(ctx, connect.NewRequest(&humancontrolv1.ConfirmMutationRequest{
		Operation: operation, RepositoryId: preview.RepositoryId,
		ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest,
	}))
	if err != nil {
		return branchIntent{}, err
	}
	if intentResp.Msg.IntentId == "" {
		return branchIntent{}, fmt.Errorf("mutation intent response did not contain an intent id")
	}
	return branchIntent{repositoryID: preview.RepositoryId, intentID: intentResp.Msg.IntentId}, nil
}

func warningFromProto(value *branchv1.BranchWarning) *warning {
	if value == nil {
		return nil
	}
	return &warning{
		Message:              value.Message,
		RequiresConfirmation: value.RequiresConfirmation,
		RequiresTracking:     value.RequiresTracking,
		RequiresFetch:        value.RequiresFetch,
	}
}
