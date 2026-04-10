package main

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// --- Scheme Commands ---

func (a *App) cmdSchemeList(args []string) error {
	_, jsonOut, err := a.cmdFlags("scheme list", args)
	if err != nil {
		return err
	}
	return a.getResource("/schemes", jsonOut, func(body []byte) error {
		var schemes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := unmarshalBody(body, &schemes); err != nil {
			return err
		}
		if len(schemes) == 0 {
			fmt.Println("No schemes found.")
			return nil
		}
		for _, s := range schemes {
			fmt.Printf("%-36s  %s\n", s.ID, s.Name)
		}
		return nil
	})
}

func (a *App) cmdSchemeGet(args []string) error {
	fs, jsonOut, err := a.cmdFlags("scheme get", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "scheme get <id> [--json]"); err != nil {
		return err
	}
	return a.getResource("/schemes/"+fs.Arg(0), jsonOut, func(body []byte) error {
		var s struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			CreatedAt string `json:"created_at"`
		}
		if err := unmarshalBody(body, &s); err != nil {
			return err
		}
		fmt.Printf("ID:      %s\nName:    %s\nCreated: %s\n", s.ID, s.Name, s.CreatedAt)
		return nil
	})
}

func (a *App) cmdSchemeCreate(args []string) error {
	fs := newFlagSet("scheme create")
	name := fs.String("name", "", "Scheme name (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" {
		if fs.NArg() > 0 {
			*name = fs.Arg(0)
		} else {
			return fmt.Errorf("usage: scheme create <name> or --name NAME")
		}
	}
	return a.postResource("/schemes", map[string]string{"name": *name}, jsonOut, func(body []byte) error {
		var s struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := unmarshalBody(body, &s); err != nil {
			return err
		}
		fmt.Printf("Created scheme: %s (ID: %s)\n", s.Name, s.ID)
		return nil
	})
}

func (a *App) cmdSchemeUpdate(args []string) error {
	fs := newFlagSet("scheme update")
	name := fs.String("name", "", "New scheme name (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireArg(fs, "scheme update <id> --name NAME [--json]"); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	return a.putResource("/schemes/"+fs.Arg(0), map[string]string{"name": *name}, jsonOut, func([]byte) error {
		fmt.Printf("Updated scheme %s\n", fs.Arg(0))
		return nil
	})
}

func (a *App) cmdSchemeDelete(args []string) error {
	fs, _, err := a.cmdFlags("scheme delete", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "scheme delete <id>"); err != nil {
		return err
	}
	return a.deleteResource("/schemes/"+fs.Arg(0), "scheme", fs.Arg(0))
}

func (a *App) cmdSchemeExport(args []string) error {
	fs, jsonOut, err := a.cmdFlags("scheme export", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "scheme export <id> [--json]"); err != nil {
		return err
	}
	return a.getResource("/schemes/"+fs.Arg(0)+"/export", jsonOut, func(body []byte) error {
		var export struct {
			Scheme struct {
				Name string `json:"name"`
			} `json:"scheme"`
			Information []json.RawMessage `json:"information"`
			Thoughts    []json.RawMessage `json:"thoughts"`
			Edges       []json.RawMessage `json:"edges"`
			Format      string            `json:"export_format"`
		}
		if err := unmarshalBody(body, &export); err != nil {
			return err
		}
		fmt.Printf("Scheme:      %s\n", export.Scheme.Name)
		fmt.Printf("Format:      %s\n", export.Format)
		fmt.Printf("Information: %d items\n", len(export.Information))
		fmt.Printf("Thoughts:    %d nodes\n", len(export.Thoughts))
		fmt.Printf("Edges:       %d connections\n", len(export.Edges))
		return nil
	})
}
