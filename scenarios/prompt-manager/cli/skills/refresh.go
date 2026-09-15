package skills

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"prompt-manager/cli/internal/appctx"
)

type refreshResponse struct {
	Digest string `json:"digest"`
	Rows   []struct {
		Runtime       string `json:"runtime"`
		Skill         string `json:"skill"`
		Status        string `json:"status"`
		SourceHash    string `json:"sourceHash"`
		InstalledHash string `json:"installedHash"`
		BaselineHash  string `json:"baselineHash"`
		ReceiptHash   string `json:"receiptHash"`
		Error         string `json:"error"`
		BackupPath    string `json:"backupPath"`
		Applied       bool   `json:"applied"`
	} `json:"rows"`
}

func cmdRefresh(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("refresh", flag.ContinueOnError)
	runtime := fs.String("runtime", "", "Declared runtime name; empty selects all configured targets")
	skills := fs.String("skills", "", "Comma-separated base-pack skill IDs; empty selects the base pack")
	apply := fs.Bool("apply", false, "Apply the reviewed preview")
	expected := fs.String("expected-digest", "", "Exact digest returned by preview")
	adopt := fs.Bool("adopt-legacy", false, "Adopt reviewed generated files without baselines, retaining backups")
	jsonOut := fs.Bool("json", false, "Print structured freshness and apply results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("refresh takes flags, not positional arguments")
	}
	if *apply && *expected == "" {
		return fmt.Errorf("--apply requires --expected-digest from a reviewed preview")
	}
	var ids []string
	if *skills != "" {
		for _, id := range strings.Split(*skills, ",") {
			ids = append(ids, strings.TrimSpace(id))
		}
	}
	var result refreshResponse
	if err := ctx.Post("/skills/refresh", map[string]any{"runtime": *runtime, "skills": ids, "apply": *apply, "expectedDigest": *expected, "adoptLegacy": *adopt}, &result); err != nil {
		return err
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
	} else {
		fmt.Printf("Preview digest: %s\n", result.Digest)
		for _, row := range result.Rows {
			fmt.Printf("%s/%s: %s applied=%t\n  source=%s installed=%s\n", row.Runtime, row.Skill, row.Status, row.Applied, row.SourceHash, row.InstalledHash)
			if row.Error != "" {
				fmt.Printf("  %s\n", row.Error)
			}
			if row.BackupPath != "" {
				fmt.Printf("  recovery copy: %s\n", row.BackupPath)
			}
		}
	}
	if *apply {
		for _, row := range result.Rows {
			if !row.Applied || row.Error != "" {
				return fmt.Errorf("refresh incomplete; inspect per-target results and preview again after resolving conflicts")
			}
		}
	}
	return nil
}
