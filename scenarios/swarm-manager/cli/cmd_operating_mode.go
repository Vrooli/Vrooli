package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// Top-level `swarm-manager operating-mode {list,get,set}` commands. Lives
// outside the per-initiative `initiatives mode-*` family because it operates
// on the mode catalog itself, not on a specific initiative's workspace.

type operatingModeListResponse struct {
	Modes []operatingModeListEntry `json:"modes"`
}

type operatingModeListEntry struct {
	Mode           string                      `json:"mode"`
	Label          string                      `json:"label"`
	Description    string                      `json:"description,omitempty"`
	UsageCount     int                         `json:"usage_count"`
	ScopeKind      string                      `json:"scope_kind"`
	RunStrategy    string                      `json:"run_strategy"`
	Default        bool                        `json:"default"`
	SupportsPhases bool                        `json:"supports_phases"`
	Phases         []operatingModeCatalogPhase `json:"phases,omitempty"`
}

type operatingModeDetailResponse struct {
	Entry             operatingModeListEntry       `json:"entry"`
	LinkedInitiatives []operatingModeLinkedInitRef `json:"linked_initiatives"`
}

type operatingModeLinkedInitRef struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Status  string `json:"status,omitempty"`
	Updated string `json:"updated,omitempty"`
}

func (a *App) cmdOperatingModeList(args []string) error {
	fs := flag.NewFlagSet("operating-mode list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/operating-modes", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeListResponse](body)
	if err != nil {
		return err
	}
	printSection("Operating Modes")
	if len(resp.Modes) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	for _, mode := range resp.Modes {
		defaultMark := ""
		if mode.Default {
			defaultMark = " [default]"
		}
		fmt.Printf("  - %s%s — %s\n", mode.Mode, defaultMark, mode.Label)
		if mode.Description != "" {
			fmt.Printf("    %s\n", mode.Description)
		}
		fmt.Printf("    scope=%s strategy=%s usage=%d initiative(s)\n", mode.ScopeKind, mode.RunStrategy, mode.UsageCount)
	}
	return nil
}

func (a *App) cmdOperatingModeGet(args []string) error {
	fs := flag.NewFlagSet("operating-mode get", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID (e.g., holistic-loop)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: operating-mode get --mode MODE [--json]\n\n%s", err)
	}
	mode := strings.TrimSpace(*modeFlag)
	body, err := a.core.Get("/operating-modes/"+mode, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeDetailResponse](body)
	if err != nil {
		return err
	}
	printOperatingModeDetail(resp)
	return nil
}

func (a *App) cmdOperatingModeSet(args []string) error {
	fs := flag.NewFlagSet("operating-mode set", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Operating mode ID")
	labelFlag := fs.String("label", "", "New display label")
	descFlag := fs.String("description", "", "New description (use --clear-description to remove)")
	clearDesc := fs.Bool("clear-description", false, "Clear the description override (restores registry default)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("mode", *modeFlag); err != nil {
		return fmt.Errorf("usage: operating-mode set --mode MODE [--label LABEL] [--description TEXT | --clear-description] [--json]\n\n%s", err)
	}
	mode := strings.TrimSpace(*modeFlag)
	patch := map[string]any{}
	if strings.TrimSpace(*labelFlag) != "" {
		patch["label"] = strings.TrimSpace(*labelFlag)
	}
	if *clearDesc {
		patch["description"] = ""
	} else if strings.TrimSpace(*descFlag) != "" {
		patch["description"] = strings.TrimSpace(*descFlag)
	}
	if len(patch) == 0 {
		return fmt.Errorf("at least one of --label, --description, or --clear-description is required")
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	body, err := a.core.Request("PATCH", "/operating-modes/"+mode, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	resp, err := decodeResponse[operatingModeDetailResponse](body)
	if err != nil {
		return err
	}
	printOperatingModeDetail(resp)
	return nil
}

func printOperatingModeDetail(resp operatingModeDetailResponse) {
	printSection("Operating Mode")
	fmt.Printf("  Mode:        %s\n", resp.Entry.Mode)
	fmt.Printf("  Label:       %s\n", resp.Entry.Label)
	if resp.Entry.Description != "" {
		fmt.Printf("  Description: %s\n", resp.Entry.Description)
	}
	fmt.Printf("  Scope:       %s\n", resp.Entry.ScopeKind)
	fmt.Printf("  Strategy:    %s\n", resp.Entry.RunStrategy)
	fmt.Printf("  Usage:       %d initiative(s)\n", resp.Entry.UsageCount)
	if len(resp.Entry.Phases) > 0 {
		fmt.Println("  Phases:")
		for _, phase := range resp.Entry.Phases {
			writeAccess := "read-only"
			if phase.WritesRepo {
				writeAccess = "writes repo"
			}
			fmt.Printf("    - %s (%s, %s)\n", phase.Phase, phase.ProfileKey, writeAccess)
		}
	}
	printSection("Linked Initiatives")
	if len(resp.LinkedInitiatives) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, init := range resp.LinkedInitiatives {
		title := init.Title
		if title == "" {
			title = "(untitled)"
		}
		statusSuffix := ""
		if init.Status != "" {
			statusSuffix = " [" + init.Status + "]"
		}
		fmt.Printf("  - %s — %s%s\n", init.Name, title, statusSuffix)
	}
}
