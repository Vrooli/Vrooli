// Package glossary owns the static onboarding glossary.
package glossary

import (
	"context"
	"sort"
	"strings"

	glossaryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary"
)

var entries = []*glossaryv1.GlossaryEntry{
	{Term: "resource", Description: "A local service that your applications can use, like a database or AI model", Category: "core"},
	{Term: "scenario", Description: "A complete application or microservice built from resources and other scenarios", Category: "core"},
	{Term: "service.json", Description: "The main configuration file that defines how your application is set up", Category: "config"},
	{Term: "postgres", Description: "A powerful relational database for storing structured data", Category: "database"},
	{Term: "redis", Description: "A fast in-memory data store for caching and messaging", Category: "database"},
	{Term: "ollama", Description: "A local AI model runner that lets you use language models without cloud services", Category: "ai"},
	{Term: "qdrant", Description: "A vector database for storing and searching AI embeddings", Category: "database"},
	{Term: "vault", Description: "A secrets management tool that securely stores passwords and API keys", Category: "security"},
	{Term: "health check", Description: "An automatic test that verifies a service is running correctly", Category: "operations"},
	{Term: "port", Description: "A numbered channel that services use to communicate (like a phone extension)", Category: "networking"},
	{Term: "dependency", Description: "A service that another service needs to work properly", Category: "core"},
	{Term: "lifecycle", Description: "The stages a service goes through: setup, start, run, and stop", Category: "operations"},
	{Term: "API", Description: "Application Programming Interface - how software components talk to each other", Category: "core"},
	{Term: "endpoint", Description: "A specific URL where a service accepts requests", Category: "networking"},
}

// configurationDescriptors is the safe, display-oriented configuration index.
// It intentionally contains routes and prerequisites, never current choices,
// credential values, private endpoints, or other target-owned state.
var configurationDescriptors = []*glossaryv1.ConfigurationDescriptor{
	{Id: "setup.scenarios", Title: "Scenarios", Purpose: "Choose the applications and capabilities to make available on the target.", Route: "/setup/scenarios", StepId: "scenarios", TargetKinds: []string{"local", "remote"}, Tags: []string{"applications", "capabilities", "selection"}},
	{Id: "setup.core-set", Title: "Core set", Purpose: "Choose the trusted base services that support the selected capabilities.", Route: "/setup/core-set", StepId: "core-set", TargetKinds: []string{"local", "remote"}, Tags: []string{"core", "services", "selection"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.resources", Title: "Resources", Purpose: "Review and enable the resources derived from the selected capabilities.", Route: "/setup/resources", StepId: "resources", TargetKinds: []string{"local", "remote"}, Tags: []string{"resources", "dependencies"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.credentials", Title: "Credentials", Purpose: "Provide operator-supplied credentials declared by the selected capabilities.", Route: "/setup/credentials", StepId: "credentials", TargetKinds: []string{"local", "remote"}, Tags: []string{"credentials", "access"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.host", Title: "Host safeguards", Purpose: "Review host tools and safeguards required by the selected capabilities.", Route: "/setup/host", StepId: "host", TargetKinds: []string{"local", "remote"}, Tags: []string{"host", "permissions", "safeguards"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.operating-mode", Title: "Operating mode", Purpose: "Choose restart and operating policies for selected capabilities.", Route: "/setup/operating-mode", StepId: "operating-mode", TargetKinds: []string{"local", "remote"}, Tags: []string{"policy", "restart", "operations"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.apply", Title: "Review and apply", Purpose: "Review the proposed changes and explicitly consent before applying them.", Route: "/setup/apply", StepId: "apply", TargetKinds: []string{"local", "remote"}, Tags: []string{"review", "consent", "apply"}, Prerequisites: []string{"setup.scenarios"}},
	{Id: "setup.validation", Title: "Readiness", Purpose: "Verify the target state and inspect recovery actions after an apply.", Route: "/setup/validation", StepId: "validation", TargetKinds: []string{"local", "remote"}, Tags: []string{"health", "readiness", "verify"}, Prerequisites: []string{"setup.apply"}},
}

func Search(_ context.Context, query string) (*glossaryv1.SearchGlossaryResponse, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	result := &glossaryv1.SearchGlossaryResponse{Entries: []*glossaryv1.GlossaryEntry{}, Query: q}
	for _, entry := range entries {
		if q == "" || strings.Contains(strings.ToLower(entry.GetTerm()), q) || strings.Contains(strings.ToLower(entry.GetDescription()), q) {
			copy := *entry
			result.Entries = append(result.Entries, &copy)
		}
	}
	result.Count = int32(len(result.Entries))
	return result, nil
}

func SearchConfiguration(_ context.Context, query, target string) (*glossaryv1.SearchConfigurationResponse, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	result := &glossaryv1.SearchConfigurationResponse{Results: []*glossaryv1.ConfigurationDescriptor{}, Query: q, Target: strings.TrimSpace(target), Fallback: true}
	for _, descriptor := range configurationDescriptors {
		if q != "" && configurationScore(descriptor, q) == 0 {
			continue
		}
		copy := *descriptor
		copy.TargetKinds = append([]string(nil), descriptor.TargetKinds...)
		copy.Tags = append([]string(nil), descriptor.Tags...)
		copy.Prerequisites = append([]string(nil), descriptor.Prerequisites...)
		copy.Score = configurationScore(descriptor, q)
		result.Results = append(result.Results, &copy)
	}
	sort.SliceStable(result.Results, func(i, j int) bool { return result.Results[i].GetScore() > result.Results[j].GetScore() })
	result.Count = int32(len(result.Results))
	return result, nil
}

func configurationScore(descriptor *glossaryv1.ConfigurationDescriptor, query string) float32 {
	if query == "" {
		return 0.5
	}
	if strings.EqualFold(descriptor.GetId(), query) || strings.EqualFold(descriptor.GetStepId(), query) {
		return 1
	}
	tokens := configurationQueryTokens(query)
	if len(tokens) == 0 {
		return 0
	}
	fields := []struct {
		values []string
		weight float32
	}{
		{[]string{descriptor.GetId(), descriptor.GetStepId(), descriptor.GetTitle()}, 3},
		{descriptor.GetTags(), 2},
		{[]string{descriptor.GetPurpose()}, 1},
		{descriptor.GetPrerequisites(), 1},
	}
	matchedWeight := float32(0)
	for _, token := range tokens {
		for _, field := range fields {
			if containsConfigurationToken(field.values, token) {
				matchedWeight += field.weight
				break
			}
		}
	}
	if matchedWeight == 0 {
		return 0
	}
	// Keep exact title/id/tag matches above broad purpose matches while
	// retaining a stable, bounded score for Search Hub's result mapper.
	score := 0.5 + matchedWeight/(float32(len(tokens))*3.0)*0.49
	if score > 0.99 {
		return 0.99
	}
	return score
}

func configurationQueryTokens(query string) []string {
	stopWords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "before": {}, "can": {}, "configure": {},
		"do": {}, "for": {}, "find": {}, "how": {}, "i": {}, "is": {}, "me": {}, "my": {},
		"of": {}, "on": {}, "please": {}, "setting": {}, "settings": {}, "show": {}, "the": {},
		"to": {}, "turn": {}, "what": {}, "where": {}, "which": {}, "with": {},
	}
	seen := map[string]struct{}{}
	tokens := make([]string, 0)
	for _, token := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	}) {
		if len(token) < 2 {
			continue
		}
		if _, stop := stopWords[token]; stop {
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func containsConfigurationToken(values []string, token string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), token) {
			return true
		}
	}
	return false
}
