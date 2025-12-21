package workflows

import (
	"encoding/json"
	"fmt"
	"net/url"

	"browser-automation-studio/cli/internal/appctx"
)

type listResponse struct {
	Workflows []workflowSummary `json:"workflows"`
}

type workflowSummary struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	FolderPath string           `json:"folder_path"`
	CreatedAt  string           `json:"created_at"`
	Workflow   *workflowSummary `json:"workflow"`
}

type workflowDetail struct {
	Workflow struct {
		ID             string          `json:"id"`
		Name           string          `json:"name"`
		FolderPath     string          `json:"folder_path"`
		FlowDefinition json.RawMessage `json:"flow_definition"`
	} `json:"workflow"`
}

type createResponse struct {
	ID       string `json:"id"`
	Workflow struct {
		ID string `json:"id"`
	} `json:"workflow"`
}

type projectListResponse struct {
	Projects []struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	} `json:"projects"`
}

type executeResponse struct {
	ExecutionID string `json:"execution_id"`
}

type workflowValidationResponse struct {
	Valid bool `json:"valid"`
	Stats struct {
		NodeCount int `json:"node_count"`
		EdgeCount int `json:"edge_count"`
	} `json:"stats"`
	Errors   []validationIssue `json:"errors"`
	Warnings []validationIssue `json:"warnings"`
}

type validationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"node_id"`
	Pointer string `json:"pointer"`
}

func listWorkflows(ctx *appctx.Context) ([]workflowSummary, []byte, error) {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/workflows"), nil)
	if err != nil {
		return nil, nil, err
	}
	var parsed listResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, body, err
	}
	return parsed.Workflows, body, nil
}

func getWorkflow(ctx *appctx.Context, id string) (workflowDetail, error) {
	var detail workflowDetail
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/workflows/"+id), nil)
	if err != nil {
		return detail, err
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		return detail, err
	}
	return detail, nil
}

func listWorkflowVersions(ctx *appctx.Context, workflowID string, limit int) ([]byte, error) {
	values := url.Values{}
	values.Set("limit", fmt.Sprintf("%d", limit))
	return ctx.Core.APIClient.Get(ctx.APIPath("/workflows/"+workflowID+"/versions"), values)
}

func getWorkflowVersion(ctx *appctx.Context, workflowID, version string) ([]byte, error) {
	return ctx.Core.APIClient.Get(ctx.APIPath("/workflows/"+workflowID+"/versions/"+version), nil)
}

func restoreWorkflowVersion(ctx *appctx.Context, workflowID, version string, payload any) ([]byte, error) {
	return ctx.Core.APIClient.Request("POST", ctx.APIPath("/workflows/"+workflowID+"/versions/"+version+"/restore"), nil, payload)
}
