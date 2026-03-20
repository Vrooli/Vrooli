package main

import (
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// --- Edge Commands ---

func (a *App) cmdEdgeList(args []string) error {
	fs, jsonOut, err := a.cmdFlags("edge list", args)
	if err != nil {
		return err
	}
	if err := requireArg(fs, "edge list <thought-id> [--json]"); err != nil {
		return err
	}
	return a.getResource("/thoughts/"+fs.Arg(0)+"/edges", jsonOut, func(body []byte) error {
		var edges []struct {
			ID       string `json:"id"`
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
			Label    string `json:"label"`
		}
		if err := unmarshalBody(body, &edges); err != nil {
			return err
		}
		if len(edges) == 0 {
			fmt.Println("No edges found.")
			return nil
		}
		for _, e := range edges {
			fmt.Printf("%-36s  %s -> %s  [%s]\n", e.ID, e.SourceID[:8], e.TargetID[:8], e.Label)
		}
		return nil
	})
}

func (a *App) cmdEdgeCreate(args []string) error {
	fs := newFlagSet("edge create")
	target := fs.String("target", "", "Target thought ID (required)")
	label := fs.String("label", "", "Edge label")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireArg(fs, "edge create <source-thought-id> --target TARGET_ID [--label LABEL]"); err != nil {
		return err
	}
	if *target == "" {
		return fmt.Errorf("--target is required")
	}
	return a.postResource("/thoughts/"+fs.Arg(0)+"/edges", map[string]string{
		"target_id": *target,
		"label":     *label,
	}, jsonOut, func(resp []byte) error {
		var e struct {
			ID string `json:"id"`
		}
		if err := unmarshalBody(resp, &e); err != nil {
			return err
		}
		fmt.Printf("Created edge %s\n", e.ID)
		return nil
	})
}

func (a *App) cmdEdgeDelete(args []string) error {
	fs := newFlagSet("edge delete")
	thoughtID := fs.String("thought", "", "Parent thought ID (required)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *thoughtID == "" {
		return fmt.Errorf("usage: edge delete <edge-id> --thought THOUGHT_ID")
	}
	return a.deleteResource("/thoughts/"+*thoughtID+"/edges/"+fs.Arg(0), "edge", fs.Arg(0))
}
