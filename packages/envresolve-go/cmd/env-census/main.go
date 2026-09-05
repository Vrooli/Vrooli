package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	envresolve "github.com/vrooli/envresolve-go"
)

type result struct {
	Pairs       int            `json:"pairs"`
	Scenarios   int            `json:"scenarios"`
	Classes     map[string]int `json:"classes"`
	Variables   map[string]int `json:"variables_by_class"`
	Occurrences []Occurrence   `json:"occurrences"`
}

type Occurrence struct {
	Scenario string `json:"scenario"`
	File     string `json:"file"`
	Variable string `json:"variable"`
	Class    string `json:"class"`
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	idx, err := envresolve.Load(*root)
	if err != nil {
		fatal(err)
	}
	osAllow := envresolve.OSStandardVariables()
	out := result{Classes: map[string]int{"A": 0, "B": 0, "C": 0}, Variables: map[string]int{}}
	seenScenarios := map[string]struct{}{}
	// Walk each scenario tree because filepath.Glob does not recurse on **.
	for _, scenarioDir := range scenarioDirs(*root) {
		manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		manifestBytes, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}
		var manifest envresolve.Manifest
		if json.Unmarshal(manifestBytes, &manifest) != nil {
			continue
		}
		scenario := filepath.Base(scenarioDir)
		manifest.Name = scenario
		_ = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil || info.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || strings.Contains(path, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			reads, parseErr := envresolve.FindEnvReads(payload)
			if parseErr != nil {
				return nil
			}
			for _, read := range reads {
				variable := read.Variable
				if _, ok := osAllow[variable]; ok {
					continue
				}
				producers := idx.Producers(variable)
				class := "C"
				satisfiable, _ := idx.Satisfiable(manifest, variable)
				if !satisfiable {
					for _, producer := range producers {
						if producer.Kind == envresolve.ResourceProducer {
							class = "A"
							break
						}
						if (producer.Kind == envresolve.ScenarioPortProducer || producer.Kind == envresolve.ScenarioAbsoluteSource) && envresolve.IsScenarioAddressVariable(variable) {
							class = "B"
						}
					}
				}
				out.Pairs++
				out.Classes[class]++
				out.Variables[variable]++
				relative, _ := filepath.Rel(*root, path)
				out.Occurrences = append(out.Occurrences, Occurrence{Scenario: scenario, File: filepath.ToSlash(relative), Variable: variable, Class: class})
				seenScenarios[scenario] = struct{}{}
			}
			return nil
		})
	}
	out.Scenarios = len(seenScenarios)
	sort.Slice(out.Occurrences, func(i, j int) bool {
		return fmt.Sprint(out.Occurrences[i].Scenario, out.Occurrences[i].File, out.Occurrences[i].Variable) < fmt.Sprint(out.Occurrences[j].Scenario, out.Occurrences[j].File, out.Occurrences[j].Variable)
	})
	json.NewEncoder(os.Stdout).Encode(out)
}

func scenarioDirs(root string) []string {
	entries, _ := os.ReadDir(filepath.Join(root, "scenarios"))
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, filepath.Join(root, "scenarios", entry.Name()))
		}
	}
	return result
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
