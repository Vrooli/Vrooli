package main

import (
	"fmt"
	"strings"
)

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
