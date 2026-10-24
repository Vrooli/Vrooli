package grants

import (
	"context"
	"fmt"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "grants",
		Description: "Authorize metadata-only credential delivery to paired nodes",
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a credential delivery grant", Run: runCreate},
			{Name: "list", Description: "List active delivery grants without values", Run: runList},
			{Name: "revoke", Description: "Revoke a delivery grant and purge its node copy", Run: runRevoke},
			{Name: "rotate", Description: "Increment an address generation and fan out the new value", Run: runRotate},
		},
	}
}

func runCreate(args []string) error {
	fs := support.NewFlagSet("grants create")
	nodeID := fs.String("node", "", "paired node identifier")
	logicalID := fs.String("identity", "", "credential logical identity")
	field := fs.String("field", "", "credential field")
	grantClass := fs.String("class", "", "grant class: infrastructure, per_install_generated, user_prompt, or remote_fetch")
	retention := fs.String("retention", "", "grant retention: durable or ephemeral")
	generation := fs.Int64("generation", 1, "current credential generation")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*nodeID) == "" || strings.TrimSpace(*logicalID) == "" || strings.TrimSpace(*field) == "" || strings.TrimSpace(*grantClass) == "" || strings.TrimSpace(*retention) == "" {
		return fmt.Errorf("grants create requires --node, --identity, --field, --class, and --retention")
	}
	client, err := credentials.NewGrantClient()
	if err != nil {
		return err
	}
	grant, err := client.Create(context.Background(), &credentialclient.CreateGrantRequest{NodeId: *nodeID, LogicalId: *logicalID, Field: *field, Class: *grantClass, Retention: *retention, Generation: *generation})
	if err != nil {
		return err
	}
	return support.PrintMutation(jsonOutput, grant, cliapp.MutationReport{Result: []string{fmt.Sprintf("grant %s created; only metadata was returned", grant.GetId())}})
}

func runList(args []string) error {
	fs := support.NewFlagSet("grants list")
	nodeID := fs.String("node", "", "filter to one paired node")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	client, err := credentials.NewGrantClient()
	if err != nil {
		return err
	}
	grants, err := client.List(context.Background(), *nodeID)
	if err != nil {
		return err
	}
	results := make([]string, 0, len(grants))
	for _, grant := range grants {
		results = append(results, fmt.Sprintf("%s | node=%s | %s:%s | class=%s | retention=%s | generation=%d | acked=%d", grant.GetId(), grant.GetNodeId(), grant.GetLogicalId(), grant.GetField(), grant.GetClass(), grant.GetRetention(), grant.GetGeneration(), grant.GetAckedGeneration()))
	}
	return support.PrintList(jsonOutput, grants, cliapp.ListReport{Summary: []string{fmt.Sprintf("Active grants: %d", len(grants)), "Values are never returned by this command"}, ResultsHeading: "Credential Grants", Results: results})
}

func runRevoke(args []string) error {
	fs := support.NewFlagSet("grants revoke")
	id := fs.String("id", "", "grant identifier")
	yes := fs.Bool("yes", false, "confirm revocation")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*id) == "" {
		return fmt.Errorf("grants revoke requires --id")
	}
	if !*yes {
		return fmt.Errorf("grants revoke requires --yes")
	}
	client, err := credentials.NewGrantClient()
	if err != nil {
		return err
	}
	grant, err := client.Revoke(context.Background(), *id)
	if err != nil {
		return err
	}
	return support.PrintMutation(jsonOutput, grant, cliapp.MutationReport{Result: []string{fmt.Sprintf("grant %s revoked; node purge was requested", *id)}})
}

func runRotate(args []string) error {
	fs := support.NewFlagSet("grants rotate")
	logicalID := fs.String("identity", "", "credential logical identity")
	field := fs.String("field", "", "credential field")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*logicalID) == "" || strings.TrimSpace(*field) == "" {
		return fmt.Errorf("grants rotate requires --identity and --field")
	}
	client, err := credentials.NewGrantClient()
	if err != nil {
		return err
	}
	response, err := client.Rotate(context.Background(), *logicalID, *field)
	if err != nil {
		return err
	}
	return support.PrintMutation(jsonOutput, response, cliapp.MutationReport{Result: []string{fmt.Sprintf("%s:%s rotated to generation %d; delivery is metadata-tracked", response.GetLogicalId(), response.GetField(), response.GetGeneration())}})
}
