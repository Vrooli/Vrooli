package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// --- Thought Commands ---

func (a *App) cmdThoughtList(args []string) error {
	fs := newFlagSet("thought list")
	schemeID := fs.String("scheme", "", "Filter by scheme ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	path := "/thoughts"
	if *schemeID != "" {
		path += "?scheme_id=" + *schemeID
	}
	return a.getResource(path, jsonOut, func(body []byte) error {
		var thoughts []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := unmarshalBody(body, &thoughts); err != nil {
			return err
		}
		if len(thoughts) == 0 {
			fmt.Println("No thoughts found.")
			return nil
		}
		for _, t := range thoughts {
			fmt.Printf("%-36s  %s\n", t.ID, t.Title)
		}
		return nil
	})
}

func (a *App) cmdThoughtGet(args []string) error {
	fs, jsonOut, err := a.cmdFlags("thought get", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "thought get <id> [--json]"); err != nil {
		return err
	}
	return a.getResource("/thoughts/"+fs.Arg(0), jsonOut, func(body []byte) error {
		var t struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		if err := unmarshalBody(body, &t); err != nil {
			return err
		}
		fmt.Printf("ID:    %s\nTitle: %s\nBody:  %s\n", t.ID, t.Title, t.Body)
		return nil
	})
}

func (a *App) cmdThoughtCreate(args []string) error {
	fs := newFlagSet("thought create")
	title := fs.String("title", "", "Thought title (required)")
	body := fs.String("body", "", "Thought body")
	schemeID := fs.String("scheme", "", "Scheme ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *title == "" {
		return fmt.Errorf("usage: thought create --title TITLE [--body BODY] [--scheme ID]")
	}
	input := map[string]any{
		"title": *title,
		"body":  *body,
	}
	if *schemeID != "" {
		input["scheme_id"] = *schemeID
	}
	return a.postResource("/thoughts", input, jsonOut, func(resp []byte) error {
		var t struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := unmarshalBody(resp, &t); err != nil {
			return err
		}
		fmt.Printf("Created thought: %s (ID: %s)\n", t.Title, t.ID)
		return nil
	})
}

func (a *App) cmdThoughtUpdate(args []string) error {
	fs := newFlagSet("thought update")
	title := fs.String("title", "", "New title")
	body := fs.String("body", "", "New body")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireArg(fs, "thought update <id> [--title TITLE] [--body BODY] [--json]"); err != nil {
		return err
	}
	input := map[string]any{}
	if *title != "" {
		input["title"] = *title
	}
	if *body != "" {
		input["body"] = *body
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one of --title or --body is required")
	}
	return a.putResource("/thoughts/"+fs.Arg(0), input, jsonOut, func([]byte) error {
		fmt.Printf("Updated thought %s\n", fs.Arg(0))
		return nil
	})
}

func (a *App) cmdThoughtDelete(args []string) error {
	fs, _, err := a.cmdFlags("thought delete", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "thought delete <id>"); err != nil {
		return err
	}
	return a.deleteResource("/thoughts/"+fs.Arg(0), "thought", fs.Arg(0))
}
