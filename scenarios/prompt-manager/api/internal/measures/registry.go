// Package measures owns prompt-manager's uniform analytical measure registry.
// The manifest remains the discovery SSOT; this package supplies the live,
// data-bearing compute functions required by the fleet measures provider.
package measures

import (
	"context"
	"fmt"
	"net/http"

	measurelib "github.com/vrooli/measures-go"
)

// Provider computes one declared measure from the owning domain's live store.
// Implementations must describe the concrete read in ExecutedQuery.
type Provider func(context.Context, measurelib.MeasureRequest) (measurelib.MeasureResult, error)

// Handler returns the shared measures-go HTTP substrate for every declared
// prompt-manager measure. Missing providers fail startup rather than exposing a
// declaration that can never return evidence.
func Handler(providers map[string]Provider) (http.Handler, error) {
	registry := measurelib.NewRegistry()
	for _, declaration := range declarations() {
		provider, ok := providers[declaration.Name]
		if !ok {
			return nil, fmt.Errorf("measure %q has no compute provider", declaration.Name)
		}
		if err := registry.Register(declaration, measurelib.ComputeFunc(provider)); err != nil {
			return nil, fmt.Errorf("register measure %q: %w", declaration.Name, err)
		}
	}
	return registry.Handler(), nil
}

func declarations() []measurelib.MeasureDeclaration {
	return []measurelib.MeasureDeclaration{
		table("actions.list", "actions", "How many governed actions are available.", []string{"how many actions are available", "count governed actions"}, "actions", "ActionsService", "ListActions"),
		table("agents.list", "agents", "How many persisted agents are available.", []string{"how many agents are available", "count persisted agents"}, "agents", "AgentsService", "ListAgents"),
		scalar("aisearch.discovery-metrics", "aisearch", "How many discovery calls are retained in telemetry.", []string{"how many discovery calls are recorded", "count discovery calls"}, "discovery calls", "DiscoveryService", "GetDiscoveryMetrics"),
		table("graph.health", "graph", "How many graph nodes carry computed health evidence.", []string{"how many graph nodes have health scores", "count graph health scores"}, "graph health scores", "GraphService", "GetHealthScores"),
		table("metrics.skill-usage", "metrics", "How many persisted skill-usage rows are available.", []string{"how many skills have usage metrics", "count skill usage rows"}, "skill usage rows", "DiscoveryService", "GetSkillUsage"),
		table("skills.list", "skills", "How many governed skills are available.", []string{"how many skills are available", "count governed skills"}, "skills", "SkillsService", "ListSkills"),
		table("tags.list", "tags", "How many persisted tags are available.", []string{"how many tags are available", "count persisted tags"}, "tags", "TagsService", "ListTags"),
		table("teams.list", "teams", "How many persisted teams are available.", []string{"how many teams are available", "count persisted teams"}, "teams", "TeamsService", "ListTeams"),
		table("topics.list", "topics", "How many persisted topics are available.", []string{"how many topics are available", "count persisted topics"}, "topics", "TopicsService", "ListTopics"),
	}
}

func table(name, domain, intent string, questions []string, unit, service, method string) measurelib.MeasureDeclaration {
	return measurelib.MeasureDeclaration{
		Name: name, Domain: domain, Intent: intent, Questions: questions,
		Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: unitField(name), Unit: unit, SummaryTemplate: "{count} " + unit},
		Effect: measurelib.EffectRead, RunEligible: true, Service: service, Method: method,
	}
}

func scalar(name, domain, intent string, questions []string, unit, service, method string) measurelib.MeasureDeclaration {
	return measurelib.MeasureDeclaration{
		Name: name, Domain: domain, Intent: intent, Questions: questions,
		Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "call_count", Unit: unit, SummaryTemplate: "{count} " + unit},
		Effect: measurelib.EffectRead, RunEligible: true, Service: service, Method: method,
	}
}

func unitField(name string) string {
	switch name {
	case "graph.health":
		return "scores"
	case "metrics.skill-usage":
		return "rows"
	default:
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '.' {
				return name[:i]
			}
		}
		return name
	}
}
