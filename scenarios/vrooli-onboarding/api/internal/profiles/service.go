// Package profiles owns the bounded, data-driven onboarding profile language.
// Profiles are metadata: they can recommend registered capabilities, but they
// cannot execute code, carry secrets, or grant permissions.
package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SupportedSchemaVersion = "1.0.0"
	MaxExpressionDepth     = 16
	MaxConditionNodes      = 5000
	MaxQuestions           = 250
)

type Scenario struct {
	Name      string
	Resources []string
}

type Profile struct {
	SchemaVersion          string     `json:"schemaVersion"`
	ID                     string     `json:"id"`
	Version                string     `json:"version"`
	Default                bool       `json:"default,omitempty"`
	TitleKey               string     `json:"titleKey"`
	DescriptionKey         string     `json:"descriptionKey"`
	Owner                  string     `json:"owner"`
	CompatibleCatalogMajor int        `json:"compatibleCatalogMajor"`
	Questions              []Question `json:"questions"`
	Rules                  []Rule     `json:"rules"`
	ManualSelection        Manual     `json:"manualSelection"`
	Provenance             Provenance `json:"provenance"`
}

type Manual struct {
	Available bool `json:"available"`
}
type Provenance struct {
	Source   string `json:"source"`
	Revision string `json:"reviewRevision"`
}

type Question struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	PromptKey     string          `json:"promptKey"`
	Required      bool            `json:"required"`
	Options       []Option        `json:"options"`
	MinSelections int             `json:"minSelections"`
	MaxSelections int             `json:"maxSelections"`
	VisibleWhen   json.RawMessage `json:"visibleWhen"`
}

type Option struct {
	ID       string `json:"id"`
	LabelKey string `json:"labelKey"`
}
type Rule struct {
	ID        string           `json:"id"`
	When      json.RawMessage  `json:"when"`
	Recommend []Recommendation `json:"recommend"`
}
type Recommendation struct {
	CapabilityRef string   `json:"capabilityRef"`
	ScenarioRefs  []string `json:"scenarioRefs"`
	ReasonKey     string   `json:"reasonKey"`
}

type ProfileSummary struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	TitleKey           string `json:"titleKey"`
	DescriptionKey     string `json:"descriptionKey"`
	Owner              string `json:"owner"`
	ProvenanceSource   string `json:"provenanceSource"`
	ProvenanceRevision string `json:"provenanceRevision"`
}

type QuestionView struct {
	Question
	Visible bool `json:"visible"`
}
type Evaluation struct {
	Profile         ProfileSummary   `json:"profile"`
	Questions       []QuestionView   `json:"questions"`
	Recommendations []Recommendation `json:"recommendations"`
	Scenarios       []string         `json:"scenarios"`
	Resources       []string         `json:"resources"`
	Issues          []Issue          `json:"issues"`
	Valid           bool             `json:"valid"`
}
type Issue struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Service struct {
	ProfileDir func(context.Context) (string, error)
	Catalog    func(context.Context) ([]Scenario, error)
}

func (s Service) List(ctx context.Context) ([]ProfileSummary, error) {
	dir, err := s.ProfileDir(ctx)
	if err != nil {
		return nil, err
	}
	profiles, err := load(dir)
	if err != nil {
		return nil, err
	}
	result := make([]ProfileSummary, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, summary(profile))
	}
	return result, nil
}

func (s Service) Evaluate(ctx context.Context, id string, answers map[string]any, targetContext map[string]any) (Evaluation, error) {
	dir, err := s.ProfileDir(ctx)
	if err != nil {
		return Evaluation{}, err
	}
	profiles, err := load(dir)
	if err != nil {
		return Evaluation{}, err
	}
	var selected *Profile
	for i := range profiles {
		if profiles[i].ID == strings.TrimSpace(id) {
			selected = &profiles[i]
			break
		}
	}
	if selected == nil {
		return Evaluation{}, fmt.Errorf("profile %q was not found", id)
	}
	catalog, err := s.Catalog(ctx)
	if err != nil {
		return Evaluation{}, err
	}
	return evaluate(*selected, answers, targetContext, catalog)
}

func LoadDirectory(dir string) ([]Profile, error) { return load(dir) }

