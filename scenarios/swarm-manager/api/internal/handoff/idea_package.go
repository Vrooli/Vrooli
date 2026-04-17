// Package handoff builds durable execution handoff artifacts for idea backlog
// items. The package intentionally derives its output from the latest finalized
// backlog state instead of relying on agents to maintain separate handoff files
// during workshop rounds.
package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swarm-manager/internal/workshop"
	"time"
)

const (
	// SchemaVersion tracks the on-disk contract for idea handoff packages.
	SchemaVersion = "idea-handoff/v1"

	handoffDirName      = "handoff"
	briefFileName       = "brief.md"
	manifestFileName    = "manifest.json"
	sourceIndexFileName = "source-index.json"
)

// BuildRequest contains the finalized backlog context required to build an
// idea handoff package.
type BuildRequest struct {
	BacklogKind             string
	BacklogName             string
	BacklogTitle            string
	BacklogDescription      string
	ItemFolder              string
	DeliverableFileName     string
	TargetScenario          string
	Operation               string
	SuggestedSteerProfileID string
	AcceptanceAllow         []string
	AcceptanceDeny          []string
	GeneratedAt             time.Time
}

// Package represents the generated idea handoff package and its primary
// artifacts.
type Package struct {
	Dir             string
	BriefPath       string
	BriefMarkdown   string
	ManifestPath    string
	Manifest        Manifest
	SourceIndexPath string
	SourceIndex     SourceIndex
}

// Manifest is the machine-readable contract consumed by downstream tooling and
// prompts.
type Manifest struct {
	SchemaVersion           string            `json:"schema_version"`
	GeneratedAt             string            `json:"generated_at"`
	Source                  string            `json:"source"`
	BacklogKind             string            `json:"backlog_kind"`
	BacklogName             string            `json:"backlog_name"`
	BacklogTitle            string            `json:"backlog_title"`
	BacklogDescription      string            `json:"backlog_description,omitempty"`
	ItemFolder              string            `json:"item_folder"`
	TargetScenario          string            `json:"target_scenario"`
	Operation               string            `json:"operation"`
	SuggestedSteerProfileID string            `json:"suggested_steer_profile_id,omitempty"`
	DeliverablePath         string            `json:"deliverable_path"`
	ManifestPath            string            `json:"manifest_path"`
	BriefPath               string            `json:"brief_path"`
	SourceIndexPath         string            `json:"source_index_path"`
	LockedDecisions         []DecisionSummary `json:"locked_decisions,omitempty"`
	OpenDecisions           []OpenDecision    `json:"open_decisions,omitempty"`
	AcceptanceAllow         []string          `json:"acceptance_allow,omitempty"`
	AcceptanceDeny          []string          `json:"acceptance_deny,omitempty"`
	ValidationCommands      []string          `json:"validation_commands,omitempty"`
}

