package remediate

import (
	"net/http"

	"test-genie/cli/internal/apijson"

	"github.com/vrooli/cli-core/cliutil"
)

type Client struct{ api *cliutil.APIClient }

func NewClient(api *cliutil.APIClient) *Client { return &Client{api: api} }
func (c *Client) Create(scenario string, request Request) (Response, []byte, error) {
	body, err := c.api.Request(http.MethodPost, "/api/v1/scenarios/"+scenario+"/remediation/jobs", nil, request)
	if err != nil {
		return Response{}, nil, err
	}
	response, err := apijson.Parse[Response](body, "parse remediation job")
	return response, body, err
}

type JobListResponse struct {
	Items []Response `json:"items"`
	Count int        `json:"count"`
}

func (c *Client) List(scenario string) (JobListResponse, []byte, error) {
	body, err := c.api.Request(http.MethodGet, "/api/v1/scenarios/"+scenario+"/remediation/jobs", nil, nil)
	if err != nil {
		return JobListResponse{}, nil, err
	}
	response, err := apijson.Parse[JobListResponse](body, "parse remediation jobs")
	return response, body, err
}

func (c *Client) Get(scenario, id string) (Response, []byte, error) {
	body, err := c.api.Request(http.MethodGet, "/api/v1/scenarios/"+scenario+"/remediation/jobs/"+id, nil, nil)
	if err != nil {
		return Response{}, nil, err
	}
	response, err := apijson.Parse[Response](body, "parse remediation job")
	return response, body, err
}

func (c *Client) Action(scenario, id, action string) (Response, []byte, error) {
	body, err := c.api.Request(http.MethodPost, "/api/v1/scenarios/"+scenario+"/remediation/jobs/"+id+"/"+action, nil, nil)
	if err != nil {
		return Response{}, nil, err
	}
	response, err := apijson.Parse[Response](body, "parse remediation job action")
	return response, body, err
}
