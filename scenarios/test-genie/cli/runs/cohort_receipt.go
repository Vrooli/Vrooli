package runs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cohortReceiptEndpoint = "/api/v1/experience-cohort-receipts"

var cohortReceiptHTTPClient = http.DefaultClient

func storeExperienceCohortCall(apiClient *cliutil.APIClient, requestJSON string) (json.RawMessage, error) {
	if apiClient == nil {
		return nil, errors.New("test-genie API client is required")
	}
	baseURL := strings.TrimRight(apiClient.BaseURL(), "/")
	if baseURL == "" {
		return nil, errors.New("test-genie API base URL is not configured")
	}
	var body any
	if err := json.Unmarshal([]byte(requestJSON), &body); err != nil {
		return nil, fmt.Errorf("decode cohort receipt request: %w", err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+cohortReceiptEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cohortReceiptHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("store experience cohort receipt: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("store experience cohort receipt: API returned %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}
	if !json.Valid(payload) {
		return nil, errors.New("Test Genie returned invalid stored cohort receipt JSON")
	}
	var stored struct {
		Receipt struct {
			SchemaVersion int    `json:"schemaVersion"`
			ReceiptID     string `json:"receiptId"`
			Scenarios     []any  `json:"scenarios"`
		} `json:"receipt"`
		Artifact struct {
			ID     string `json:"id"`
			SHA256 string `json:"sha256"`
		} `json:"artifact"`
	}
	if err := json.Unmarshal(payload, &stored); err != nil || stored.Receipt.SchemaVersion != 1 || stored.Receipt.ReceiptID == "" || len(stored.Receipt.Scenarios) == 0 || stored.Artifact.ID == "" || stored.Artifact.SHA256 == "" {
		return nil, errors.New("Test Genie returned an incomplete stored cohort receipt")
	}
	return append(json.RawMessage(nil), payload...), nil
}

func storeExperienceCohortPrimitive(apiClient *cliutil.APIClient) cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(
		func(ctx cliapp.OperationContext) (json.RawMessage, error) {
			return storeExperienceCohortCall(apiClient, ctx.Flag("request-json"))
		},
		func(ctx cliapp.RunContext, payload json.RawMessage) error {
			_, err := ctx.Stdout().Write(append(payload, '\n'))
			return err
		},
		func(ctx cliapp.RunContext, payload json.RawMessage) error {
			_, err := ctx.Stdout().Write(append(payload, '\n'))
			return err
		},
	)
}
