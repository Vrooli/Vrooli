package remediate

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const UsageLine = "test-genie remediate <scenario> --execution <uuid> --findings <stable-id,...> --role <role-ref> [--context text]"

var ArgsSchema = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario owning the completed execution"}}, Flags: []cliapp.Flag{{Name: "execution", Description: "Completed Test Genie execution UUID"}, {Name: "findings", Description: "Comma-separated stable finding IDs"}, {Name: "requirements", Description: "Comma-separated requirement IDs"}, {Name: "role", Description: "Portable Agent Manager role reference"}, {Name: "context", Description: "Optional operator context"}}}

func HelpText() string {
	return "Launch one evidence-bound remediation job from a completed execution.\n\nExample:\n  test-genie remediate demo --execution <uuid> --findings afid:123 --role code.default"
}

type Request struct {
	SourceExecutionID string   `json:"sourceExecutionId"`
	FindingIDs        []string `json:"findingIds"`
	RequirementIDs    []string `json:"requirementIds,omitempty"`
	RoleRef           string   `json:"roleRef"`
	AdditionalContext string   `json:"additionalContext,omitempty"`
}
type Response struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Scenario string `json:"scenario"`
}

func Primitive(client *Client) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (Response, error) {
		request, err := requestFromContext(ctx)
		if err != nil {
			return Response{}, err
		}
		response, _, err := client.Create(ctx.Positional("scenario"), request)
		return response, err
	}, func(_ cliapp.OperationContext, response Response) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Remediation job %s created for %s", response.ID, response.Scenario)}, Changes: []string{"Status: " + response.Status}}
	})
}
func requestFromContext(ctx cliapp.OperationContext) (Request, error) {
	request := Request{SourceExecutionID: strings.TrimSpace(ctx.Flag("execution")), FindingIDs: cliutil.ParseCSV(ctx.Flag("findings")), RequirementIDs: cliutil.ParseCSV(ctx.Flag("requirements")), RoleRef: strings.TrimSpace(ctx.Flag("role")), AdditionalContext: strings.TrimSpace(ctx.Flag("context"))}
	if request.SourceExecutionID == "" || request.RoleRef == "" || (len(request.FindingIDs) == 0 && len(request.RequirementIDs) == 0) {
		return Request{}, fmt.Errorf("execution, role, and at least one finding or requirement selector are required")
	}
	return request, nil
}
