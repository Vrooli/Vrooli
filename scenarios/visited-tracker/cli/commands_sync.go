package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	patterns := fs.String("patterns", "", "Comma-separated patterns")
	removeDeleted := fs.Bool("remove-deleted", false, "Remove deleted files")
	_ = fs.String("structure", "", "Structure snapshot (ignored)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	campaignID, err := a.resolveCampaignID(campaignAutoOptions{}, *jsonOutput)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(*patterns) != "" {
		payload["patterns"] = cliutil.ParseCSV(*patterns)
	}
	if *removeDeleted {
		payload["remove_deleted"] = true
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/campaigns/"+campaignID+"/structure/sync"), nil, payload)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var response syncResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse sync response: %w", err)
	}

	fmt.Println("Structure synchronized")
	fmt.Printf("Added: %d files\n", response.Added)
	fmt.Printf("Removed: %d files\n", response.Removed)
	fmt.Printf("Total: %d files\n", response.Total)
	return nil
}
