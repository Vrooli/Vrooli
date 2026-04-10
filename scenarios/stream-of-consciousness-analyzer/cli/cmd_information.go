package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// --- Information Commands ---

func (a *App) cmdInfoList(args []string) error {
	fs, jsonOut, err := a.cmdFlags("info list", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "info list <scheme-id> [--json]"); err != nil {
		return err
	}
	return a.getResource("/schemes/"+fs.Arg(0)+"/information", jsonOut, func(body []byte) error {
		var items []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if err := unmarshalBody(body, &items); err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No information items found.")
			return nil
		}
		for _, i := range items {
			content := i.Content
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			fmt.Printf("%-36s  [%s] %s\n", i.ID, i.Type, content)
		}
		return nil
	})
}

func (a *App) cmdInfoCreate(args []string) error {
	fs := newFlagSet("info create")
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	infoType := fs.String("type", "text", "Information type")
	content := fs.String("content", "", "Content (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *schemeID == "" || *content == "" {
		return fmt.Errorf("usage: info create --scheme ID --content TEXT [--type TYPE]")
	}
	return a.postResource("/schemes/"+*schemeID+"/information", map[string]any{
		"type":    *infoType,
		"content": *content,
	}, jsonOut, func(resp []byte) error {
		var i struct {
			ID string `json:"id"`
		}
		if err := unmarshalBody(resp, &i); err != nil {
			return err
		}
		fmt.Printf("Created information item %s\n", i.ID)
		return nil
	})
}

func (a *App) cmdInfoUpdate(args []string) error {
	fs := newFlagSet("info update")
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	content := fs.String("content", "", "New content")
	infoType := fs.String("type", "", "New type")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *schemeID == "" {
		return fmt.Errorf("usage: info update <info-id> --scheme SCHEME_ID [--content TEXT] [--type TYPE]")
	}
	input := map[string]any{}
	if *content != "" {
		input["content"] = *content
	}
	if *infoType != "" {
		input["type"] = *infoType
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one of --content or --type is required")
	}
	return a.putResource("/schemes/"+*schemeID+"/information/"+fs.Arg(0), input, jsonOut, func([]byte) error {
		fmt.Printf("Updated information item %s\n", fs.Arg(0))
		return nil
	})
}

func (a *App) cmdInfoDelete(args []string) error {
	fs := newFlagSet("info delete")
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *schemeID == "" {
		return fmt.Errorf("usage: info delete <info-id> --scheme SCHEME_ID")
	}
	return a.deleteResource("/schemes/"+*schemeID+"/information/"+fs.Arg(0), "information item", fs.Arg(0))
}
