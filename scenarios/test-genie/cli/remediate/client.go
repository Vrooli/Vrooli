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
