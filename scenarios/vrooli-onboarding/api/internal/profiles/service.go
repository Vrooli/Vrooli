// Package profiles owns the bounded, data-driven onboarding profile language.
// Profiles are metadata: they can recommend registered capabilities, but they
// cannot execute code, carry secrets, or grant permissions.
package profiles

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	SupportedSchemaVersion = "1.0.0"
	MaxExpressionDepth     = 16
	MaxConditionNodes      = 5000
	MaxQuestions           = 250
	MaxOptionsPerQuestion  = 100
	MaxRules               = 500
	MaxRecommendations     = 2000
	MaxProfiles            = 100
	MaxProfileBytes        = 256 * 1024
)

var (
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
	referencePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
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
	Default       json.RawMessage `json:"default"`
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
	RuleID        string   `json:"ruleId,omitempty"`
	Key           string   `json:"key"`
	Selected      bool     `json:"selected"`
	Required      bool     `json:"required,omitempty"`
}

type ProfileSummary struct {
	ID                       string `json:"id"`
	Version                  string `json:"version"`
	Default                  bool   `json:"default,omitempty"`
	SchemaVersion            string `json:"schemaVersion"`
	CompatibleCatalogMajor   int    `json:"compatibleCatalogMajor"`
	TitleKey                 string `json:"titleKey"`
	DescriptionKey           string `json:"descriptionKey"`
	Owner                    string `json:"owner"`
	ProvenanceSource         string `json:"provenanceSource"`
	ProvenanceRevision       string `json:"provenanceRevision"`
	ManualSelectionAvailable bool   `json:"manualSelectionAvailable"`
}

type QuestionView struct {
	Question
	Visible bool `json:"visible"`
}
type Evaluation struct {
	Profile         ProfileSummary   `json:"profile"`
	Questions       []QuestionView   `json:"questions"`
	Recommendations []Recommendation `json:"recommendations"`
	Explanations    []Explanation    `json:"explanations"`
	Scenarios       []string         `json:"scenarios"`
	Resources       []string         `json:"resources"`
	Issues          []Issue          `json:"issues"`
	Digest          string           `json:"digest"`
	Valid           bool             `json:"valid"`
}

type Explanation struct {
	RuleID        string   `json:"ruleId"`
	CapabilityRef string   `json:"capabilityRef,omitempty"`
	ScenarioRefs  []string `json:"scenarioRefs,omitempty"`
	ReasonKey     string   `json:"reasonKey"`
	Selected      bool     `json:"selected"`
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
	return s.EvaluateWithManualDecisions(ctx, id, answers, targetContext, nil)
}

func (s Service) EvaluateWithManualDecisions(ctx context.Context, id string, answers map[string]any, targetContext map[string]any, manualDecisions map[string]bool) (Evaluation, error) {
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
	return evaluate(*selected, answers, targetContext, manualDecisions, catalog)
}

func LoadDirectory(dir string) ([]Profile, error) { return load(dir) }

