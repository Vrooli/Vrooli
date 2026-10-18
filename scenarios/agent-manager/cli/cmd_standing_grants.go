package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Standing Grants Command Dispatcher
//
// Operator surface for the standing-grant model: create, list, and revoke the
// operator-approved, bounded authority that lets a service principal create runs
// unattended. Writes (create/revoke) require a verified independent human; the
// --local-owner exchange authenticates the host operator as that human so the
// operator never hand-runs a raw API call.
// =============================================================================

func (a *App) cmdStandingGrants(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-manager standing-grants create|list|diagnose-prompt-manager|revoke ...")
	}
	switch args[0] {
	case "create":
		return a.standingGrantsCreate(args[1:])
	case "list":
		return a.standingGrantsList(args[1:])
	case "diagnose-prompt-manager":
		return a.standingGrantsDiagnosePromptManager(args[1:])
	case "revoke":
		return a.standingGrantsRevoke(args[1:])
	case "help", "-h", "--help":
		fmt.Println("agent-manager standing-grants create --subject <name> --max-concurrent-runs N --ttl 30d [--local-owner] [--json]")
		fmt.Println("agent-manager standing-grants list [--local-owner] [--json]")
		fmt.Println("agent-manager standing-grants diagnose-prompt-manager [--json] (requires agent.investigate authority)")
		fmt.Println("agent-manager standing-grants revoke <id> [--local-owner] [--json]")
		return nil
	default:
		return fmt.Errorf("unknown standing-grants subcommand: %s", args[0])
	}
}

