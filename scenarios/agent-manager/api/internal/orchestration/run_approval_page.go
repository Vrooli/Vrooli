// This file builds the operator's run-approval decision from what the run
// already holds: the task, the agent's final output and the sandbox diff. The
// cover says in plain words what would change; the page shows the work itself.

package orchestration

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/decisionpage"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// Run approval cover and page. The operator reads the push cold: the cover says
// what an approval does in plain words, and the page shows the work itself (the
// brief the agent was given, what it says it did, every changed file and the
// diff), so the change can be judged without opening Agent Manager.

const (
	runApprovalDiffBudget = 160 << 10
	runApprovalFileRows   = 40
)

// runApprovalFacts builds the decision from the run, its task when one can be
// read, and the current sandbox diff. Everything the page shows is durable run
// state, so a retry rebuilds the same event.
func (o *Orchestrator) runApprovalFacts(ctx context.Context, run *domain.Run, diff *sandbox.DiffResult) operatordecision.Facts {
	var task *domain.Task
	if o.tasks != nil && run.TaskID != uuid.Nil {
		task, _ = o.tasks.Get(ctx, run.TaskID)
	}
	files := diff.Stats.FilesChanged
	if files == 0 && len(diff.Files) > 0 {
		files = len(diff.Files)
	}
	area := operatordecision.Clip(changeArea(task, diff.Files), 120)
	count := operatordecision.Count(files, "file", "files")

	f := operatordecision.Facts{
		Subject:  fmt.Sprintf("Apply an agent's changes to %s in %s?", count, area),
		Question: fmt.Sprintf("An agent finished its task and changed %s in %s. Apply these changes to the real project?", count, area),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Apply the changes to the project"},
			{Key: "reject", Label: "Discard the changes"},
		},
		// reversible=true: an applied change can be reverted like any other; no
		// silent default is set, so nothing is applied without an explicit answer.
		Reversible: true,
		Urgency:    "normal",
	}
	if files <= 0 {
		// An empty-looking payload is not proof that no files changed.
		f.Subject = "Apply an agent's changes in " + area + "?"
		f.Question = "An agent finished its task in " + area + ", but its file count is unavailable. Apply its changes to the real project anyway?"
	}
	if run.Result != nil && run.Result.Success {
		f.Recommended = "approve"
		f.RecommendationReason = "The agent reports it finished successfully; skim the changes on this page first."
	}
	label := approvalRunLabel(run)
	var big strings.Builder
	if label != "" {
		fmt.Fprintf(&big, "The task was %q. ", label)
	}
	big.WriteString("The agent worked on a private copy of the code (a sandbox), so nothing it did has reached the real project yet. ")
	big.WriteString("Applying copies its changes into the canonical repository, the real project. Discarding throws the copy away and the project stays as it is. ")
	big.WriteString("If you do not answer, nothing is applied and the changes keep waiting.")
	f.BigPicture = big.String()

	var tech strings.Builder
	fmt.Fprintf(&tech, "Run %s", run.ID)
	if label != "" {
		fmt.Fprintf(&tech, " (%q)", label)
	}
	if task != nil {
		fmt.Fprintf(&tech, "; task %s, scope %s", task.ID, task.ScopePath)
	}
	if run.SandboxID != nil {
		fmt.Fprintf(&tech, "; sandbox %s; patch sha256 %s", run.SandboxID, diff.PatchSHA256)
	}
	tech.WriteString(".\n")
	if files > 0 {
		fmt.Fprintf(&tech, "The sandbox diff currently shows %s (+%d −%d lines).", count, diff.Stats.LinesAdded, diff.Stats.LinesRemoved)
	} else {
		tech.WriteString("The sandbox diff payload looks empty and its complete file count is unavailable.")
	}
	tech.WriteString(" Review the diff and run results before choosing. No changes are applied without an explicit approval, and this decision is not broader task or goal acceptance.")
	f.TechnicalDetails = tech.String()

	f.Explainer = operatordecision.Fit(runApprovalDiffBudget, func(budget int) string {
		return runApprovalPage(f, run, task, diff, area, files, budget)
	})
	return f
}

