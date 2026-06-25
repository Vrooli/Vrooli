// Package cmd provides the migrate-workshop CLI command.
//
// This command converts existing backlog items from the old
// clarify/suggest/enhance folder structure to the new workshop/plan.md
// structure. It is a one-time migration intended to be run after the
// workshop feature lands.
package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// backlogKinds lists every kind directory that may contain items.
var backlogKinds = []string{"ideas", "research", "fix", "execute", "chore"}

// ---------- JSON types for OLD format ----------

type oldSpec struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type oldQuestion struct {
	ID         string   `json:"id"`
	Category   string   `json:"category,omitempty"`
	Importance string   `json:"importance,omitempty"`
	Question   string   `json:"question"`
	Options    []string `json:"options,omitempty"`
	Context    string   `json:"context,omitempty"`
	Answer     any      `json:"answer"` // string or null
}

type oldQuestionsFile struct {
	Version     any           `json:"version,omitempty"` // int or string in legacy data
	GeneratedAt string        `json:"generated_at,omitempty"`
	Questions   []oldQuestion `json:"questions"`
}

type oldSuggestion struct {
	ID              string `json:"id"`
	Suggestion      string `json:"suggestion"`
	Details         string `json:"details,omitempty"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

type oldSuggestionsFile struct {
	Suggestions    []oldSuggestion `json:"suggestions"`
	GeneratedAt    string          `json:"generated_at,omitempty"`
	GeneratedAtAlt string          `json:"generatedAt,omitempty"`
}

// ---------- JSON types for NEW format ----------

type workshopReadiness struct {
	ProblemClarity int `json:"problem_clarity"`
	ScopeDefined   int `json:"scope_defined"`
	ApproachSolid  int `json:"approach_solid"`
	Testable       int `json:"testable"`
	RiskAwareness  int `json:"risk_awareness"`
}

type workshopOption struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Rationale string `json:"rationale"`
}

type workshopItem struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Topic    string           `json:"topic,omitempty"`
	Text     string           `json:"text,omitempty"`
	Context  string           `json:"context,omitempty"`
	Options  []workshopOption `json:"options,omitempty"`
	Selected *string          `json:"selected"`
	Freeform *string          `json:"freeform"`
	Notes    *string          `json:"notes"`
}

type workshopRound struct {
	Round       int               `json:"round"`
	GeneratedAt string            `json:"generated_at"`
	Readiness   workshopReadiness `json:"readiness"`
	Items       []workshopItem    `json:"items"`
	PlanUpdates string            `json:"plan_updates"`
}

// ---------- Migration options ----------

// MigrateWorkshopOptions holds the flags for the migration command.
type MigrateWorkshopOptions struct {
	Root   string
	DryRun bool
}

// RunMigrateWorkshop is the entry point called from the CLI wiring.
func RunMigrateWorkshop(opts MigrateWorkshopOptions) error {
	root := opts.Root
	if root == "" {
		root = "scenarios/swarm-manager"
	}

	// Verify root exists.
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("root directory %q not found: %w", root, err)
	}

	migrated := 0
	skipped := 0
	var errs []string

	for _, kind := range backlogKinds {
		kindDir := filepath.Join(root, kind)
		entries, err := os.ReadDir(kindDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", kindDir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			itemDir := filepath.Join(kindDir, entry.Name())
			did, err := migrateItem(itemDir, kind, opts.DryRun)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s/%s: %v", kind, entry.Name(), err))
				continue
			}
			if did {
				migrated++
			} else {
				skipped++
			}
		}
	}

	fmt.Printf("\nMigration complete: %d migrated, %d skipped (already migrated or no data)\n", migrated, skipped)
	if len(errs) > 0 {
		fmt.Printf("Errors (%d):\n", len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d item(s) failed", len(errs))
	}
	return nil
}

// migrateItem migrates a single backlog item. Returns true if it performed
// work (or would have in dry-run mode).
func migrateItem(itemDir string, kind string, dryRun bool) (bool, error) {
	specPath := filepath.Join(itemDir, "spec.json")
	if _, err := os.Stat(specPath); err != nil {
		// No spec.json → skip silently (not a real item).
		return false, nil
	}

	workshopExists := dirExists(filepath.Join(itemDir, "workshop"))
	planExists := fileExists(filepath.Join(itemDir, "plan.md"))

	clarifyPath := filepath.Join(itemDir, "clarify", "questions.json")
	suggestPath := filepath.Join(itemDir, "suggest", "suggestions.json")
	enhancePath := filepath.Join(itemDir, "enhance", "summary.md")

	hasClarify := fileExists(clarifyPath)
	hasSuggest := fileExists(suggestPath)
	hasEnhance := fileExists(enhancePath)

	// If fully migrated (workshop exists, plan exists, no old dirs), skip.
	if workshopExists && planExists && !hasClarify && !hasSuggest && !hasEnhance {
		return false, nil
	}

	// Workshop already exists but no plan and no old dirs — create plan stub.
	if !hasClarify && !hasSuggest && !hasEnhance && workshopExists && !planExists {
		return migrateWorkshopOnlyStub(itemDir, dryRun)
	}

	if !hasClarify && !hasSuggest && !hasEnhance {
		if kind != "ideas" {
			return migrateNonIdeaStub(itemDir, dryRun)
		}
		// Idea with no refinement data — create empty workshop/.
		return migrateEmptyWorkshop(itemDir, dryRun)
	}

	itemName := filepath.Base(itemDir)
	prefix := modePrefix(dryRun)
	fmt.Printf("%s migrating %s/%s\n", prefix, kind, itemName)

	if err := migratePlan(itemDir, enhancePath, prefix, hasEnhance, planExists, dryRun); err != nil {
		return false, err
	}

	if err := migrateWorkshopRounds(itemDir, clarifyPath, suggestPath, prefix, hasClarify, hasSuggest, hasEnhance, workshopExists, dryRun); err != nil {
		return false, err
	}

	if err := backupOldDirs(itemDir, prefix, dryRun); err != nil {
		return false, err
	}

	if err := removeOldDirs(itemDir, prefix, dryRun); err != nil {
		return false, err
	}

	return true, nil
}

// migrateWorkshopOnlyStub creates a plan.md stub from spec.json for an item
// that already has a workshop/ directory but no plan and no legacy data.
func migrateWorkshopOnlyStub(itemDir string, dryRun bool) (bool, error) {
	if !dryRun {
		spec, err := readSpec(itemDir)
		if err == nil {
			planContent := fmt.Sprintf("# Implementation Plan: %s\n\n## Purpose\n%s\n", spec.Title, spec.Description)
			_ = os.WriteFile(filepath.Join(itemDir, "plan.md"), []byte(planContent), 0o644)
		}
	}
	fmt.Printf("  [complete] %s: created plan.md stub\n", filepath.Base(itemDir))
	return true, nil
}

// migratePlan performs Step 1: create plan.md from enhance/summary.md when
// present, otherwise a stub from spec.json. No-op when plan.md already exists.
func migratePlan(itemDir, enhancePath, prefix string, hasEnhance, planExists, dryRun bool) error {
	if planExists {
		return nil
	}
	if hasEnhance {
		planDst := filepath.Join(itemDir, "plan.md")
		fmt.Printf("  %s copy enhance/summary.md → plan.md\n", prefix)
		if !dryRun {
			data, err := os.ReadFile(enhancePath)
			if err != nil {
				return fmt.Errorf("read enhance/summary.md: %w", err)
			}
			if err := os.WriteFile(planDst, data, 0o644); err != nil {
				return fmt.Errorf("write plan.md: %w", err)
			}
		}
		return nil
	}
	// No enhance but old data exists — create plan stub from spec.
	spec, specErr := readSpec(itemDir)
	if specErr == nil {
		fmt.Printf("  %s create plan.md stub from spec.json\n", prefix)
		if !dryRun {
			planContent := fmt.Sprintf("# Implementation Plan: %s\n\n## Purpose\n%s\n", spec.Title, spec.Description)
			_ = os.WriteFile(filepath.Join(itemDir, "plan.md"), []byte(planContent), 0o644)
		}
	}
	return nil
}

// migrateWorkshopRounds performs Step 2: convert clarify/suggest data into
// workshop round files, or create an empty workshop/ directory. No-op when a
// workshop/ directory already exists.
func migrateWorkshopRounds(itemDir, clarifyPath, suggestPath, prefix string, hasClarify, hasSuggest, hasEnhance, workshopExists, dryRun bool) error {
	if workshopExists {
		return nil
	}

	roundNum := 1

	if hasClarify {
		round, err := clarifyToRound(clarifyPath, roundNum, hasSuggest, hasEnhance)
		if err != nil {
			return fmt.Errorf("convert clarify: %w", err)
		}
		if err := writeRound(itemDir, prefix, roundNum, "clarify/questions.json", round, dryRun); err != nil {
			return err
		}
		roundNum++
	}

	if hasSuggest {
		round, err := suggestToRound(suggestPath, roundNum, hasClarify, hasEnhance)
		if err != nil {
			return fmt.Errorf("convert suggest: %w", err)
		}
		if err := writeRound(itemDir, prefix, roundNum, "suggest/suggestions.json", round, dryRun); err != nil {
			return err
		}
	}

	// If neither clarify nor suggest but enhance exists, still create the
	// workshop directory (empty).
	if !hasClarify && !hasSuggest {
		fmt.Printf("  %s create workshop/ (empty)\n", prefix)
		if !dryRun {
			if err := os.MkdirAll(filepath.Join(itemDir, "workshop"), 0o755); err != nil {
				return fmt.Errorf("mkdir workshop: %w", err)
			}
		}
	}
	return nil
}

// writeRound logs and (unless dry-run) writes a single workshop round file.
func writeRound(itemDir, prefix string, roundNum int, sourceLabel string, round *workshopRound, dryRun bool) error {
	roundFile := fmt.Sprintf("round-%03d.json", roundNum)
	fmt.Printf("  %s create workshop/%s from %s (%d items)\n", prefix, roundFile, sourceLabel, len(round.Items))
	if !dryRun {
		if err := writeWorkshopRound(itemDir, roundFile, round); err != nil {
			return err
		}
	}
	return nil
}

// backupOldDirs performs Step 3: copy each existing legacy dir into
// .swarm/pre-workshop-migration/.
func backupOldDirs(itemDir, prefix string, dryRun bool) error {
	backupBase := filepath.Join(itemDir, ".swarm", "pre-workshop-migration")
	for _, dir := range []string{"clarify", "suggest", "enhance"} {
		src := filepath.Join(itemDir, dir)
		if !dirExists(src) {
			continue
		}
		dst := filepath.Join(backupBase, dir)
		fmt.Printf("  %s backup %s/ → .swarm/pre-workshop-migration/%s/\n", prefix, dir, dir)
		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return fmt.Errorf("mkdir backup: %w", err)
			}
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("backup %s: %w", dir, err)
			}
		}
	}
	return nil
}

// removeOldDirs performs Step 4: delete each existing legacy dir.
func removeOldDirs(itemDir, prefix string, dryRun bool) error {
	for _, dir := range []string{"clarify", "suggest", "enhance"} {
		src := filepath.Join(itemDir, dir)
		if !dirExists(src) {
			continue
		}
		fmt.Printf("  %s remove %s/\n", prefix, dir)
		if !dryRun {
			if err := os.RemoveAll(src); err != nil {
				return fmt.Errorf("remove %s: %w", dir, err)
			}
		}
	}
	return nil
}

// migrateNonIdeaStub creates a plan.md stub for non-idea items that have no
// refinement data.
func migrateNonIdeaStub(itemDir string, dryRun bool) (bool, error) {
	planPath := filepath.Join(itemDir, "plan.md")
	if fileExists(planPath) {
		return false, nil
	}

	spec, err := readSpec(itemDir)
	if err != nil {
		return false, err
	}

	itemName := filepath.Base(itemDir)
	kindDir := filepath.Base(filepath.Dir(itemDir))
	prefix := modePrefix(dryRun)
	fmt.Printf("%s creating plan.md stub for %s/%s\n", prefix, kindDir, itemName)

	content := fmt.Sprintf("# %s\n\n%s\n", spec.Title, spec.Description)
	if !dryRun {
		if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
			return false, fmt.Errorf("write plan.md: %w", err)
		}
	}
	return true, nil
}

// migrateEmptyWorkshop creates an empty workshop/ directory for idea items
// that have no refinement data.
func migrateEmptyWorkshop(itemDir string, dryRun bool) (bool, error) {
	workshopDir := filepath.Join(itemDir, "workshop")
	if dirExists(workshopDir) {
		return false, nil
	}

	itemName := filepath.Base(itemDir)
	kindDir := filepath.Base(filepath.Dir(itemDir))
	prefix := modePrefix(dryRun)
	fmt.Printf("%s creating empty workshop/ for %s/%s\n", prefix, kindDir, itemName)

	if !dryRun {
		if err := os.MkdirAll(workshopDir, 0o755); err != nil {
			return false, fmt.Errorf("mkdir workshop: %w", err)
		}
	}
	return true, nil
}

// ---------- Conversion helpers ----------

func clarifyToRound(path string, roundNum int, hasSuggest, hasEnhance bool) (*workshopRound, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var qf oldQuestionsFile
	if err := json.Unmarshal(data, &qf); err != nil {
		return nil, fmt.Errorf("parse questions.json: %w", err)
	}

	items := make([]workshopItem, 0, len(qf.Questions))
	answeredCount := 0
	for _, q := range qf.Questions {
		answer := answerString(q.Answer)

		// Build options from the old question options.
		var opts []workshopOption
		for i, opt := range q.Options {
			key := string(rune('A' + i))
			opts = append(opts, workshopOption{Key: key, Label: opt, Rationale: ""})
		}
		// If the answer doesn't match any option, it's a freeform "Other" response.
		var selected *string
		var freeform *string
		if answer != "" {
			answeredCount++
			matched := false
			for i, opt := range q.Options {
				if opt == answer {
					key := string(rune('A' + i))
					selected = &key
					matched = true
					break
				}
			}
			if !matched {
				other := "__other__"
				selected = &other
				freeform = &answer
			}
		}

		items = append(items, workshopItem{
			ID:       q.ID,
			Type:     "decision",
			Topic:    q.Question,
			Context:  q.Context,
			Options:  opts,
			Selected: selected,
			Freeform: freeform,
		})
	}

	readiness := computeReadiness(answeredCount, len(qf.Questions), hasSuggest, hasEnhance)

	genAt := qf.GeneratedAt
	if genAt == "" {
		genAt = time.Now().UTC().Format(time.RFC3339)
	}

	return &workshopRound{
		Round:       roundNum,
		GeneratedAt: genAt,
		Readiness:   readiness,
		Items:       items,
		PlanUpdates: "Migrated from clarify/suggest workflow",
	}, nil
}

func suggestToRound(path string, roundNum int, hasClarify, hasEnhance bool) (*workshopRound, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sf oldSuggestionsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse suggestions.json: %w", err)
	}

	items := make([]workshopItem, 0, len(sf.Suggestions))
	for _, s := range sf.Suggestions {
		// Map old suggestion to a decision with Accept/Reject options.
		opts := []workshopOption{
			{Key: "A", Label: "Accept: " + s.Suggestion, Rationale: s.Details},
			{Key: "B", Label: "Reject", Rationale: "Do not implement this suggestion"},
		}
		var selected *string
		var notes *string
		switch mapSuggestionStatus(s.Status) {
		case "accepted":
			a := "A"
			selected = &a
		case "rejected":
			b := "B"
			selected = &b
		}
		noteText := s.Notes
		if noteText == "" && s.RejectionReason != "" {
			noteText = s.RejectionReason
		}
		if noteText != "" {
			notes = &noteText
		}
		items = append(items, workshopItem{
			ID:       s.ID,
			Type:     "decision",
			Topic:    s.Suggestion,
			Context:  s.Details,
			Options:  opts,
			Selected: selected,
			Notes:    notes,
		})
	}

	genAt := sf.GeneratedAt
	if genAt == "" {
		genAt = sf.GeneratedAtAlt
	}
	if genAt == "" {
		genAt = time.Now().UTC().Format(time.RFC3339)
	}

	// For the suggestion round, readiness reflects the combined state.
	readiness := computeReadiness(0, 0, hasClarify, hasEnhance)
	// Boost scores based on whether all decisions are resolved.
	allDecided := true
	for _, item := range items {
		if item.Selected == nil || *item.Selected == "" {
			allDecided = false
			break
		}
	}
	if allDecided && len(items) > 0 {
		readiness.ScopeDefined = max(readiness.ScopeDefined, 2)
	}

	return &workshopRound{
		Round:       roundNum,
		GeneratedAt: genAt,
		Readiness:   readiness,
		Items:       items,
		PlanUpdates: "Migrated from clarify/suggest workflow",
	}, nil
}

func computeReadiness(answeredCount, totalQuestions int, hasSuggest, hasEnhance bool) workshopReadiness {
	// All questions answered + all suggestions decided + enhance exists
	allAnswered := totalQuestions > 0 && answeredCount == totalQuestions

	if allAnswered && hasSuggest && hasEnhance {
		return workshopReadiness{
			ProblemClarity: 2,
			ScopeDefined:   2,
			ApproachSolid:  2,
			Testable:       1,
			RiskAwareness:  1,
		}
	}

	if allAnswered {
		return workshopReadiness{
			ProblemClarity: 2,
			ScopeDefined:   1,
			ApproachSolid:  0,
			Testable:       0,
			RiskAwareness:  0,
		}
	}

	// No refinement data or partially answered.
	return workshopReadiness{}
}

func mapSuggestionStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "accepted":
		return "accepted"
	case "rejected":
		return "rejected"
	default:
		return "pending"
	}
}

func answerString(v any) string {
	if v == nil {
		return ""
	}
	switch a := v.(type) {
	case string:
		return a
	default:
		b, _ := json.Marshal(a)
		return string(b)
	}
}

// ---------- File I/O helpers ----------

func writeWorkshopRound(itemDir, filename string, round *workshopRound) error {
	workshopDir := filepath.Join(itemDir, "workshop")
	if err := os.MkdirAll(workshopDir, 0o755); err != nil {
		return fmt.Errorf("mkdir workshop: %w", err)
	}
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal round: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(workshopDir, filename), data, 0o644)
}

func readSpec(itemDir string) (*oldSpec, error) {
	data, err := os.ReadFile(filepath.Join(itemDir, "spec.json"))
	if err != nil {
		return nil, fmt.Errorf("read spec.json: %w", err)
	}
	var spec oldSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse spec.json: %w", err)
	}
	return &spec, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func modePrefix(dryRun bool) string {
	if dryRun {
		return "[dry-run]"
	}
	return "[migrate]"
}
