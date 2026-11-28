package pipeline

// Distribution filtering is deliberately a consumer of the same manifest
// vocabulary as conformance. It does not scrape prose or maintain a second
// command registry: skill frontmatter declares requirements and the target
// cli-manifest is the authority for what a bundle recipient has.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type skillDistributionResult struct {
	Skill           string   `json:"skill"`
	Source          string   `json:"source"`
	Target          string   `json:"target"`
	Distributable   bool     `json:"distributable"`
	MissingCommands []string `json:"missingCommands,omitempty"`
}

type distributabilityReport struct {
	Scenario string                    `json:"scenario"`
	Target   string                    `json:"target"`
	Skills   []skillDistributionResult `json:"skills"`
}

type skillRequirementFrontmatter struct {
	Metadata struct {
		Requires struct {
			Commands []string `yaml:"commands"`
		} `yaml:"requires"`
	} `yaml:"metadata"`
}

func readSkillRequirements(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(b)
	if !strings.HasPrefix(text, "---") {
		return nil, fmt.Errorf("%s: missing frontmatter", path)
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("%s: incomplete frontmatter", path)
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("%s: unterminated frontmatter", path)
	}
	var fm skillRequirementFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return nil, fmt.Errorf("%s: parse frontmatter: %w", path, err)
	}
	return fm.Metadata.Requires.Commands, nil
}

func loadTargetCommands(path string) (string, map[string]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	var manifest cliManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return "", nil, fmt.Errorf("parse %s: %w", path, err)
	}
	commands := make(map[string]bool)
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			commands[group.Name+" "+command.Name] = true
			commands[command.Name] = true
		}
	}
	return manifestName(path, manifest), commands, nil
}

// manifestName keeps the target executable optional. The current cli-manifest
// schema has no required name in cliManifest, so use the filename's parent
// convention only as a convenience and match both qualified and unqualified
// requirements below.
func manifestName(path string, manifest cliManifest) string {
	if strings.TrimSpace(manifest.Name) != "" {
		return strings.TrimSpace(manifest.Name)
	}
	base := filepath.Base(filepath.Dir(path))
	if base == "cli" {
		base = filepath.Base(filepath.Dir(filepath.Dir(path)))
	}
	if base != "" && base != "." {
		return base
	}
	return filepath.Base(path)
}

func requirementCommand(requirement, targetName string) string {
	fields := strings.Fields(requirement)
	if len(fields) == 0 {
		return ""
	}
	if targetName != "" && fields[0] == targetName {
		fields = fields[1:]
	}
	if len(fields) >= 2 {
		return fields[len(fields)-2] + " " + fields[len(fields)-1]
	}
	return fields[0]
}

func evaluateSkillDistribution(source, target string) (skillDistributionResult, error) {
	result := skillDistributionResult{Source: source, Target: target, Skill: filepath.Base(filepath.Dir(source)), Distributable: true}
	requires, err := readSkillRequirements(source)
	if err != nil {
		return result, err
	}
	targetName, commands, err := loadTargetCommands(target)
	if err != nil {
		return result, err
	}
	for _, requirement := range requires {
		command := requirementCommand(requirement, targetName)
		if command == "" || commands[command] {
			continue
		}
		result.Distributable = false
		result.MissingCommands = append(result.MissingCommands, requirement)
	}
	return result, nil
}

func distributabilityForScenario(root, scenario, target string) (distributabilityReport, error) {
	manifestPath := target
	if !filepath.IsAbs(manifestPath) {
		manifestPath = filepath.Join(root, manifestPath)
	}
	m, scenarioRoot, err := (&handler{root: root}).read(scenario)
	if err != nil {
		return distributabilityReport{}, err
	}
	if m.Plugin == nil {
		return distributabilityReport{}, fmt.Errorf("scenario %q has no plugin declaration", scenario)
	}
	report := distributabilityReport{Scenario: scenario, Target: target}
	for _, skill := range m.Plugin.Skills {
		source := filepath.Join(scenarioRoot, skill.Source)
		result, err := evaluateSkillDistribution(source, manifestPath)
		if err != nil {
			return distributabilityReport{}, err
		}
		report.Skills = append(report.Skills, result)
	}
	return report, nil
}