func runApprovalPage(f operatordecision.Facts, run *domain.Run, task *domain.Task, diff *sandbox.DiffResult, area string, files, diffBudget int) string {
	page := operatordecision.NewPage(f.Subject).Raw(decisionpage.Style).
		Lead("An agent finished a task on a private copy of the code. Its changes are below. Nothing reaches the real project until you choose.")
	rows := []operatordecision.Row{}
	if label := approvalRunLabel(run); label != "" {
		rows = append(rows, operatordecision.Row{Label: "Task", Value: label})
	}
	rows = append(rows, operatordecision.Row{Label: "Where", Value: area})
	if files > 0 {
		rows = append(rows, operatordecision.Row{Label: "Changed", Value: changeBreakdown(diff, files)},
			operatordecision.Row{Label: "Lines", Value: fmt.Sprintf("%d added, %d removed", diff.Stats.LinesAdded, diff.Stats.LinesRemoved)})
	}
	if run.EndedAt != nil {
		rows = append(rows, operatordecision.Row{Label: "Finished", Value: run.EndedAt.UTC().Format("Mon Jan 2 15:04 UTC")})
	}
	if agent := runAgentName(run); agent != "" {
		rows = append(rows, operatordecision.Row{Label: "Agent", Value: agent})
	}
	if s := run.Summary; s != nil && (s.TokensUsed > 0 || s.TurnsUsed > 0) {
		rows = append(rows, operatordecision.Row{Label: "Effort", Value: fmt.Sprintf("%d turns, %d tokens", s.TurnsUsed, s.TokensUsed)})
	}
	page.Facts(rows...)
	if task != nil && strings.TrimSpace(task.Description) != "" {
		page.Section("What the agent was asked to do", "").Raw(decisionpage.Quote(operatordecision.Clip(task.Description, 1200)))
	}
	if run.Result != nil && strings.TrimSpace(run.Result.FinalOutput) != "" {
		page.Section("What the agent says it did", "").Raw(decisionpage.Quote(operatordecision.Clip(run.Result.FinalOutput, 1500)))
	} else if run.Result != nil && !run.Result.Success {
		page.Callout("warning", "The agent did not report success.", "Read the changes closely before applying them.")
	}
	if len(diff.Files) > 0 {
		sorted := append([]sandbox.FileChange(nil), diff.Files...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].FilePath < sorted[j].FilePath })
		table := [][]string{}
		for i, file := range sorted {
			if i == runApprovalFileRows {
				table = append(table, []string{fmt.Sprintf("… and %d more", len(sorted)-i), "", ""})
				break
			}
			table = append(table, []string{file.FilePath, changeWord(file.ChangeType), fmt.Sprintf("+%d −%d", file.LinesAdded, file.LinesRemoved)})
		}
		page.Section("Files", "").Table([]string{"File", "Change", "Lines"}, table)
	}
	if diffBudget > 0 && strings.TrimSpace(diff.UnifiedDiff) != "" {
		page.Section("The changes", "Green lines are added, red lines are removed. Tap a file to open or close it.").
			Raw(decisionpage.Diff(diff.UnifiedDiff, diffBudget))
	}
	page.Options(f, map[string]string{
		"approve": "Copies these changes into the real project now. They can be reverted later like any other change.",
		"reject":  "Throws the private copy away. The real project stays as it is, and the agent's work is lost unless it runs again.",
	})
	return page.IfNoAnswer("Nothing is applied. The changes wait in the sandbox until you decide.").HTML()
}

// changeArea names, in words, the part of the project a change touches: the
// task's scope when it has one, otherwise the scenarios or packages its files
// fall under.
func changeArea(task *domain.Task, files []sandbox.FileChange) string {
	if task != nil {
		if name := projectPart(task.ScopePath); name != "" {
			return name
		}
	}
	seen := map[string]bool{}
	var parts []string
	for _, f := range files {
		if name := projectPart(f.FilePath); name != "" && !seen[name] {
			seen[name] = true
			parts = append(parts, name)
		}
	}
	sort.Strings(parts)
	switch len(parts) {
	case 0:
		return "the project"
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	}
	return fmt.Sprintf("%s and %d other parts of the project", parts[0], len(parts)-1)
}

// projectPart maps a repository-relative path to the scenario, resource or
// package it belongs to ("scenarios/agent-inbox/api" is "agent-inbox").
func projectPart(p string) string {
	p = strings.Trim(path.Clean("/"+strings.TrimSpace(p)), "/")
	segments := strings.Split(p, "/")
	if p == "" || p == "." {
		return ""
	}
	switch segments[0] {
	case "scenarios", "resources":
		if len(segments) > 1 {
			return segments[1]
		}
	case "packages":
		if len(segments) > 1 {
			return "the shared " + segments[1] + " package"
		}
	case "docs":
		return "the docs"
	}
	return ""
}

func changeBreakdown(diff *sandbox.DiffResult, files int) string {
	parts := []string{}
	if diff.Stats.FilesAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d new", diff.Stats.FilesAdded))
	}
	if diff.Stats.FilesModified > 0 {
		parts = append(parts, fmt.Sprintf("%d edited", diff.Stats.FilesModified))
	}
	if diff.Stats.FilesDeleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", diff.Stats.FilesDeleted))
	}
	text := operatordecision.Count(files, "file", "files")
	if len(parts) > 0 {
		text += ": " + strings.Join(parts, ", ")
	}
	return text
}

func changeWord(t sandbox.FileChangeType) string {
	switch t {
	case sandbox.FileChangeAdded:
		return "New"
	case sandbox.FileChangeDeleted:
		return "Deleted"
	}
	return "Edited"
}

func runAgentName(run *domain.Run) string {
	if run.ResolvedConfig == nil {
		return ""
	}
	name := string(run.ResolvedConfig.RunnerType)
	if model := strings.TrimSpace(run.ResolvedConfig.Model); model != "" {
		if name == "" {
			return model
		}
		name += " (" + model + ")"
	}
	return name
}

// approvalRunLabel uses the label copied onto the run at admission. It avoids
// a task lookup during notification and keeps this context stable across retries.
func approvalRunLabel(run *domain.Run) string {
	if run == nil {
		return ""
	}
	return operatordecision.Clip(strings.Join(strings.Fields(run.Label), " "), 140)
}