// DecisionSummary captures a resolved workshop decision in a compact format.
type DecisionSummary struct {
	Round         int    `json:"round"`
	ID            string `json:"id"`
	Topic         string `json:"topic"`
	SelectedKey   string `json:"selected_key"`
	SelectedLabel string `json:"selected_label,omitempty"`
	Freeform      string `json:"freeform,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

// OpenDecision captures a workshop decision that remains unresolved at the time
// the handoff package was generated.
type OpenDecision struct {
	Round   int    `json:"round"`
	ID      string `json:"id"`
	Topic   string `json:"topic"`
	Context string `json:"context,omitempty"`
}

// SourceIndex enumerates the authoritative source artifacts that were used to
// derive the package.
type SourceIndex struct {
	ItemFolder          string   `json:"item_folder"`
	PlanPath            string   `json:"plan_path"`
	SpecPath            string   `json:"spec_path"`
	NotesPath           string   `json:"notes_path,omitempty"`
	ResearchSummaryPath string   `json:"research_summary_path,omitempty"`
	ArchiveDir          string   `json:"archive_dir,omitempty"`
	WorkshopRoundPaths  []string `json:"workshop_round_paths,omitempty"`
}

// BuildIdeaPackage derives and writes the authoritative handoff package for an
// idea backlog item.
func BuildIdeaPackage(req BuildRequest) (*Package, error) {
	if strings.TrimSpace(req.BacklogKind) != "idea" {
		return nil, fmt.Errorf("idea handoff only supports backlog kind %q", strings.TrimSpace(req.BacklogKind))
	}
	itemFolder := strings.TrimSpace(req.ItemFolder)
	if itemFolder == "" {
		return nil, fmt.Errorf("item folder is required")
	}
	deliverableName := strings.TrimSpace(req.DeliverableFileName)
	if deliverableName == "" {
		deliverableName = "plan.md"
	}
	deliverablePath := filepath.Join(itemFolder, deliverableName)
	if _, err := os.Stat(deliverablePath); err != nil {
		return nil, fmt.Errorf("deliverable not available for idea handoff: %w", err)
	}

	generatedAt := req.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}

	rounds, err := workshop.LoadRounds(itemFolder)
	if err != nil {
		return nil, fmt.Errorf("load workshop rounds: %w", err)
	}
	locked, open := summarizeDecisions(rounds)

	handoffDir := filepath.Join(itemFolder, handoffDirName)
	if err := os.MkdirAll(handoffDir, 0o755); err != nil {
		return nil, fmt.Errorf("create handoff dir: %w", err)
	}

	briefPath := filepath.Join(handoffDir, briefFileName)
	manifestPath := filepath.Join(handoffDir, manifestFileName)
	sourceIndexPath := filepath.Join(handoffDir, sourceIndexFileName)

	sourceIndex := SourceIndex{
		ItemFolder:          itemFolder,
		PlanPath:            deliverablePath,
		SpecPath:            filepath.Join(itemFolder, "spec.json"),
		NotesPath:           fileIfExists(filepath.Join(itemFolder, "notes.md")),
		ResearchSummaryPath: fileIfExists(filepath.Join(itemFolder, "research", "summary.md")),
		ArchiveDir:          dirIfExists(filepath.Join(itemFolder, "archive")),
		WorkshopRoundPaths:  buildRoundPaths(itemFolder, rounds),
	}

	manifest := Manifest{
		SchemaVersion:           SchemaVersion,
		GeneratedAt:             generatedAt.Format(time.RFC3339),
		Source:                  "swarm-manager",
		BacklogKind:             strings.TrimSpace(req.BacklogKind),
		BacklogName:             strings.TrimSpace(req.BacklogName),
		BacklogTitle:            strings.TrimSpace(req.BacklogTitle),
		BacklogDescription:      strings.TrimSpace(req.BacklogDescription),
		ItemFolder:              itemFolder,
		TargetScenario:          strings.TrimSpace(req.TargetScenario),
		Operation:               strings.TrimSpace(req.Operation),
		SuggestedSteerProfileID: strings.TrimSpace(req.SuggestedSteerProfileID),
		DeliverablePath:         deliverablePath,
		ManifestPath:            manifestPath,
		BriefPath:               briefPath,
		SourceIndexPath:         sourceIndexPath,
		LockedDecisions:         locked,
		OpenDecisions:           open,
		AcceptanceAllow:         append([]string(nil), req.AcceptanceAllow...),
		AcceptanceDeny:          append([]string(nil), req.AcceptanceDeny...),
		ValidationCommands:      validationCommands(strings.TrimSpace(req.TargetScenario)),
	}

	brief := renderBrief(manifest, sourceIndex)

	if err := writeJSONFile(sourceIndexPath, sourceIndex); err != nil {
		return nil, err
	}
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return nil, err
	}
	if err := os.WriteFile(briefPath, []byte(brief), 0o644); err != nil {
		return nil, fmt.Errorf("write brief: %w", err)
	}

	return &Package{
		Dir:             handoffDir,
		BriefPath:       briefPath,
		BriefMarkdown:   brief,
		ManifestPath:    manifestPath,
		Manifest:        manifest,
		SourceIndexPath: sourceIndexPath,
		SourceIndex:     sourceIndex,
	}, nil
}

func renderBrief(manifest Manifest, sourceIndex SourceIndex) string {
	var b strings.Builder

	b.WriteString("# Idea Execution Handoff\n\n")
	b.WriteString("This package captures the finalized swarm-manager idea context for downstream ecosystem-manager execution. It is regenerated from the latest finalized backlog state when idea execution begins so downstream work starts from a stable contract rather than scattered workshop artifacts.\n\n")

	b.WriteString("## Execution Contract\n\n")
	b.WriteString(fmt.Sprintf("- Backlog item: `%s/%s`\n", manifest.BacklogKind, manifest.BacklogName))
	if manifest.BacklogTitle != "" {
		b.WriteString(fmt.Sprintf("- Title: %s\n", manifest.BacklogTitle))
	}
	if manifest.TargetScenario != "" {
		b.WriteString(fmt.Sprintf("- Target scenario: `%s`\n", manifest.TargetScenario))
	}
	if manifest.Operation != "" {
		b.WriteString(fmt.Sprintf("- Recommended ecosystem operation: `%s`\n", manifest.Operation))
	}
	if manifest.SuggestedSteerProfileID != "" {
		b.WriteString(fmt.Sprintf("- Recommended steer profile: `%s`\n", manifest.SuggestedSteerProfileID))
	}
	b.WriteString(fmt.Sprintf("- Item folder: `%s`\n", manifest.ItemFolder))
	b.WriteString(fmt.Sprintf("- Plan: `%s`\n", sourceIndex.PlanPath))
	b.WriteString(fmt.Sprintf("- Manifest: `%s`\n", manifest.ManifestPath))
	b.WriteString(fmt.Sprintf("- Source index: `%s`\n", manifest.SourceIndexPath))
	b.WriteString("\n")

	b.WriteString("## Downstream Requirements\n\n")
	b.WriteString("- Read `plan.md` and `manifest.json` before creating the ecosystem-manager task.\n")
	b.WriteString("- Use this `brief.md` file as the ecosystem-manager task notes.\n")
	b.WriteString("- Preserve the origin metadata so later ecosystem-manager loops can trace back to the swarm-manager source artifacts.\n\n")

	if manifest.BacklogDescription != "" {
		b.WriteString("## Product Intent\n\n")
		b.WriteString(manifest.BacklogDescription)
		b.WriteString("\n\n")
	}

	b.WriteString("## Locked Decisions\n\n")
	if len(manifest.LockedDecisions) == 0 {
		b.WriteString("- None captured in workshop state.\n\n")
	} else {
		for _, decision := range manifest.LockedDecisions {
			line := fmt.Sprintf("- Round %03d `%s`: %s", decision.Round, decision.ID, decision.Topic)
			if decision.SelectedLabel != "" {
				line += fmt.Sprintf(" -> %s", decision.SelectedLabel)
			} else if decision.SelectedKey != "" {
				line += fmt.Sprintf(" -> option %s", decision.SelectedKey)
			}
			b.WriteString(line + "\n")
			if decision.Freeform != "" {
				b.WriteString(fmt.Sprintf("  Freeform: %s\n", decision.Freeform))
			}
			if decision.Notes != "" {
				b.WriteString(fmt.Sprintf("  Notes: %s\n", decision.Notes))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Remaining Open Decisions\n\n")
	if len(manifest.OpenDecisions) == 0 {
		b.WriteString("- None.\n\n")
	} else {
		for _, decision := range manifest.OpenDecisions {
			line := fmt.Sprintf("- Round %03d `%s`: %s", decision.Round, decision.ID, decision.Topic)
			if decision.Context != "" {
				line += fmt.Sprintf(" — %s", decision.Context)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Execution Boundaries\n\n")
	if len(manifest.AcceptanceAllow) == 0 {
		b.WriteString("- acceptance_allow: none recorded\n")
	} else {
		b.WriteString("- acceptance_allow:\n")
		for _, pattern := range manifest.AcceptanceAllow {
			b.WriteString(fmt.Sprintf("  - `%s`\n", pattern))
		}
	}
	if len(manifest.AcceptanceDeny) == 0 {
		b.WriteString("- acceptance_deny: none recorded\n")
	} else {
		b.WriteString("- acceptance_deny:\n")
		for _, pattern := range manifest.AcceptanceDeny {
			b.WriteString(fmt.Sprintf("  - `%s`\n", pattern))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Validation Starting Point\n\n")
	for _, command := range manifest.ValidationCommands {
		b.WriteString(fmt.Sprintf("- `%s`\n", command))
	}
	b.WriteString("\n")

	b.WriteString("## Supporting Sources\n\n")
	b.WriteString(fmt.Sprintf("- Spec: `%s`\n", sourceIndex.SpecPath))
	if sourceIndex.NotesPath != "" {
		b.WriteString(fmt.Sprintf("- Processing notes: `%s`\n", sourceIndex.NotesPath))
	}
	if sourceIndex.ResearchSummaryPath != "" {
		b.WriteString(fmt.Sprintf("- Research summary: `%s`\n", sourceIndex.ResearchSummaryPath))
	}
	if sourceIndex.ArchiveDir != "" {
		b.WriteString(fmt.Sprintf("- Archive dir: `%s`\n", sourceIndex.ArchiveDir))
	}
	if len(sourceIndex.WorkshopRoundPaths) > 0 {
		b.WriteString("- Workshop rounds:\n")
		for _, path := range sourceIndex.WorkshopRoundPaths {
			b.WriteString(fmt.Sprintf("  - `%s`\n", path))
		}
	}
	b.WriteString("\n")

	return b.String()
}

func validationCommands(target string) []string {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	return []string{
		fmt.Sprintf("vrooli scenario status %s", target),
		fmt.Sprintf("scenario-completeness-scoring score %s", target),
		fmt.Sprintf("scenario-auditor audit %s --timeout 240", target),
	}
}
