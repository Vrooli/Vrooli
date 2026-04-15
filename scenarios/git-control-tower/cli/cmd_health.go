package main

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// [REQ:GCT-OT-P0-002] Repository status API

type repoStatusResponse struct {
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

func (a *App) cmdRepoStatus(_ []string) error {
	body, err := a.core.Get("/repo/status", nil)
	if err != nil {
		return err
	}

	var parsed repoStatusResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.RepoDir != "" {
		fmt.Printf("Repo: %s\n", parsed.RepoDir)
		if parsed.Branch.Head != "" {
			fmt.Printf("Branch: %s\n", parsed.Branch.Head)
		}
		if parsed.Branch.Upstream != "" {
			fmt.Printf("Upstream: %s (ahead %d, behind %d)\n", parsed.Branch.Upstream, parsed.Branch.Ahead, parsed.Branch.Behind)
		}
		fmt.Printf("Changes: staged=%d unstaged=%d untracked=%d conflicts=%d\n",
			parsed.Summary.Staged, parsed.Summary.Unstaged, parsed.Summary.Untracked, parsed.Summary.Conflicts)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
