package autofiler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/scenarios"
)

type BacklogCreator interface {
	Create(backlog.BacklogItem, backlog.CreationContext) error
}

type GoalManager interface {
	Create(goals.CreateRequest) (*goals.GoalWithScope, error)
	AddTargets(name string, targets []string) (*goals.GoalWithScope, error)
}

type Filer struct {
	backlog BacklogCreator
	goals   GoalManager
}

func NewFiler(backlogCreator BacklogCreator, goalManager GoalManager) *Filer {
	return &Filer{backlog: backlogCreator, goals: goalManager}
}

type FileOptions struct {
	Mode     Mode
	Strategy Strategy
	GoalName string
}

type FileResult struct {
	Created bool
	Item    backlog.BacklogItem
}

func (f *Filer) File(ctx context.Context, finding Finding, opts FileOptions) (FileResult, error) {
	if f.backlog == nil {
		return FileResult{}, errors.New("autofiler: backlog creator is required")
	}
	if err := validateFinding(finding); err != nil {
		return FileResult{}, err
	}
	opts = normalizeFileOptions(opts)
	item := itemForFinding(finding, opts, time.Now().UTC())
	err := f.backlog.Create(item, backlog.CreationContext{
		Context:    ctx,
		Source:     backlog.SourceAutoFiler,
		Entrypoint: string(opts.Strategy),
	})
	if err != nil {
		if errors.Is(err, backlog.ErrItemExists) {
			if err := f.ensureGoalTarget(opts.GoalName, ItemRef(item)); err != nil {
				return FileResult{}, err
			}
			return FileResult{Created: false, Item: item}, nil
		}
		return FileResult{}, err
	}
	if err := f.ensureGoalTarget(opts.GoalName, ItemRef(item)); err != nil {
		return FileResult{}, err
	}
	return FileResult{Created: true, Item: item}, nil
}

func (f *Filer) ensureGoalTarget(goalName, itemRef string) error {
	if f.goals == nil {
		return nil
	}
	goalName = strings.TrimSpace(goalName)
	if goalName == "" {
		return errors.New("autofiler: goal name is required")
	}
	if _, err := f.goals.Create(goals.CreateRequest{
		Name:        goalName,
		Title:       goalName,
		Description: "Automatically filed maintenance findings.",
		Priority:    5,
		Seeded:      true,
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("ensure auto-filer goal %q: %w", goalName, err)
	}
	if _, err := f.goals.AddTargets(goalName, []string{itemRef}); err != nil {
		return fmt.Errorf("attach %s to auto-filer goal %q: %w", itemRef, goalName, err)
	}
	return nil
}

func validateFinding(f Finding) error {
	if strings.TrimSpace(f.ID) == "" {
		return errors.New("autofiler: finding ID is required")
	}
	if strings.TrimSpace(f.Scenario) == "" {
		return errors.New("autofiler: finding scenario is required")
	}
	return nil
}

func normalizeFileOptions(opts FileOptions) FileOptions {
	if opts.Mode != ModeAutoAdd {
		opts.Mode = ModeSuggest
	}
	if strings.TrimSpace(string(opts.Strategy)) == "" {
		opts.Strategy = StrategyFeaturePending
	}
	if strings.TrimSpace(opts.GoalName) == "" {
		opts.GoalName = "automated-maintenance"
	}
	return opts
}

func itemForFinding(f Finding, opts FileOptions, now time.Time) backlog.BacklogItem {
	status := backlog.StatusSuggested
	if opts.Mode == ModeAutoAdd {
		status = backlog.StatusBacklog
	}
	name := stableItemName(f)
	title := strings.TrimSpace(f.Title)
	if title == "" {
		dimension := strings.TrimSpace(f.Dimension)
		if dimension == "" {
			dimension = "readiness"
		}
		title = fmt.Sprintf("[%s] maintenance: %s", strings.TrimSpace(f.Scenario), dimension)
	}
	timestamp := now.Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:            name,
		Title:           title,
		Description:     descriptionForFinding(f, opts),
		Status:          status,
		Kind:            backlog.KindFix,
		Priority:        priorityForSeverity(f.Severity),
		Tags:            []string{DefaultTag, string(opts.Strategy)},
		Created:         timestamp,
		Updated:         timestamp,
		FindingRef:      f.StableID(),
		SuggestedSkills: append([]string(nil), f.RecommendedSkillIDs...),
		AcceptanceAllow: []string{
			"scenarios/" + slugPathSegment(f.Scenario) + "/**",
		},
		CreatedBy: &identity.Provenance{
			Actor:  identity.TypeAgent,
			Source: Origin(opts.Strategy, f.StableID()),
		},
	}
	// Scenario-health findings already originate from the shared remediation
	// factory. Keep their outcome and acceptance verbatim so the suggest path
	// and an operator-applied preview describe the same bounded work.
	if strings.HasPrefix(f.StableID(), "srh:") {
		item.Description = strings.TrimSpace(f.Description)
		if strings.TrimSpace(f.Details) != "" {
			item.Description += "\n\n" + strings.TrimSpace(f.Details)
		}
	}
	return item
}

func descriptionForFinding(f Finding, opts FileOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Auto-filer flagged `%s` from the %s strategy.\n\n", strings.TrimSpace(f.Scenario), opts.Strategy)
	if strings.TrimSpace(f.Description) != "" {
		b.WriteString(strings.TrimSpace(f.Description))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(f.Dimension) != "" || strings.TrimSpace(string(f.Severity)) != "" {
		fmt.Fprintf(&b, "Finding: %s", strings.TrimSpace(f.Dimension))
		if strings.TrimSpace(string(f.Severity)) != "" {
			fmt.Fprintf(&b, " (%s)", strings.ToUpper(strings.TrimSpace(string(f.Severity))))
		}
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(f.Details) != "" {
		b.WriteString("Details:\n")
		b.WriteString(strings.TrimSpace(f.Details))
		b.WriteString("\n\n")
	}
	b.WriteString("Filed automatically under the backlog auto-filer policy. Refine or accept through the normal backlog flow.")
	return b.String()
}

func priorityForSeverity(sev Severity) int {
	switch sev {
	case SeverityRed:
		return 2
	case SeverityYellow:
		return 4
	default:
		return 3
	}
}

func stableItemName(f Finding) string {
	if strings.HasPrefix(f.StableID(), "srh:") {
		return scenarios.RemediationItemName(f.StableID())
	}
	scenario := slugify(f.Scenario)
	if scenario == "" {
		scenario = "scenario"
	}
	id := slugify(f.StableID())
	if id == "" {
		id = "finding"
	}
	name := "auto-filer-" + scenario + "-" + id
	if len(name) > 96 {
		name = name[:96]
		name = strings.Trim(name, "-")
	}
	return name
}

func slugPathSegment(value string) string {
	slug := slugify(value)
	if slug == "" {
		return "unknown"
	}
	return slug
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
