package devfixtures

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"landing-page-business-suite/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Local subscription fixtures",
		Commands: []cliapp.Command{
			seedCommand(deps), tokenCommand(deps), balanceCommand(deps), zeroCommand(deps),
		},
	}
}

func seedCommand(deps support.Dependencies) cliapp.Command {
	return cliapp.Command{Name: "fixture-seed", NeedsAPI: true, Description: "Seed an idempotent local subscription fixture (development only)", Run: func(args []string) error {
		fs := flag.NewFlagSet("fixture-seed", flag.ContinueOnError)
		email := fs.String("email", "", "fixture account email")
		tier := fs.String("tier", "solo", "subscription tier: free, solo, pro, studio, or business")
		credits := fs.Int64("credits", 0, "initial credit balance")
		bundle := fs.String("bundle-key", "business_suite", "application bundle key")
		if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
			return err
		}
		if err := requireLocal(deps); err != nil {
			return err
		}
		return requestJSON(deps, "POST", "/dev/fixtures/seed", nil, map[string]any{"email": *email, "tier": *tier, "credit_balance": *credits, "bundle_key": *bundle})
	}}
}

func tokenCommand(deps support.Dependencies) cliapp.Command {
	return emailCommand(deps, "fixture-token", "Mint a short-lived consumer access token for a local fixture", "POST", "/dev/fixtures/token", nil)
}

func balanceCommand(deps support.Dependencies) cliapp.Command {
	return emailCommand(deps, "fixture-balance", "Print a local fixture credit balance", "GET", "/dev/fixtures/balance", func(email string) (url.Values, any) {
		return url.Values{"email": []string{email}}, nil
	})
}

func zeroCommand(deps support.Dependencies) cliapp.Command {
	return emailCommand(deps, "fixture-zero", "Set a local fixture credit balance to zero", "POST", "/dev/fixtures/zero", nil)
}

func emailCommand(deps support.Dependencies, name, description, method, path string, query func(string) (url.Values, any)) cliapp.Command {
	return cliapp.Command{Name: name, NeedsAPI: true, Description: description, Run: func(args []string) error {
		fs := flag.NewFlagSet(name, flag.ContinueOnError)
		email := fs.String("email", "", "fixture account email")
		if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
			return err
		}
		if err := requireLocal(deps); err != nil {
			return err
		}
		var values url.Values
		var body any
		if query != nil {
			values, body = query(*email)
		} else {
			body = map[string]any{"email": *email}
		}
		return requestJSON(deps, method, path, values, body)
	}}
}

func requireLocal(deps support.Dependencies) error {
	base := deps.CurrentAPIBase()
	parsed, err := url.Parse(base)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("fixture commands require a local API base")
	}
	host := parsed.Hostname()
	if host != "localhost" && net.ParseIP(host) == nil {
		return fmt.Errorf("fixture commands refuse non-local API base %q", base)
	}
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() {
		return fmt.Errorf("fixture commands refuse non-local API base %q", base)
	}
	return nil
}

func requestJSON(deps support.Dependencies, method, path string, query url.Values, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	def := support.EndpointDef{Name: "fixture", Method: method, Path: path, Description: "local fixture"}
	response, err := deps.Request(def, path, query, encoded)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(response))) > 0 {
		cliutil.PrintJSON(response)
	}
	return nil
}
