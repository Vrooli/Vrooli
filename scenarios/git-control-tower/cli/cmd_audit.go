package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// [REQ:GCT-OT-P0-007] Audit log query

type auditEntry struct {
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

type auditResponse struct {
	Entries []auditEntry `json:"entries"`
	Total   int          `json:"total"`
}

type auditFlags struct {
	operation string
	limit     string
}

func parseAuditFlags(args []string) auditFlags {
	var f auditFlags
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

func formatAuditEntry(e *auditEntry) {
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

func (a *App) cmdAudit(args []string) error {
	f := parseAuditFlags(args)

	query := url.Values{}
	if f.operation != "" {
		query.Set("operation", f.operation)
	}
	if f.limit != "" {
		query.Set("limit", f.limit)
	}

	body, err := a.core.Get("/audit", query)
	if err != nil {
		return err
	}

	var resp auditResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil && resp.Entries != nil {
		if len(resp.Entries) == 0 {
			fmt.Println("No audit entries found")
			return nil
		}

		fmt.Printf("Audit Log (%d of %d entries)\n", len(resp.Entries), resp.Total)
		fmt.Println(strings.Repeat("-", 60))

		for i := range resp.Entries {
			formatAuditEntry(&resp.Entries[i])
		}

		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
