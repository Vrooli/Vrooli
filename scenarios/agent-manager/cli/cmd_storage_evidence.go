package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Storage: the shared effort evidence home (~/.vrooli/evidence)
// =============================================================================

type evidenceBytes struct {
	Regenerable          int64            `json:"regenerableBytes"`
	ProtectedRegenerable int64            `json:"protectedRegenerableBytes"`
	Referenced           int64            `json:"referencedBytes"`
	Unknown              int64            `json:"unknownBytes"`
	Total                int64            `json:"totalBytes"`
	ByClass              map[string]int64 `json:"regenerableByClass"`
}

type evidenceReport struct {
	Root       string        `json:"root"`
	Grace      string        `json:"grace"`
	CitedPaths int           `json:"citedPaths"`
	Totals     evidenceBytes `json:"totals"`
	Efforts    []struct {
		Name      string `json:"name"`
		Lifecycle struct {
			State        string `json:"state"`
			AutoEligible bool   `json:"autoEligible"`
			Reason       string `json:"reason"`
		} `json:"lifecycle"`
		Bytes evidenceBytes `json:"bytes"`
	} `json:"efforts"`
	Warnings []string `json:"warnings"`
}

// storageEvidence prints the dry-run classification of the evidence home.
func (a *App) storageEvidence(args []string) error {
	fs := flag.NewFlagSet("storage evidence", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Maintenance.api.WithTimeout(10*time.Minute).Request("GET", "/api/v1/storage/evidence", nil, nil)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var r evidenceReport
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("decode evidence report: %w", err)
	}
	fmt.Printf("Evidence: %s total %s (regenerable %s, cited-regenerable %s, referenced %s, unknown %s) cited paths=%d grace=%s\n",
		r.Root, humanBytes(r.Totals.Total), humanBytes(r.Totals.Regenerable), humanBytes(r.Totals.ProtectedRegenerable),
		humanBytes(r.Totals.Referenced), humanBytes(r.Totals.Unknown), r.CitedPaths, r.Grace)
	for _, e := range r.Efforts {
		fmt.Printf("  %-48s %-10s auto=%-5t total %-9s regenerable %-9s referenced %-9s unknown %s\n", e.Name, e.Lifecycle.State,
			e.Lifecycle.AutoEligible, humanBytes(e.Bytes.Total), humanBytes(e.Bytes.Regenerable), humanBytes(e.Bytes.Referenced), humanBytes(e.Bytes.Unknown))
	}
	for _, w := range r.Warnings {
		fmt.Printf("Warning: %s\n", w)
	}
	fmt.Println("Prune a finished effort: agent-manager storage evidence-prune --effort <dir> [--apply --reason <why> [--local-owner]]")
	return nil
}

// evidencePruneAgentHint names the approval path an agent run uses instead of
// owner authority.
const evidencePruneAgentHint = "inside an agent run, run --apply WITHOUT --local-owner: it raises an operator decision in the web-console inbox " +
	"and Agent Manager applies exactly the previewed prune once approved (the answer is delivered back to this run; " +
	"check with `agent-manager storage evidence-prune --status <decision-id>`)"

