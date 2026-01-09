package main

import (
	"flag"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdListDrafts(args []string) error {
	fs := flag.NewFlagSet("list-drafts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, parsed, err := a.services.Drafts.List()
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if parsed.Total == 0 || len(parsed.Drafts) == 0 {
		fmt.Println("No drafts found.")
		return nil
	}
	for _, draft := range parsed.Drafts {
		fmt.Printf("%s/%s (id=%s updated=%s)\n", draft.EntityType, draft.EntityName, draft.ID, draft.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}