func load(dir string) ([]Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read onboarding profiles: %w", err)
	}
	result := make([]Profile, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("read profile %s: %w", entry.Name(), readErr)
		}
		var profile Profile
		if err := json.Unmarshal(data, &profile); err != nil {
			return nil, fmt.Errorf("decode profile %s: %w", entry.Name(), err)
		}
		if err := Validate(profile); err != nil {
			return nil, fmt.Errorf("profile %s: %w", entry.Name(), err)
		}
		if seen[profile.ID] {
			return nil, fmt.Errorf("duplicate profile id %q", profile.ID)
		}
		seen[profile.ID] = true
		result = append(result, profile)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Default != result[j].Default {
			return result[i].Default
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func Validate(profile Profile) error {
	if profile.SchemaVersion != SupportedSchemaVersion {
		return fmt.Errorf("unsupported schema version %q", profile.SchemaVersion)
	}
	for field, value := range map[string]string{"id": profile.ID, "version": profile.Version, "owner": profile.Owner} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if len(profile.Questions) > MaxQuestions {
		return fmt.Errorf("question limit exceeded: %d > %d", len(profile.Questions), MaxQuestions)
	}
	questionIDs := map[string]bool{}
	for _, question := range profile.Questions {
		if strings.TrimSpace(question.ID) == "" || questionIDs[question.ID] {
			return fmt.Errorf("question ids must be unique and non-empty")
		}
		questionIDs[question.ID] = true
		switch question.Type {
		case "multi-select", "single-select", "text", "boolean":
		default:
			return fmt.Errorf("question %q has unsupported type %q", question.ID, question.Type)
		}
		optionIDs := map[string]bool{}
		for _, option := range question.Options {
			if option.ID == "" || optionIDs[option.ID] {
				return fmt.Errorf("question %q has duplicate or empty option", question.ID)
			}
			optionIDs[option.ID] = true
		}
		if question.MinSelections < 0 || question.MaxSelections < 0 || (question.MaxSelections > 0 && question.MinSelections > question.MaxSelections) {
			return fmt.Errorf("question %q has invalid selection bounds", question.ID)
		}
		if len(question.VisibleWhen) > 0 {
			if err := validateCondition(question.VisibleWhen, 0, new(int)); err != nil {
				return fmt.Errorf("question %q visibility: %w", question.ID, err)
			}
		}
	}
	ruleIDs := map[string]bool{}
	for _, rule := range profile.Rules {
		if rule.ID == "" || ruleIDs[rule.ID] {
			return errors.New("rule ids must be unique and non-empty")
		}
		ruleIDs[rule.ID] = true
		if len(rule.When) > 0 {
			if err := validateCondition(rule.When, 0, new(int)); err != nil {
				return fmt.Errorf("rule %q: %w", rule.ID, err)
			}
		}
		for _, recommendation := range rule.Recommend {
			if strings.TrimSpace(recommendation.CapabilityRef) == "" && len(recommendation.ScenarioRefs) == 0 {
				return fmt.Errorf("rule %q contains an empty recommendation", rule.ID)
			}
		}
	}
	return nil
}

func evaluate(profile Profile, answers, targetContext map[string]any, catalog []Scenario) (Evaluation, error) {
	result := Evaluation{Profile: summary(profile), Valid: true}
	for _, question := range profile.Questions {
		visible := true
		if len(question.VisibleWhen) > 0 {
			var err error
			visible, err = matches(question.VisibleWhen, answers, targetContext, 0, new(int))
			if err != nil {
				return Evaluation{}, err
			}
		}
		if visible {
			result.Questions = append(result.Questions, QuestionView{Question: question, Visible: true})
			validateAnswer(&result, question, answers[question.ID])
		}
	}
	for _, rule := range profile.Rules {
		matched := true
		if len(rule.When) > 0 {
			var err error
			matched, err = matches(rule.When, answers, targetContext, 0, new(int))
			if err != nil {
				return Evaluation{}, err
			}
		}
		if matched {
			result.Recommendations = append(result.Recommendations, rule.Recommend...)
		}
	}
	seenScenario := map[string]bool{}
	seenResource := map[string]bool{}
	known := map[string]Scenario{}
	for _, item := range catalog {
		known[item.Name] = item
	}
	for _, recommendation := range result.Recommendations {
		for _, name := range recommendation.ScenarioRefs {
			if !seenScenario[name] {
				seenScenario[name] = true
				result.Scenarios = append(result.Scenarios, name)
			}
			if item, ok := known[name]; ok {
				for _, resource := range item.Resources {
					if !seenResource[resource] {
						seenResource[resource] = true
						result.Resources = append(result.Resources, resource)
					}
				}
			} else {
				result.Issues = append(result.Issues, Issue{Field: "recommendation", Code: "unknown_scenario", Message: fmt.Sprintf("profile recommends unavailable scenario %q", name)})
			}
		}
	}
	sort.Strings(result.Scenarios)
	sort.Strings(result.Resources)
	if len(result.Issues) > 0 {
		result.Valid = false
	}
	return result, nil
}

func validateAnswer(result *Evaluation, question Question, raw any) {
	if raw == nil {
		if question.Required {
			result.Issues = append(result.Issues, Issue{Field: question.ID, Code: "required", Message: "answer is required"})
		}
		return
	}
	allowed := map[string]bool{}
	for _, option := range question.Options {
		allowed[option.ID] = true
	}
	add := func(code, message string) {
		result.Issues = append(result.Issues, Issue{Field: question.ID, Code: code, Message: message})
	}
	switch question.Type {
	case "multi-select":
		values, ok := raw.([]any)
		if !ok {
			add("type", "expected an array of selections")
			return
		}
		if question.MinSelections > 0 && len(values) < question.MinSelections {
			add("min_selections", fmt.Sprintf("choose at least %d option(s)", question.MinSelections))
		}
		if question.MaxSelections > 0 && len(values) > question.MaxSelections {
			add("max_selections", fmt.Sprintf("choose at most %d option(s)", question.MaxSelections))
		}
		for _, value := range values {
			name, ok := value.(string)
			if !ok || !allowed[name] {
				add("option", fmt.Sprintf("unknown option %v", value))
			}
		}
	case "single-select":
		value, ok := raw.(string)
		if !ok || !allowed[value] {
			add("option", "choose one supported option")
		}
	case "text":
		if value, ok := raw.(string); !ok || strings.TrimSpace(value) == "" {
			add("value", "enter a value")
		}
	case "boolean":
		if _, ok := raw.(bool); !ok {
			add("type", "expected true or false")
		}
	}
}

func summary(profile Profile) ProfileSummary {
	return ProfileSummary{ID: profile.ID, Version: profile.Version, TitleKey: profile.TitleKey, DescriptionKey: profile.DescriptionKey, Owner: profile.Owner, ProvenanceSource: profile.Provenance.Source, ProvenanceRevision: profile.Provenance.Revision}
}

func validateCondition(raw json.RawMessage, depth int, nodes *int) error {
	if depth > MaxExpressionDepth {
		return errors.New("condition depth exceeds supported bound")
	}
	*nodes++
	if *nodes > MaxConditionNodes {
		return errors.New("condition node limit exceeded")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("invalid condition: %w", err)
	}
	object, ok := value.(map[string]any)
	if !ok || len(object) != 1 {
		return errors.New("condition must contain exactly one operator")
	}
	for operator, child := range object {
		switch operator {
		case "all", "any":
			items, ok := child.([]any)
			if !ok {
				return fmt.Errorf("%s expects an array", operator)
			}
			for _, item := range items {
				encoded, _ := json.Marshal(item)
				if err := validateCondition(encoded, depth+1, nodes); err != nil {
					return err
				}
			}
		case "not":
			encoded, _ := json.Marshal(child)
			if err := validateCondition(encoded, depth+1, nodes); err != nil {
				return err
			}
		case "contains", "eq", "in":
			if _, ok := child.(map[string]any); !ok {
				return fmt.Errorf("%s expects an object", operator)
			}
		default:
			return fmt.Errorf("unsupported condition operator %q", operator)
		}
	}
	return nil
}

func matches(raw json.RawMessage, answers, targetContext map[string]any, depth int, nodes *int) (bool, error) {
	if err := validateCondition(raw, depth, nodes); err != nil {
		return false, err
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, err
	}
	for operator, child := range value {
		switch operator {
		case "all":
			var result = true
			for _, item := range child.([]any) {
				encoded, _ := json.Marshal(item)
				matched, err := matches(encoded, answers, targetContext, depth+1, nodes)
				if err != nil {
					return false, err
				}
				result = result && matched
			}
			return result, nil
		case "any":
			for _, item := range child.([]any) {
				encoded, _ := json.Marshal(item)
				matched, err := matches(encoded, answers, targetContext, depth+1, nodes)
				if err != nil {
					return false, err
				}
				if matched {
					return true, nil
				}
			}
			return false, nil
		case "not":
			encoded, _ := json.Marshal(child)
			matched, err := matches(encoded, answers, targetContext, depth+1, nodes)
			return !matched, err
		case "contains":
			return compareContains(child, answers, targetContext)
		case "eq":
			return compareEq(child, answers, targetContext)
		case "in":
			return compareIn(child, answers, targetContext)
		}
	}
	return false, nil
}

func operand(value any, answers, targetContext map[string]any) any {
	if key, ok := value.(string); ok {
		if answer, exists := answers[key]; exists {
			return answer
		}
		if contextValue, exists := targetContext[key]; exists {
			return contextValue
		}
	}
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	if answer, ok := object["answer"].(string); ok {
		return answers[answer]
	}
	if contextKey, ok := object["context"].(string); ok {
		return targetContext[contextKey]
	}
	if setting, ok := object["setting"].(string); ok {
		return targetContext[setting]
	}
	return value
}
func compareContains(value any, answers, context map[string]any) (bool, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return false, errors.New("contains expects an object")
	}
	current := operand(object["answer"], answers, context)
	wanted := object["value"]
	list, ok := current.([]any)
	if ok {
		for _, item := range list {
			if fmt.Sprint(item) == fmt.Sprint(wanted) {
				return true, nil
			}
		}
		return false, nil
	}
	return strings.Contains(fmt.Sprint(current), fmt.Sprint(wanted)), nil
}
func compareEq(value any, answers, context map[string]any) (bool, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return false, errors.New("eq expects an object")
	}
	return fmt.Sprint(operand(object["answer"], answers, context)) == fmt.Sprint(object["value"]), nil
}
func compareIn(value any, answers, context map[string]any) (bool, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return false, errors.New("in expects an object")
	}
	current := fmt.Sprint(operand(object["answer"], answers, context))
	values, ok := object["values"].([]any)
	if !ok {
		return false, errors.New("in expects values")
	}
	for _, value := range values {
		if current == fmt.Sprint(value) {
			return true, nil
		}
	}
	return false, nil
}