func load(dir string) ([]Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read onboarding profiles: %w", err)
	}
	result := make([]Profile, 0, len(entries))
	seen := map[string]bool{}
	profileCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		profileCount++
		if profileCount > MaxProfiles {
			return nil, fmt.Errorf("profile catalog limit exceeded: %d > %d", profileCount, MaxProfiles)
		}
		path := filepath.Join(dir, entry.Name())
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, fmt.Errorf("stat profile %s: %w", entry.Name(), statErr)
		}
		if info.Size() > MaxProfileBytes {
			return nil, fmt.Errorf("profile %s exceeds %d byte limit", entry.Name(), MaxProfileBytes)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read profile %s: %w", entry.Name(), readErr)
		}
		var profile Profile
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&profile); err != nil {
			return nil, fmt.Errorf("decode profile %s: %w", entry.Name(), err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			if err == nil {
				return nil, fmt.Errorf("decode profile %s: multiple JSON values", entry.Name())
			}
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
	defaultCount := 0
	for _, profile := range result {
		if profile.Default {
			defaultCount++
		}
	}
	if defaultCount > 1 {
		return nil, fmt.Errorf("profile catalog declares %d default profiles; at most one is allowed", defaultCount)
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
	for field, value := range map[string]string{
		"id": profile.ID, "version": profile.Version, "titleKey": profile.TitleKey,
		"descriptionKey": profile.DescriptionKey, "owner": profile.Owner,
		"provenance.source": profile.Provenance.Source, "provenance.reviewRevision": profile.Provenance.Revision,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if !identifierPattern.MatchString(profile.ID) {
		return fmt.Errorf("id %q must contain letters, numbers, underscores, and hyphens", profile.ID)
	}
	if !versionPattern.MatchString(profile.Version) {
		return fmt.Errorf("version %q must be semantic major.minor.patch", profile.Version)
	}
	if profile.CompatibleCatalogMajor != 1 {
		return fmt.Errorf("compatibleCatalogMajor %d is unsupported", profile.CompatibleCatalogMajor)
	}
	if len(profile.Questions) > MaxQuestions {
		return fmt.Errorf("question limit exceeded: %d > %d", len(profile.Questions), MaxQuestions)
	}
	questionIDs := map[string]bool{}
	for _, question := range profile.Questions {
		if !identifierPattern.MatchString(question.ID) || questionIDs[question.ID] {
			return errors.New("question ids must be unique and use letters, numbers, underscores, and hyphens")
		}
		questionIDs[question.ID] = true
	}
	for _, question := range profile.Questions {
		if strings.TrimSpace(question.PromptKey) == "" {
			return fmt.Errorf("question %q promptKey is required", question.ID)
		}
		switch question.Type {
		case "multi-select", "single-select", "text", "boolean":
		default:
			return fmt.Errorf("question %q has unsupported type %q", question.ID, question.Type)
		}
		optionIDs := map[string]bool{}
		if len(question.Options) > MaxOptionsPerQuestion {
			return fmt.Errorf("question %q option limit exceeded: %d > %d", question.ID, len(question.Options), MaxOptionsPerQuestion)
		}
		for _, option := range question.Options {
			if !identifierPattern.MatchString(option.ID) || optionIDs[option.ID] || strings.TrimSpace(option.LabelKey) == "" {
				return fmt.Errorf("question %q has invalid, duplicate, or empty option", question.ID)
			}
			optionIDs[option.ID] = true
		}
		if question.Type == "multi-select" || question.Type == "single-select" {
			if len(question.Options) == 0 {
				return fmt.Errorf("question %q requires options", question.ID)
			}
		} else if len(question.Options) > 0 {
			return fmt.Errorf("question %q cannot define options for type %q", question.ID, question.Type)
		}
		if question.MinSelections < 0 || question.MaxSelections < 0 || (question.MaxSelections > 0 && question.MinSelections > question.MaxSelections) {
			return fmt.Errorf("question %q has invalid selection bounds", question.ID)
		}
		if question.MaxSelections > len(question.Options) {
			return fmt.Errorf("question %q maxSelections exceeds option count", question.ID)
		}
		if question.Type != "multi-select" && (question.MinSelections > 0 || question.MaxSelections > 0) {
			return fmt.Errorf("question %q selection bounds require multi-select", question.ID)
		}
		if err := validateDefault(question, optionIDs); err != nil {
			return fmt.Errorf("question %q default: %w", question.ID, err)
		}
		if len(question.VisibleWhen) > 0 {
			if err := validateConditionReferences(question.VisibleWhen, questionIDs); err != nil {
				return fmt.Errorf("question %q visibility: %w", question.ID, err)
			}
			if err := validateCondition(question.VisibleWhen, 0, new(int)); err != nil {
				return fmt.Errorf("question %q visibility: %w", question.ID, err)
			}
		}
	}
	if err := validateQuestionGraph(profile); err != nil {
		return err
	}
	ruleIDs := map[string]bool{}
	if len(profile.Rules) > MaxRules {
		return fmt.Errorf("rule limit exceeded: %d > %d", len(profile.Rules), MaxRules)
	}
	recommendationCount := 0
	for _, rule := range profile.Rules {
		if !identifierPattern.MatchString(rule.ID) || ruleIDs[rule.ID] {
			return errors.New("rule ids must be unique and use letters, numbers, underscores, and hyphens")
		}
		ruleIDs[rule.ID] = true
		if len(rule.When) > 0 {
			if err := validateConditionReferences(rule.When, questionIDs); err != nil {
				return fmt.Errorf("rule %q: %w", rule.ID, err)
			}
			if err := validateCondition(rule.When, 0, new(int)); err != nil {
				return fmt.Errorf("rule %q: %w", rule.ID, err)
			}
		}
		for _, recommendation := range rule.Recommend {
			recommendationCount++
			if recommendationCount > MaxRecommendations {
				return fmt.Errorf("recommendation limit exceeded: %d > %d", recommendationCount, MaxRecommendations)
			}
			if strings.TrimSpace(recommendation.CapabilityRef) == "" && len(recommendation.ScenarioRefs) == 0 {
				return fmt.Errorf("rule %q contains an empty recommendation", rule.ID)
			}
			if recommendation.CapabilityRef != "" && !referencePattern.MatchString(recommendation.CapabilityRef) {
				return fmt.Errorf("rule %q has invalid capabilityRef %q", rule.ID, recommendation.CapabilityRef)
			}
			if strings.TrimSpace(recommendation.ReasonKey) == "" {
				return fmt.Errorf("rule %q recommendation reasonKey is required", rule.ID)
			}
			for _, scenario := range recommendation.ScenarioRefs {
				if !identifierPattern.MatchString(scenario) {
					return fmt.Errorf("rule %q has invalid scenarioRef %q", rule.ID, scenario)
				}
			}
		}
	}
	return nil
}

func evaluate(profile Profile, answers, targetContext map[string]any, manualDecisions map[string]bool, catalog []Scenario) (Evaluation, error) {
	effectiveAnswers, err := answersWithDefaults(profile, answers)
	if err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{Profile: summary(profile), Valid: true}
	seenRecommendation := map[string]bool{}
	for _, question := range profile.Questions {
		visible := true
		if len(question.VisibleWhen) > 0 {
			visible, err = matches(question.VisibleWhen, effectiveAnswers, targetContext, 0, new(int))
			if err != nil {
				return Evaluation{}, err
			}
		}
		if visible {
			result.Questions = append(result.Questions, QuestionView{Question: question, Visible: true})
			validateAnswer(&result, question, effectiveAnswers[question.ID])
		}
	}
	for _, rule := range profile.Rules {
		matched := true
		if len(rule.When) > 0 {
			matched, err = matches(rule.When, effectiveAnswers, targetContext, 0, new(int))
			if err != nil {
				return Evaluation{}, err
			}
		}
		if matched {
			for _, recommendation := range rule.Recommend {
				recommendation.RuleID = rule.ID
				recommendation.Key = recommendationKey(recommendation)
				recommendation.Selected = true
				if decision, exists := manualDecisions[recommendation.Key]; exists {
					recommendation.Selected = decision || recommendation.Required
					if recommendation.Required && !decision {
						result.Issues = append(result.Issues, Issue{Field: "manualDecisions." + recommendation.Key, Code: "required_recommendation", Message: "a required recommendation cannot be removed"})
					}
				}
				if !seenRecommendation[recommendation.Key] {
					seenRecommendation[recommendation.Key] = true
					result.Recommendations = append(result.Recommendations, recommendation)
				}
				result.Explanations = append(result.Explanations, Explanation{
					RuleID: rule.ID, CapabilityRef: recommendation.CapabilityRef,
					ScenarioRefs: append([]string(nil), recommendation.ScenarioRefs...), ReasonKey: recommendation.ReasonKey, Selected: recommendation.Selected,
				})
			}
		}
	}
	seenScenario := map[string]bool{}
	seenResource := map[string]bool{}
	known := map[string]Scenario{}
	for _, item := range catalog {
		known[item.Name] = item
	}
	for _, recommendation := range result.Recommendations {
		if !recommendation.Selected {
			continue
		}
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
	result.Digest = evaluationDigest(profile, effectiveAnswers, targetContext, result)
	return result, nil
}

func recommendationKey(recommendation Recommendation) string {
	scenarios := append([]string(nil), recommendation.ScenarioRefs...)
	sort.Strings(scenarios)
	return strings.Join([]string{recommendation.CapabilityRef, recommendation.ReasonKey, strings.Join(scenarios, "\x00")}, "\x00")
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

func answersWithDefaults(profile Profile, answers map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(profile.Questions)+len(answers))
	for key, value := range answers {
		result[key] = value
	}
	for _, question := range profile.Questions {
		if _, exists := result[question.ID]; exists || len(question.Default) == 0 {
			continue
		}
		var value any
		if err := json.Unmarshal(question.Default, &value); err != nil {
			return nil, fmt.Errorf("decode default for question %q: %w", question.ID, err)
		}
		result[question.ID] = value
	}
	return result, nil
}

func evaluationDigest(profile Profile, answers, targetContext map[string]any, result Evaluation) string {
	if targetContext == nil {
		targetContext = map[string]any{}
	}
	payload := struct {
		ProfileID       string           `json:"profileId"`
		ProfileVersion  string           `json:"profileVersion"`
		Answers         map[string]any   `json:"answers"`
		TargetContext   map[string]any   `json:"targetContext"`
		Recommendations []Recommendation `json:"recommendations"`
		Scenarios       []string         `json:"scenarios"`
		Resources       []string         `json:"resources"`
		Issues          []Issue          `json:"issues"`
	}{
		ProfileID: profile.ID, ProfileVersion: profile.Version, Answers: answers, TargetContext: targetContext,
		Recommendations: result.Recommendations, Scenarios: result.Scenarios, Resources: result.Resources, Issues: result.Issues,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func validateQuestionGraph(profile Profile) error {
	questions := make(map[string]Question, len(profile.Questions))
	dependencies := make(map[string][]string, len(profile.Questions))
	for _, question := range profile.Questions {
		questions[question.ID] = question
	}
	for _, question := range profile.Questions {
		if len(question.VisibleWhen) == 0 {
			continue
		}
		refs := conditionAnswerReferences(question.VisibleWhen)
		for _, reference := range refs {
			if reference == question.ID {
				return fmt.Errorf("question %q visibility references itself", question.ID)
			}
			dependencies[question.ID] = appendUnique(dependencies[question.ID], reference)
		}
		possible, err := conditionCanMatch(question.VisibleWhen, questions)
		if err != nil {
			return fmt.Errorf("question %q visibility: %w", question.ID, err)
		}
		if !possible {
			return fmt.Errorf("question %q visibility is unreachable", question.ID)
		}
	}
	state := make(map[string]uint8, len(questions))
	var visit func(string) error
	visit = func(questionID string) error {
		switch state[questionID] {
		case 1:
			return fmt.Errorf("question visibility cycle includes %q", questionID)
		case 2:
			return nil
		}
		state[questionID] = 1
		for _, dependency := range dependencies[questionID] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[questionID] = 2
		return nil
	}
	for questionID := range questions {
		if err := visit(questionID); err != nil {
			return err
		}
	}
	return nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func conditionAnswerReferences(raw json.RawMessage) []string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for operator, child := range object {
		switch operator {
		case "all", "any":
			var refs []string
			for _, item := range child.([]any) {
				encoded, _ := json.Marshal(item)
				for _, reference := range conditionAnswerReferences(encoded) {
					refs = appendUnique(refs, reference)
				}
			}
			return refs
		case "not":
			encoded, _ := json.Marshal(child)
			return conditionAnswerReferences(encoded)
		case "contains", "eq", "in":
			if operand, ok := child.(map[string]any); ok {
				if answer, ok := operand["answer"].(string); ok {
					return []string{answer}
				}
			}
		}
	}
	return nil
}

func conditionCanMatch(raw json.RawMessage, questions map[string]Question) (bool, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, err
	}
	object, ok := value.(map[string]any)
	if !ok || len(object) != 1 {
		return false, errors.New("condition must contain exactly one operator")
	}
	for operator, child := range object {
		switch operator {
		case "all", "any":
			items := child.([]any)
			matches := operator == "all"
			for _, item := range items {
				encoded, _ := json.Marshal(item)
				possible, err := conditionCanMatch(encoded, questions)
				if err != nil {
					return false, err
				}
				if operator == "all" {
					matches = matches && possible
				} else {
					matches = matches || possible
				}
			}
			return matches, nil
		case "not":
			// A child being satisfiable does not make its negation unreachable;
			// it can still be false for another valid answer. Keep this branch
			// conservative and let runtime evaluation decide the actual value.
			return true, nil
		case "contains", "eq", "in":
			operand := child.(map[string]any)
			answer, isAnswer := operand["answer"].(string)
			if !isAnswer {
				return true, nil
			}
			question, ok := questions[answer]
			if !ok {
				return false, fmt.Errorf("unknown answer %q", answer)
			}
			if question.Type != "multi-select" && question.Type != "single-select" {
				return true, nil
			}
			options := make(map[string]bool, len(question.Options))
			for _, option := range question.Options {
				options[option.ID] = true
			}
			if operator == "in" {
				for _, item := range operand["values"].([]any) {
					if name, ok := item.(string); ok && options[name] {
						return true, nil
					}
				}
				return false, nil
			}
			name, ok := operand["value"].(string)
			return ok && options[name], nil
		}
	}
	return false, nil
}

func validateDefault(question Question, optionIDs map[string]bool) error {
	if len(question.Default) == 0 || string(question.Default) == "null" {
		if string(question.Default) == "null" {
			return errors.New("must not be null")
		}
		return nil
	}
	var value any
	if err := json.Unmarshal(question.Default, &value); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	switch question.Type {
	case "multi-select":
		values, ok := value.([]any)
		if !ok {
			return errors.New("must be an array")
		}
		if question.MinSelections > 0 && len(values) < question.MinSelections {
			return fmt.Errorf("has fewer than minSelections (%d)", question.MinSelections)
		}
		if question.MaxSelections > 0 && len(values) > question.MaxSelections {
			return fmt.Errorf("has more than maxSelections (%d)", question.MaxSelections)
		}
		seen := map[string]bool{}
		for _, item := range values {
			name, ok := item.(string)
			if !ok || !optionIDs[name] || seen[name] {
				return fmt.Errorf("contains an invalid or duplicate option %v", item)
			}
			seen[name] = true
		}
	case "single-select":
		name, ok := value.(string)
		if !ok || !optionIDs[name] {
			return errors.New("must be a supported option")
		}
	case "text":
		if text, ok := value.(string); !ok || strings.TrimSpace(text) == "" {
			return errors.New("must be a non-empty string")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("must be boolean")
		}
	}
	return nil
}

func validateConditionReferences(raw json.RawMessage, questionIDs map[string]bool) error {
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
			if !ok || len(items) == 0 {
				return fmt.Errorf("%s expects a non-empty array", operator)
			}
			for _, item := range items {
				encoded, _ := json.Marshal(item)
				if err := validateConditionReferences(encoded, questionIDs); err != nil {
					return err
				}
			}
		case "not":
			encoded, _ := json.Marshal(child)
			if err := validateConditionReferences(encoded, questionIDs); err != nil {
				return err
			}
		case "contains", "eq", "in":
			operandObject, ok := child.(map[string]any)
			if !ok {
				return fmt.Errorf("%s expects an object", operator)
			}
			if err := validateOperandReference(operandObject, questionIDs); err != nil {
				return fmt.Errorf("%s: %w", operator, err)
			}
			if operator == "contains" || operator == "eq" {
				if _, exists := operandObject["value"]; !exists {
					return fmt.Errorf("%s requires value", operator)
				}
			} else {
				values, ok := operandObject["values"].([]any)
				if !ok || len(values) == 0 {
					return errors.New("in requires a non-empty values array")
				}
			}
		default:
			return fmt.Errorf("unsupported condition operator %q", operator)
		}
	}
	return nil
}

func validateOperandReference(object map[string]any, questionIDs map[string]bool) error {
	references := 0
	for _, key := range []string{"answer", "context", "setting"} {
		if _, exists := object[key]; exists {
			references++
			name, ok := object[key].(string)
			if !ok || strings.TrimSpace(name) == "" {
				return fmt.Errorf("%s must be a non-empty string", key)
			}
			if key == "answer" && !questionIDs[name] {
				return fmt.Errorf("unknown answer %q", name)
			}
		}
	}
	if references != 1 {
		return errors.New("requires exactly one of answer, context, or setting")
	}
	return nil
}

func summary(profile Profile) ProfileSummary {
	return ProfileSummary{ID: profile.ID, Version: profile.Version, Default: profile.Default, SchemaVersion: profile.SchemaVersion, CompatibleCatalogMajor: profile.CompatibleCatalogMajor, TitleKey: profile.TitleKey, DescriptionKey: profile.DescriptionKey, Owner: profile.Owner, ProvenanceSource: profile.Provenance.Source, ProvenanceRevision: profile.Provenance.Revision, ManualSelectionAvailable: profile.ManualSelection.Available}
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
	current := operandReference(object, answers, context)
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
	return fmt.Sprint(operandReference(object, answers, context)) == fmt.Sprint(object["value"]), nil
}
func compareIn(value any, answers, context map[string]any) (bool, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return false, errors.New("in expects an object")
	}
	current := fmt.Sprint(operandReference(object, answers, context))
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

// operandReference resolves the single reference allowed by the profile
// schema. Context and setting deliberately share the target-context map: the
// two names describe whether the profile is referring to execution context or
// a named non-secret setting, but neither may read from a secret store.
func operandReference(object map[string]any, answers, targetContext map[string]any) any {
	if answer, ok := object["answer"].(string); ok {
		return answers[answer]
	}
	if contextKey, ok := object["context"].(string); ok {
		return targetContext[contextKey]
	}
	if setting, ok := object["setting"].(string); ok {
		return targetContext[setting]
	}
	return nil
}