// storageEvidencePrune removes regenerable intermediates of named efforts
// (operator assertion that the work is finished) or, with no --effort, of
// auto-eligible efforts. It previews unless --apply is given. From an agent
// run, --apply asks the operator instead of removing.
func (a *App) storageEvidencePrune(args []string) error {
	fs := flag.NewFlagSet("storage evidence-prune", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	var efforts, classes multiFlag
	fs.Var(&efforts, "effort", "Effort directory under the evidence home (repeatable)")
	fs.Var(&classes, "class", "Limit to a class: platform_release, release_archive, dependency_tree, build_binary, render_frames (repeatable)")
	apply := fs.Bool("apply", false, "Remove the candidates; without it the prune is a preview")
	reason := fs.String("reason", "", "Why the named efforts are finished (required with --apply)")
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; inside an agent run, omit it and --apply requests operator approval")
	status := fs.String("status", "", "Show an approval request raised by an agent's --apply")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *status != "" {
		return a.storageEvidencePruneStatus(*status, *jsonOut)
	}
	if *apply && (strings.TrimSpace(*reason) == "" || len(*reason) > 512) {
		return fmt.Errorf("--apply requires --reason with 1-512 bytes")
	}
	api := a.services.Maintenance.api.WithTimeout(30 * time.Minute)
	agentRun := strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != ""
	if *localOwner {
		if agentRun {
			return fmt.Errorf("--local-owner is unavailable inside an identified agent run; %s", evidencePruneAgentHint)
		}
		owner, err := localOwnerAPI(api)
		if err != nil {
			return err
		}
		api = owner
	}
	request := map[string]any{"efforts": []string(efforts), "classes": []string(classes), "dry_run": !*apply, "reason": *reason}
	if session := strings.TrimSpace(os.Getenv("WC_WEB_CONSOLE_SESSION_ID")); session != "" {
		request["web_console_session_id"] = session
	}
	payload, _ := json.Marshal(request)
	body, err := api.Request("POST", "/api/v1/storage/evidence/prune", nil, payload)
	if err != nil {
		if *apply && !*localOwner && !agentRun {
			return fmt.Errorf("%w (an owner runs --apply with --local-owner; %s)", apiError(body, err), evidencePruneAgentHint)
		}
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var pending struct {
		Status        string `json:"status"`
		DecisionID    string `json:"decision_id"`
		PreviewBytes  int64  `json:"preview_bytes"`
		PreviewFiles  int    `json:"preview_files"`
		PreviewPaths  int    `json:"preview_paths"`
		Message       string `json:"message"`
		StatusCommand string `json:"status_command"`
		PublishError  string `json:"publish_error"`
		Await         struct {
			ParkCommand string `json:"park_command"`
		} `json:"await"`
	}
	if json.Unmarshal(body, &pending) == nil && pending.Status != "" {
		switch pending.Status {
		case "approval_pending":
			fmt.Printf("Operator approval requested: decision %s\n", pending.DecisionID)
			fmt.Printf("Would remove %d paths, %d files, %s once approved.\n", pending.PreviewPaths, pending.PreviewFiles, humanBytes(pending.PreviewBytes))
			fmt.Println(pending.Message)
			if pending.PublishError != "" {
				fmt.Printf("Warning: the decision is recorded but its notification failed: %s\n", pending.PublishError)
			}
			fmt.Printf("Check: %s\n", pending.StatusCommand)
			fmt.Printf("Optional park until answered: %s\n", pending.Await.ParkCommand)
		default:
			fmt.Println(pending.Message)
		}
		return nil
	}
	var receipt struct {
		Mode         string `json:"mode"`
		DryRun       bool   `json:"dryRun"`
		RemovedBytes int64  `json:"removedBytes"`
		Removed      []struct {
			Path  string `json:"path"`
			Class string `json:"class"`
			Bytes int64  `json:"bytes"`
		} `json:"removed"`
		Skipped []struct {
			Effort string `json:"effort"`
			Path   string `json:"path"`
			Reason string `json:"reason"`
		} `json:"skipped"`
		Errors      []string `json:"errors"`
		ReceiptPath string   `json:"receiptPath"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return fmt.Errorf("decode prune receipt: %w", err)
	}
	verb := "Would remove"
	if !receipt.DryRun {
		verb = "Removed"
	}
	for _, r := range receipt.Removed {
		fmt.Printf("  %-16s %9s %s\n", r.Class, humanBytes(r.Bytes), r.Path)
	}
	for _, s := range receipt.Skipped {
		fmt.Printf("  kept %s %s: %s\n", s.Effort, s.Path, s.Reason)
	}
	for _, e := range receipt.Errors {
		fmt.Printf("  error: %s\n", e)
	}
	fmt.Printf("%s %d paths, %s (%s)\n", verb, len(receipt.Removed), humanBytes(receipt.RemovedBytes), receipt.Mode)
	if receipt.ReceiptPath != "" {
		fmt.Printf("Receipt: %s\n", receipt.ReceiptPath)
	}
	return nil
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// storageEvidencePruneStatus shows an agent's approval request.
func (a *App) storageEvidencePruneStatus(id string, jsonOut bool) error {
	body, err := a.services.Maintenance.api.Request("GET", "/api/v1/storage/evidence/prune/requests/"+url.PathEscape(strings.TrimSpace(id)), nil, nil)
	if err != nil {
		return apiError(body, err)
	}
	if jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var st struct {
		DecisionID   string   `json:"decision_id"`
		State        string   `json:"state"`
		Efforts      []string `json:"efforts"`
		Classes      []string `json:"classes"`
		PreviewBytes int64    `json:"preview_bytes"`
		PreviewFiles int      `json:"preview_files"`
		ResolvedBy   string   `json:"resolved_by"`
		ReceiptPath  string   `json:"receipt_path"`
		ApplyError   string   `json:"apply_error"`
	}
	if err := json.Unmarshal(body, &st); err != nil {
		return fmt.Errorf("decode prune request status: %w", err)
	}
	classes := strings.Join(st.Classes, ",")
	if classes == "" {
		classes = "all"
	}
	fmt.Printf("Decision %s: %s (efforts %s, classes %s, preview %s in %d files)\n", st.DecisionID, st.State, strings.Join(st.Efforts, ","), classes, humanBytes(st.PreviewBytes), st.PreviewFiles)
	if st.ResolvedBy != "" {
		fmt.Printf("Answered by: %s\n", st.ResolvedBy)
	}
	if st.ReceiptPath != "" {
		fmt.Printf("Receipt: %s\n", st.ReceiptPath)
	}
	if st.ApplyError != "" {
		fmt.Printf("Apply error: %s\n", st.ApplyError)
	}
	return nil
}
