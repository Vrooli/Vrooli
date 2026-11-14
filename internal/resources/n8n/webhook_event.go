package n8n

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxSyntheticWebhookEventBytes    = 64 << 10
	maxSyntheticWebhookExportBytes   = maxSyntheticWorkflowJSON
	maxSyntheticWebhookResponseBytes = 1 << 20
	syntheticWebhookPublishWait      = 20 * time.Second
)

var (
	syntheticWebhookPathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}(?:/[A-Za-z0-9_-]{1,64}){0,7}$`)
	// Publishing is allowed only for pinned built-in action nodes plus one
	// Webhook trigger. Unknown/community node definitions fail closed because
	// workflow JSON does not prove that their type has no trigger behavior.
	syntheticWebhookNodeAllowlist = map[string]struct{}{
		"n8n-nodes-base.webhook":          {},
		"n8n-nodes-base.httpRequest":      {},
		"n8n-nodes-base.set":              {},
		"n8n-nodes-base.code":             {},
		"n8n-nodes-base.if":               {},
		"n8n-nodes-base.switch":           {},
		"n8n-nodes-base.merge":            {},
		"n8n-nodes-base.noOp":             {},
		"n8n-nodes-base.respondToWebhook": {},
		"n8n-nodes-base.postgres":         {},
	}
)

const readSyntheticWorkflowProgram = "const fs=require('fs');const p=process.argv[1];const s=fs.statSync(p);if(!s.isFile()||s.size<1||s.size>10485760)process.exit(2);process.stdout.write(fs.readFileSync(p));"

// SyntheticWebhookReceipt reports content-free hashes, sizes, and HTTP status.
// It never returns the event, response body, route, workflow ID, or logs.
type SyntheticWebhookReceipt struct {
	SandboxID        string `json:"sandboxId"`
	RuntimeID        string `json:"runtimeId"`
	OperationID      string `json:"operationId"`
	Status           string `json:"status"`
	InputSHA256      string `json:"inputSha256"`
	InputBytes       int    `json:"inputBytes"`
	ResponseSHA256   string `json:"responseSha256,omitempty"`
	ResponseBytes    int    `json:"responseBytes,omitempty"`
	HTTPStatus       int    `json:"httpStatus,omitempty"`
	PublicationState string `json:"publicationState"`
	// Published is conservative: true means publication may still be registered
	// in n8n's asynchronous publication service. A successful unpublish CLI call
	// only confirms that the request was accepted, not that every worker removed
	// the route. Stopping the disposable runtime is the definitive cleanup.
	Published       bool `json:"published"`
	CleanupRequired bool `json:"cleanupRequired,omitempty"`
}

type syntheticWebhookRoute struct {
	Method string
	Path   string
}

type exportedSyntheticWorkflow struct {
	ID     string `json:"id"`
	Active *bool  `json:"active"`
	Nodes  []struct {
		Type       string         `json:"type"`
		Disabled   bool           `json:"disabled"`
		Parameters map[string]any `json:"parameters"`
	} `json:"nodes"`
}

// DispatchSyntheticWebhook derives a static POST route from the exact
// owner-imported workflow, validates its trigger graph, publishes only the
// owner-derived workflow ID, sends one synthetic JSON event to the n8n
// loopback endpoint, and always attempts to unpublish before returning.
func (o *Owner) DispatchSyntheticWebhook(ctx context.Context, sandboxID, runtimeID string, operationID uuid.UUID, eventJSON []byte) (receipt SyntheticWebhookReceipt, resultErr error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return SyntheticWebhookReceipt{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return SyntheticWebhookReceipt{}, ErrForeignObject
	}
	if operationID == uuid.Nil {
		return SyntheticWebhookReceipt{}, errors.New("webhook operation ID must be a nonzero UUID")
	}
	if _, err := validateSyntheticWebhookEvent(eventJSON); err != nil {
		return SyntheticWebhookReceipt{}, err
	}
	canonical, containerName, err := o.ownedRunningContainer(ctx, canonical, runtimeID)
	if err != nil {
		return SyntheticWebhookReceipt{}, err
	}
	runner, ok := o.runner().(quietCommandRunner)
	if !ok {
		return SyntheticWebhookReceipt{}, errors.New("runtime command runner cannot guarantee webhook output redaction")
	}
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		if err != nil {
			return SyntheticWebhookReceipt{}, err
		}
		return SyntheticWebhookReceipt{}, errors.New("n8n task runtime is not running")
	}
	runtime, err := o.runtimeFromContainer(ctx, canonical, runtimeID, container)
	if err != nil {
		return SyntheticWebhookReceipt{}, err
	}
	workflowID := workflowIDFor(canonical, runtimeID, operationID)
	stagePath := "/tmp/.vrooli-webhook-export-" + workflowID + "-" + uuid.NewString() + ".json"
	baseReceipt := SyntheticWebhookReceipt{SandboxID: canonical, RuntimeID: runtimeID, OperationID: operationID.String(), Status: "failed", PublicationState: "inactive"}
	attemptUnpublish := func() error {
		if err := unpublishSyntheticWorkflow(context.Background(), runner, containerName, workflowID); err != nil {
			baseReceipt.PublicationState = "unpublish-failed"
			baseReceipt.Published = true
			return err
		}
		baseReceipt.PublicationState = "unpublish-requested"
		baseReceipt.Published = true
		return nil
	}

	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "rm", "-f", "--", stagePath); err != nil && !isNotFound(err) {
		baseReceipt.Status = "lookup-cleanup-required"
		baseReceipt.CleanupRequired = true
		return baseReceipt, errors.New("could not prepare task tmpfs for workflow lookup")
	}
	defer func() {
		if cleanupErr := removeWorkflowInput(context.Background(), runner, containerName, stagePath); cleanupErr != nil {
			receipt.CleanupRequired = true
			if receipt.Status == "completed" {
				receipt.Status = "completed-cleanup-required"
			} else {
				receipt.Status += "-cleanup-required"
			}
			if resultErr == nil {
				resultErr = errors.New("synthetic webhook completed but tmpfs export cleanup failed")
			}
		}
	}()
	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "n8n", "export:workflow", "--id="+workflowID, "--output="+stagePath); err != nil {
		return baseReceipt, errors.New("owner-imported workflow lookup failed")
	}
	exportBytes, exceeded, err := runner.RunStdoutLimitQuiet(ctx, maxSyntheticWebhookExportBytes, "exec", "--user", "node", containerName, "node", "-e", readSyntheticWorkflowProgram, stagePath)
	if err != nil || exceeded || len(exportBytes) > maxSyntheticWebhookExportBytes {
		return baseReceipt, errors.New("owner-imported workflow lookup exceeded its bounded export")
	}
	route, err := validateExportedSyntheticWebhook(exportBytes, workflowID)
	if err != nil {
		cleanupErr := attemptUnpublish()
		baseReceipt.CleanupRequired = cleanupErr != nil
		return baseReceipt, err
	}

	publishCtx, cancel := context.WithTimeout(ctx, syntheticWorkflowRunTTL)
	defer cancel()
	if _, err := runner.RunQuiet(publishCtx, "exec", "--user", "node", containerName, "n8n", "publish:workflow", "--id="+workflowID); err != nil {
		cleanupErr := attemptUnpublish()
		baseReceipt.CleanupRequired = cleanupErr != nil
		if cleanupErr != nil {
			baseReceipt.Status = "publish-failed-cleanup-required"
		}
		return baseReceipt, errors.New("synthetic webhook workflow could not be published")
	}
	baseReceipt.Published = true
	baseReceipt.PublicationState = "publish-requested"

	publishWait := o.WebhookPublishWait
	if publishWait <= 0 {
		publishWait = syntheticWebhookPublishWait
	}
	wait := time.NewTimer(publishWait)
	select {
	case <-publishCtx.Done():
		wait.Stop()
		cleanupErr := attemptUnpublish()
		baseReceipt.CleanupRequired = cleanupErr != nil
		baseReceipt.Status = "publish-timeout"
		return baseReceipt, errors.New("synthetic webhook publication did not complete within its bounded wait")
	case <-wait.C:
	}

	eventHash := sha256.Sum256(eventJSON)
	baseReceipt.InputSHA256 = hex.EncodeToString(eventHash[:])
	baseReceipt.InputBytes = len(eventJSON)
	requestCtx, requestCancel := context.WithTimeout(publishCtx, syntheticWorkflowRunTTL)
	defer requestCancel()
	request, err := http.NewRequestWithContext(requestCtx, route.Method, strings.TrimRight(runtime.Endpoint, "/")+route.Path, bytes.NewReader(eventJSON))
	if err != nil {
		cleanupErr := attemptUnpublish()
		baseReceipt.CleanupRequired = cleanupErr != nil
		return baseReceipt, errors.New("could not construct owner-derived webhook request")
	}
	request.Header.Set("Content-Type", "application/json")
	response, requestErr := o.webhookClient().Do(request)
	if requestErr == nil {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxSyntheticWebhookResponseBytes+1))
		_ = response.Body.Close()
		if readErr == nil && len(body) <= maxSyntheticWebhookResponseBytes {
			responseHash := sha256.Sum256(body)
			baseReceipt.ResponseSHA256 = hex.EncodeToString(responseHash[:])
			baseReceipt.ResponseBytes = len(body)
			baseReceipt.HTTPStatus = response.StatusCode
		} else {
			requestErr = errors.New("response exceeded bounded capture")
		}
	}
	cleanupErr := attemptUnpublish()
	baseReceipt.CleanupRequired = cleanupErr != nil
	if requestErr != nil {
		baseReceipt.Status = "dispatch-failed"
		if cleanupErr != nil {
			baseReceipt.Status = "dispatch-failed-cleanup-required"
		}
		return baseReceipt, errors.New("synthetic webhook dispatch failed")
	}
	if cleanupErr != nil {
		baseReceipt.Status = "completed-cleanup-required"
		return baseReceipt, errors.New("synthetic webhook completed but unpublish cleanup failed")
	}
	baseReceipt.Status = "completed"
	if baseReceipt.HTTPStatus < 200 || baseReceipt.HTTPStatus >= 300 {
		baseReceipt.Status = "completed-with-http-error"
	}
	return baseReceipt, nil
}

func (o *Owner) webhookClient() *http.Client {
	client := o.client()
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if copy.Timeout < syntheticWorkflowRunTTL {
		copy.Timeout = syntheticWorkflowRunTTL
	}
	return &copy
}

func unpublishSyntheticWorkflow(ctx context.Context, runner quietCommandRunner, containerName, workflowID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := runner.RunQuiet(cleanupCtx, "exec", "--user", "node", containerName, "n8n", "unpublish:workflow", "--id="+workflowID); err != nil {
		return errors.New("synthetic webhook unpublish failed")
	}
	return nil
}

func validateSyntheticWebhookEvent(data []byte) (map[string]any, error) {
	if len(data) == 0 || len(data) > MaxSyntheticWebhookEventBytes {
		return nil, fmt.Errorf("synthetic webhook JSON must contain 1..%d bytes", MaxSyntheticWebhookEventBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkUniqueJSONValue(decoder); err != nil {
		return nil, fmt.Errorf("invalid synthetic webhook JSON: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("synthetic webhook JSON contains trailing content")
	}
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil || event == nil {
		return nil, errors.New("synthetic webhook input must be one JSON object")
	}
	if err := rejectHTTPMockSensitiveFields(event); err != nil {
		return nil, fmt.Errorf("synthetic webhook input rejected: %w", err)
	}
	if httpMockCredential.Match(data) {
		return nil, errors.New("synthetic webhook input contains a credential-shaped value")
	}
	for _, email := range httpMockEmail.FindAll(data, -1) {
		if !strings.HasSuffix(strings.ToLower(string(email)), ".invalid") {
			return nil, errors.New("synthetic webhook input contains a non-synthetic email")
		}
	}
	return event, nil
}

func validateExportedSyntheticWebhook(data []byte, expectedWorkflowID string) (syntheticWebhookRoute, error) {
	var exports []exportedSyntheticWorkflow
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := checkUniqueJSONValue(decoder); err != nil {
		return syntheticWebhookRoute{}, errors.New("workflow export has duplicate or malformed JSON fields")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return syntheticWebhookRoute{}, errors.New("workflow export contains trailing JSON")
	}
	if err := json.Unmarshal(data, &exports); err != nil || len(exports) != 1 {
		return syntheticWebhookRoute{}, errors.New("workflow export must contain exactly one workflow")
	}
	workflow := exports[0]
	if workflow.ID != expectedWorkflowID || workflow.Active == nil || *workflow.Active || len(workflow.Nodes) == 0 || len(workflow.Nodes) > 256 {
		return syntheticWebhookRoute{}, errors.New("workflow export identity or node count is invalid")
	}
	webhookCount := 0
	var route syntheticWebhookRoute
	for _, node := range workflow.Nodes {
		if _, allowed := syntheticWebhookNodeAllowlist[node.Type]; !allowed {
			return syntheticWebhookRoute{}, errors.New("workflow node type is outside the synthetic webhook execution profile")
		}
		if node.Disabled {
			continue
		}
		if node.Type == "n8n-nodes-base.httpRequest" {
			if err := validateSyntheticHTTPNode(node.Parameters); err != nil {
				return syntheticWebhookRoute{}, err
			}
		}
		if node.Type != "n8n-nodes-base.webhook" {
			continue
		}
		webhookCount++
		method, _ := node.Parameters["httpMethod"].(string)
		path, _ := node.Parameters["path"].(string)
		multiple, _ := node.Parameters["multipleMethods"].(bool)
		auth, _ := node.Parameters["authentication"].(string)
		responseMode, _ := node.Parameters["responseMode"].(string)
		if multiple || method != http.MethodPost || !syntheticWebhookPathPattern.MatchString(path) || auth != "" && auth != "none" || responseMode != "lastNode" {
			return syntheticWebhookRoute{}, errors.New("workflow Webhook must use one static unauthenticated POST route with last-node response mode")
		}
		route = syntheticWebhookRoute{Method: method, Path: "/webhook/" + path}
	}
	if webhookCount != 1 {
		return syntheticWebhookRoute{}, errors.New("workflow must contain exactly one enabled Webhook trigger")
	}
	return route, nil
}

func validateSyntheticHTTPNode(parameters map[string]any) error {
	raw, _ := parameters["url"].(string)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Port() != strconv.Itoa(syntheticHTTPMockPort) || parsed.User != nil || parsed.Fragment != "" || !httpMockHostPattern.MatchString(strings.ToLower(parsed.Hostname())) {
		return errors.New("workflow HTTP Request must target an exact synthetic.invalid host on the fixed HTTP mock port")
	}
	if containsSyntheticHTTPProxyOption(parameters) {
		return errors.New("workflow HTTP Request proxy settings are not allowed")
	}
	return nil
}

func containsSyntheticHTTPProxyOption(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
			if normalized == "proxy" || normalized == "httpproxy" || normalized == "proxyurl" {
				return true
			}
			if containsSyntheticHTTPProxyOption(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsSyntheticHTTPProxyOption(child) {
				return true
			}
		}
	}
	return false
}
