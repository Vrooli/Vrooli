package remediate

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const UsageLine = "test-genie remediate create <scenario> --execution <uuid> --findings <stable-id,...> --role <role-ref> [--context text]"

var ArgsSchema = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario owning the completed execution"}}, Flags: []cliapp.Flag{{Name: "execution", Description: "Completed Test Genie execution UUID"}, {Name: "findings", Description: "Comma-separated stable finding IDs"}, {Name: "requirements", Description: "Comma-separated requirement IDs"}, {Name: "role", Description: "Portable Agent Manager role reference"}, {Name: "context", Description: "Optional operator context"}}}

func HelpText() string {
	return "Create and operate evidence-bound remediation jobs.\n\nExample:\n  test-genie remediate create demo --execution <uuid> --findings afid:123 --role code.default"
}

type Request struct {
	SourceExecutionID string   `json:"sourceExecutionId"`
	FindingIDs        []string `json:"findingIds"`
	RequirementIDs    []string `json:"requirementIds,omitempty"`
	RoleRef           string   `json:"roleRef"`
	AdditionalContext string   `json:"additionalContext,omitempty"`
}
type Response struct {
	ID                     string    `json:"id"`
	Status                 string    `json:"status"`
	Scenario               string    `json:"scenario"`
	SelectedFindingIDs     []string  `json:"selectedFindingIds,omitempty"`
	SelectedRequirementIDs []string  `json:"selectedRequirementIds,omitempty"`
	SourceHash             string    `json:"sourceHash,omitempty"`
	SelectionHash          string    `json:"selectionHash,omitempty"`
	Attempts               []Attempt `json:"attempts,omitempty"`
}
type Attempt struct {
	Kind   string `json:"kind"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
	RunID  string `json:"runId,omitempty"`
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

func ListPrimitive(client *Client) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (JobListResponse, error) {
		response, _, err := client.List(ctx.Positional("scenario"))
		return response, err
	}, func(_ cliapp.OperationContext, response JobListResponse) cliapp.MutationReport {
		items := make([]string, 0, len(response.Items))
		for _, job := range response.Items {
			items = append(items, fmt.Sprintf("%s  %s  %s", job.ID, job.Status, job.Scenario))
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("%d remediation job(s)", response.Count)}, Changes: items}
	})
}

func GetPrimitive(client *Client) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (Response, error) {
		response, _, err := client.Get(ctx.Positional("scenario"), ctx.Positional("job_id"))
		return response, err
	}, jobReport("Remediation job"))
}

func ActionPrimitive(client *Client, action string) cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (Response, error) {
		response, _, err := client.Action(ctx.Positional("scenario"), ctx.Positional("job_id"), action)
		return response, err
	}, jobReport("Remediation "+action))
}

func jobReport(label string) func(cliapp.OperationContext, Response) cliapp.MutationReport {
	return func(_ cliapp.OperationContext, response Response) cliapp.MutationReport {
		changes := []string{"Status: " + response.Status, fmt.Sprintf("Selected findings: %d; requirements: %d", len(response.SelectedFindingIDs), len(response.SelectedRequirementIDs))}
		for _, attempt := range response.Attempts {
			changes = append(changes, fmt.Sprintf("%s %s%s", attempt.Kind, attempt.State, nonEmpty(" · ", attempt.Detail)))
		}
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s %s", label, response.ID)}, Changes: changes}
	}
}

func nonEmpty(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
}
func requestFromContext(ctx cliapp.OperationContext) (Request, error) {
	request := Request{SourceExecutionID: strings.TrimSpace(ctx.Flag("execution")), FindingIDs: cliutil.ParseCSV(ctx.Flag("findings")), RequirementIDs: cliutil.ParseCSV(ctx.Flag("requirements")), RoleRef: strings.TrimSpace(ctx.Flag("role")), AdditionalContext: strings.TrimSpace(ctx.Flag("context"))}
	if request.SourceExecutionID == "" || request.RoleRef == "" || (len(request.FindingIDs) == 0 && len(request.RequirementIDs) == 0) {
		return Request{}, fmt.Errorf("execution, role, and at least one finding or requirement selector are required")
	}
	return request, nil
}