func (a *App) standingGrantsDiagnosePromptManager(args []string) error {
	fs := flag.NewFlagSet("standing-grants diagnose-prompt-manager", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Maintenance.api.Request("GET", "/api/v1/standing-grants/diagnose/prompt-manager-run-create", nil, nil)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var result struct {
		Outcome    string `json:"outcome"`
		Reason     string `json:"reason"`
		ObservedAt string `json:"observed_at"`
		Limits     []struct {
			Dimension string  `json:"dimension"`
			Current   float64 `json:"current"`
			Bound     float64 `json:"bound"`
		} `json:"limits"`
		LimitsTruncated bool `json:"limits_truncated"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Prompt Manager run.create: %s (%s; observed %s)\n", result.Outcome, result.Reason, result.ObservedAt)
	for _, limit := range result.Limits {
		fmt.Printf("  %s: %.2f / %.2f\n", limit.Dimension, limit.Current, limit.Bound)
	}
	if result.LimitsTruncated {
		fmt.Println("  Additional exceeded bounds are omitted.")
	}
	return nil
}

// standingGrantResource is the audience the operator's local-owner token must
// carry for agent-manager to verify it. It matches the resource-audience entry
// added to scenario-authenticator (agent-manager=scenario-authenticator:agent-manager).
const standingGrantResource = "agent-manager"

// localOwnerAPIForResource exchanges explicit local human authority for an owner
// token scoped to a specific resource audience, so a verified-independent-human
// endpoint on that resource accepts it. Refused inside an identified agent run so
// an agent cannot elevate — the same guard as localOwnerAPI.
func localOwnerAPIForResource(api *cliutil.APIClient, resource string) (*cliutil.APIClient, error) {
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
		return nil, fmt.Errorf("--local-owner is unavailable inside an identified agent run; use the granted credential")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	credential, err := cliutil.ResolveAgentCredential(ctx)
	if err != nil {
		return nil, fmt.Errorf("local owner caller authority unavailable: %w", err)
	}
	if credential != "" {
		return nil, fmt.Errorf("--local-owner is unavailable inside an identified agent run; use the granted credential")
	}
	login, err := authn.ExchangeLocalMachinePrincipalForResource(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("local owner exchange unavailable: %w", err)
	}
	return api.WithToken(login.Tokens.AccessToken), nil
}

// parseGrantTTL accepts a Go duration (e.g. "720h") or a day suffix ("30d").
func parseGrantTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("--ttl is required (e.g. 30d)")
	}
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid --ttl %q: use a positive day count like 30d", raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid --ttl %q: use e.g. 30d or 720h", raw)
	}
	return d, nil
}

func (a *App) standingGrantsCreate(args []string) error {
	fs := flag.NewFlagSet("standing-grants create", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; unavailable inside an identified agent run")
	subject := fs.String("subject", "", "Verified service subject the grant is keyed to (e.g. prompt-manager)")
	maxConcurrent := fs.Int("max-concurrent-runs", 0, "Ceiling on concurrent runs attributed to the operator (0 = unbounded)")
	maxBudget := fs.Float64("max-run-budget-usd", 0, "Advisory budget ceiling in USD (not yet enforced)")
	ttl := fs.String("ttl", "", "Grant lifetime, e.g. 30d or 720h (max 30d)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*subject) == "" {
		return fmt.Errorf("--subject is required")
	}
	ttlSeconds, err := parseGrantTTL(*ttl)
	if err != nil {
		return err
	}
	api := a.services.Maintenance.api
	if *localOwner {
		owner, err := localOwnerAPIForResource(api, standingGrantResource)
		if err != nil {
			return err
		}
		api = owner
	}
	payload, _ := json.Marshal(map[string]any{
		"subject":             strings.TrimSpace(*subject),
		"max_concurrent_runs": *maxConcurrent,
		"max_run_budget_usd":  *maxBudget,
		"ttl_seconds":         int64(ttlSeconds / time.Second),
	})
	body, err := api.Request("POST", "/api/v1/standing-grants", nil, payload)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Created standing grant for %s (ttl %s, max_concurrent_runs %d).\n", strings.TrimSpace(*subject), *ttl, *maxConcurrent)
	return nil
}

func (a *App) standingGrantsList(args []string) error {
	fs := flag.NewFlagSet("standing-grants list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; unavailable inside an identified agent run")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	api := a.services.Maintenance.api
	if *localOwner {
		owner, err := localOwnerAPIForResource(api, standingGrantResource)
		if err != nil {
			return err
		}
		api = owner
	}
	body, err := api.Request("GET", "/api/v1/standing-grants", nil, nil)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp struct {
		HasMore bool `json:"has_more"`
		Grants  []struct {
			ID                string `json:"id"`
			Subject           string `json:"subject"`
			State             string `json:"state"`
			MaxConcurrentRuns int    `json:"max_concurrent_runs"`
			ExpiresAt         string `json:"expires_at"`
		} `json:"grants"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		// Unexpected shape — fall back to raw rather than hide the response.
		cliutil.PrintJSON(body)
		return nil
	}
	if len(resp.Grants) == 0 {
		fmt.Println("No standing grants.")
		return nil
	}
	for _, g := range resp.Grants {
		fmt.Printf("%s  subject=%s  state=%s  max_concurrent_runs=%d  expires=%s\n",
			g.ID, g.Subject, g.State, g.MaxConcurrentRuns, g.ExpiresAt)
	}
	if resp.HasMore {
		fmt.Println("More standing grants exist; only the first 64 are shown.")
	}
	return nil
}

func (a *App) standingGrantsRevoke(args []string) error {
	fs := flag.NewFlagSet("standing-grants revoke", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	localOwner := fs.Bool("local-owner", false, "Explicit local operator authentication; unavailable inside an identified agent run")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 || strings.TrimSpace(rest[0]) == "" {
		return fmt.Errorf("usage: agent-manager standing-grants revoke <id> [--local-owner]")
	}
	api := a.services.Maintenance.api
	if *localOwner {
		owner, err := localOwnerAPIForResource(api, standingGrantResource)
		if err != nil {
			return err
		}
		api = owner
	}
	body, err := api.Request("POST", "/api/v1/standing-grants/"+strings.TrimSpace(rest[0])+"/revoke", nil, nil)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println("Revoked.")
	return nil
}
