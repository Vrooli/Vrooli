// Package main provides resource resolution for deployment manifest generation.
//
// This file contains the ResourceResolver interface and its implementation,
// which determines the effective resource list for a scenario by combining:
//   - Resources declared in the scenario's service.json
//   - Resources identified by the scenario-dependency-analyzer
//   - Resources explicitly requested by the caller
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResourceResolver abstracts resource discovery from service.json and analyzer.
type ResourceResolver interface {
	// ResolveResources determines the effective resource list for a scenario.
	// It merges resources from service.json, analyzer, and explicit requests.
	ResolveResources(ctx context.Context, scenario string, requestedResources []string) ResolvedResources
}

// ResolvedResources contains the resolved resource lists from different sources.
type ResolvedResources struct {
	// Effective is the final merged list used for secret lookups.
	Effective []string

	// FromServiceJSON contains resources declared in service.json.
	FromServiceJSON []string

	// FromAnalyzer contains resources identified by the analyzer.
	FromAnalyzer []string

	// FromRequest contains resources explicitly requested by the caller.
	FromRequest []string
}

// DefaultResourceResolver implements ResourceResolver.
type DefaultResourceResolver struct {
	analyzerClient AnalyzerClient
	logger         *Logger
}

// NewResourceResolver creates a production ResourceResolver.
func NewResourceResolver(analyzer AnalyzerClient, logger *Logger) *DefaultResourceResolver {
	return &DefaultResourceResolver{
		analyzerClient: analyzer,
		logger:         logger,
	}
}

// ResolveResources determines the effective resource list for a scenario.
//
// Resolution priority:
//  1. If requestedResources are provided and intersect with base resources,
//     use the intersection (allows filtering to specific resources).
//  2. Otherwise, use the base resources (service.json → analyzer → request fallback).
//  3. If no resources are found, synthesize a placeholder resource.
func (r *DefaultResourceResolver) ResolveResources(ctx context.Context, scenario string, requestedResources []string) ResolvedResources {
	result := ResolvedResources{
		FromRequest: dedupeStrings(requestedResources),
	}

	// Fetch resources from service.json
	result.FromServiceJSON = r.resolveScenarioResources(scenario)

	// Fetch resources from analyzer
	if r.analyzerClient != nil {
		report, _ := r.analyzerClient.FetchDeploymentReport(ctx, scenario)
		result.FromAnalyzer = dedupeStrings(analyzerResourceNames(report))
	}

	// Determine base resources with fallback chain
	baseResources := result.FromServiceJSON
	if len(baseResources) == 0 && len(result.FromAnalyzer) > 0 {
		baseResources = result.FromAnalyzer
	}
	if len(baseResources) == 0 && len(result.FromRequest) > 0 {
		baseResources = result.FromRequest
	}
	if len(baseResources) == 0 {
		// Synthesize a placeholder so bundle consumers receive a prompt plan
		baseResources = []string{fmt.Sprintf("%s-core", scenario)}
	}

	// Apply intersection filter if requested resources were provided
	result.Effective = baseResources
	if len(result.FromRequest) > 0 {
		if intersected := intersectStrings(baseResources, result.FromRequest); len(intersected) > 0 {
			result.Effective = intersected
		}
	}

	return result
}

// resolveScenarioResources reads the scenario's service.json to find declared resources.
// It searches multiple potential locations for the service.json file.
func (r *DefaultResourceResolver) resolveScenarioResources(scenario string) []string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}

	data := r.findServiceJSON(scenario)
	if len(data) == 0 {
		return nil
	}

	return r.parseResourcesFromServiceJSON(data)
}

// findServiceJSON searches for the scenario's service.json in candidate locations.
func (r *DefaultResourceResolver) findServiceJSON(scenario string) []byte {
	readServiceJSON := func(root string) []byte {
		paths := []string{
			filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json"),
			filepath.Join(root, scenario, ".vrooli", "service.json"),
		}
		for _, servicePath := range paths {
			if payload, err := os.ReadFile(servicePath); err == nil {
				return payload
			}
		}
		return nil
	}

	// Build candidate root directories
	var candidates []string
	if envRoot := os.Getenv("VROOLI_ROOT"); envRoot != "" {
		candidates = append(candidates, envRoot)
	}
	if cwd, err := os.Getwd(); err == nil {
		absCwd, _ := filepath.Abs(cwd)
		candidates = append(candidates, absCwd)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "Vrooli"))
	}

	// Search each candidate, climbing up the directory tree
	for _, root := range candidates {
		if root == "" {
			continue
		}

		cur := root
		for {
			if payload := readServiceJSON(cur); payload != nil {
				return payload
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}

	return nil
}

// parseResourcesFromServiceJSON extracts resource names from service.json content.
func (r *DefaultResourceResolver) parseResourcesFromServiceJSON(data []byte) []string {
	var doc struct {
		Service struct {
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		} `json:"service"`
	}

	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}

	if len(doc.Service.Dependencies.Resources) == 0 {
		return nil
	}

	resources := make([]string, 0, len(doc.Service.Dependencies.Resources))
	for name := range doc.Service.Dependencies.Resources {
		name = strings.TrimSpace(name)
		if name != "" {
			resources = append(resources, name)
		}
	}
	return dedupeStrings(resources)
}

// analyzerResourceNames extracts resource names from an analyzer report.
func analyzerResourceNames(report *analyzerDeploymentReport) []string {
	if report == nil {
		return nil
	}

	var names []string
	for _, dep := range report.Dependencies {
		if dep.Type == "resource" && dep.Name != "" {
			names = append(names, dep.Name)
		}
	}
	return names
}
