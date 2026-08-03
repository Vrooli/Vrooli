package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdSubscription(args []string) error {
	if len(args) < 2 || args[0] != "periods" {
		return a.subscriptionHelp()
	}
	switch args[1] {
	case "create":
		return a.subscriptionCreate(args[2:])
	case "list":
		return a.subscriptionList(args[2:])
	case "remove":
		return a.subscriptionRemove(args[2:])
	default:
		return fmt.Errorf("unknown subscription periods subcommand: %s", args[1])
	}
}

func (a *App) subscriptionHelp() error {
	fmt.Println("Usage: agent-manager subscription periods <create|list|remove>")
	fmt.Println("  create --provider P --plan-ref PLAN --from RFC3339 --to RFC3339 --amount-micro-usd N [--quota-tokens N]")
	fmt.Println("  list [--provider P] [--plan-ref PLAN] [--json]")
	fmt.Println("  remove <id>")
	return nil
}

func (a *App) subscriptionCreate(args []string) error {
	fs := flag.NewFlagSet("subscription periods create", flag.ContinueOnError)
	provider := fs.String("provider", "", "provider")
	plan := fs.String("plan-ref", "", "plan reference")
	from := fs.String("from", "", "period start RFC3339")
	to := fs.String("to", "", "period end RFC3339")
	amount := fs.Int64("amount-micro-usd", 0, "period amount in micro-USD")
	quota := fs.Int64("quota-tokens", 0, "optional token quota")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *provider == "" || *plan == "" || *from == "" || *to == "" {
		return fmt.Errorf("--provider, --plan-ref, --from, and --to are required")
	}
	start, err := time.Parse(time.RFC3339, *from)
	if err != nil {
		return err
	}
	end, err := time.Parse(time.RFC3339, *to)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"provider": *provider, "plan_ref": *plan, "starts_at": start, "ends_at": end, "amount_micro_usd": *amount, "quota_tokens": *quota})
	body, err := a.services.Subscriptions.Create(payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
	} else {
		fmt.Println("subscription period created")
	}
	return nil
}

func (a *App) subscriptionList(args []string) error {
	fs := flag.NewFlagSet("subscription periods list", flag.ContinueOnError)
	provider := fs.String("provider", "", "provider")
	plan := fs.String("plan-ref", "", "plan reference")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Subscriptions.List(*provider, *plan)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}

func (a *App) subscriptionRemove(args []string) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("period id is required")
	}
	_, err := a.services.Subscriptions.Remove(args[0])
	if err == nil {
		fmt.Println("subscription period removed")
	}
	return err
}
