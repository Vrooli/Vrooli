package audit

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/audit/audit_v1connect"
)

var auditClientFactory = func(core *cliapp.ScenarioApp) auditconnect.AuditServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return auditconnect.NewAuditServiceClient(httpClient, baseURL)
}

type entry struct {
	ID            int64    `json:"id"`
	Operation     string   `json:"operation"`
	RepoDir       string   `json:"repo_dir"`
	Branch        string   `json:"branch,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	CommitHash    string   `json:"commit_hash,omitempty"`
	CommitMessage string   `json:"commit_message,omitempty"`
	Success       bool     `json:"success"`
	Error         string   `json:"error,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

type flags struct {
	operation string
	limit     string
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "audit",
		Description: "Query git-control-tower audit logs",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "Query audit logs ([--operation=TYPE] [--limit=N])", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func parseFlags(args []string) flags {
	var f flags
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--operation="):
			f.operation = strings.TrimPrefix(arg, "--operation=")
		case strings.HasPrefix(arg, "--limit="):
			f.limit = strings.TrimPrefix(arg, "--limit=")
		}
	}
	return f
}

func formatEntry(e *entry) {
	status := "OK"
	if !e.Success {
		status = "FAIL"
	}
	fmt.Printf("[%s] %s %s\n", e.Operation, status, e.Timestamp)
	if e.CommitHash != "" {
		fmt.Printf("  Commit: %s\n", e.CommitHash)
	}
	if e.CommitMessage != "" {
		msg := e.CommitMessage
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}
		fmt.Printf("  Message: %s\n", msg)
	}
	if len(e.Paths) > 0 {
		fmt.Printf("  Paths: %s\n", strings.Join(e.Paths, ", "))
	}
	if e.Error != "" {
		fmt.Printf("  Error: %s\n", e.Error)
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	f := parseFlags(args)
	limit := int64(50)
	if f.limit != "" {
		parsed, err := strconv.ParseInt(f.limit, 10, 32)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("--limit must be a positive integer")
		}
		limit = parsed
	}
	result, err := auditClientFactory(core).ListEntries(context.Background(), connect.NewRequest(&auditv1.ListAuditEntriesRequest{
		Operation: f.operation,
		Limit:     int32(limit),
	}))
	if err != nil {
		return err
	}
	if result == nil || result.Msg == nil {
		return fmt.Errorf("audit query returned no result")
	}
	if len(result.Msg.GetEntries()) == 0 {
		fmt.Println("No audit entries found")
		return nil
	}
	fmt.Printf("Audit Log (%d of %d entries)\n", len(result.Msg.GetEntries()), result.Msg.GetTotal())
	fmt.Println(strings.Repeat("-", 60))
	for _, item := range result.Msg.GetEntries() {
		if item == nil {
			continue
		}
		formatEntry(&entry{ID: item.GetId(), Operation: item.GetOperation(), RepoDir: item.GetRepoDir(), Branch: item.GetBranch(), Paths: item.GetPaths(), CommitHash: item.GetCommitHash(), CommitMessage: item.GetCommitMessage(), Success: item.GetSuccess(), Error: item.GetError(), Timestamp: item.GetTimestamp()})
	}
	return nil
}
