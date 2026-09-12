package scenariocli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type DemandRequest struct {
	Operation string
	Scenario  string
	Variant   string
	Consumer  string
	Kind      string
	RequestID string
	LeaseID   string
	TTL       string
	Reason    string
	Metadata  string
	JSON      bool
	Limit     int
}

func ParseDemandRequest(globalsJSON bool, args []string) (DemandRequest, error) {
	spec := commandSpec(CommandDemand)
	parsed, err := commandtree.ParseArgs("scenario demand", commandHelpText(CommandDemand), spec.Args, args)
	if err != nil {
		return DemandRequest{}, err
	}
	req := DemandRequest{
		Operation: parsed.Positionals[0], Scenario: parsed.FlagValue("--scenario"), Variant: parsed.FlagValue("--variant"),
		Consumer: parsed.FlagValue("--consumer"), Kind: parsed.FlagValue("--kind"), RequestID: parsed.FlagValue("--request-id"),
		LeaseID: parsed.FlagValue("--lease-id"), TTL: parsed.FlagValue("--ttl"), Reason: parsed.FlagValue("--reason"),
		Metadata: parsed.FlagValue("--metadata"), JSON: globalsJSON || parsed.HasFlag("--json"),
	}
	if req.Operation != "acquire" && req.Operation != "renew" && req.Operation != "release" && req.Operation != "history" {
		return DemandRequest{}, fmt.Errorf("scenario demand: operation must be acquire, renew, release, or history")
	}
	req.Limit = 100
	if raw := parsed.FlagValue("--limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > scenarioruntime.DemandAuditRetention || req.Operation != "history" {
			return DemandRequest{}, fmt.Errorf("scenario demand: --limit requires history and a value from 1 to %d", scenarioruntime.DemandAuditRetention)
		}
		req.Limit = limit
	}
	if req.Operation == "history" && strings.TrimSpace(req.Scenario) == "" {
		return DemandRequest{}, fmt.Errorf("scenario demand history requires --scenario")
	}
	if req.TTL != "" {
		if _, err := time.ParseDuration(req.TTL); err != nil {
			return DemandRequest{}, fmt.Errorf("scenario demand: invalid --ttl %q: %w", req.TTL, err)
		}
	}
	return req, nil
}

func demandOptions() []commandtree.OptionArg {
	return []commandtree.OptionArg{
		{Name: "--scenario", ValueName: "name"}, {Name: "--variant", ValueName: "variant"},
		{Name: "--consumer", ValueName: "id"}, {Name: "--kind", ValueName: "kind"},
		{Name: "--request-id", ValueName: "id"}, {Name: "--lease-id", ValueName: "id"},
		{Name: "--ttl", ValueName: "duration"}, {Name: "--reason", ValueName: "reason"},
		{Name: "--metadata", ValueName: "json"}, commandtree.JSONOption(),
		{Name: "--limit", ValueName: "count", Description: "Maximum recent history rows (1–1000; default 100)"},
	}
}
