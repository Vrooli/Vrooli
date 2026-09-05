package remediate

import "github.com/vrooli/cli-core/cliapp"

var jobArgs = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario owning the remediation job"}, {Name: "job_id", Required: true, Description: "Remediation job id"}}}
var listArgs = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario owning remediation jobs"}}}

// Register exposes the complete durable lifecycle. The default preserves the
// concise create shorthand while making inspection, cancellation, recovery,
// and verification first-class commands with the same JSON transport result.
func Register(client *Client) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "remediate", Description: "Create and operate evidence-bound remediation jobs", NeedsAPI: true, DefaultSubcommand: "create", Subcommands: []cliapp.Command{
		cliapp.Command{Name: "create", Description: "Create and launch an evidence-bound remediation job", Usage: UsageLine, HelpText: HelpText(), Args: ArgsSchema, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(Primitive(client)),
		cliapp.Command{Name: "list", Description: "List remediation jobs for a scenario", Args: listArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(ListPrimitive(client)),
		cliapp.Command{Name: "show", Description: "Show one remediation job and its lifecycle evidence", Args: jobArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(GetPrimitive(client)),
		cliapp.Command{Name: "cancel", Description: "Cancel an active remediation job", Args: jobArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(ActionPrimitive(client, "cancel")),
		cliapp.Command{Name: "recover", Description: "Recover a durable pending launch without duplicating remote work", Args: jobArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(ActionPrimitive(client, "recover")),
		cliapp.Command{Name: "retry", Description: "Start a new retry attempt after a terminal launch failure", Args: jobArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(ActionPrimitive(client, "retry")),
		cliapp.Command{Name: "verify", Description: "Start server-owned verification for an agent-completed job", Args: jobArgs, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}.WithPrimitive(ActionPrimitive(client, "verify")),
	}}
}
