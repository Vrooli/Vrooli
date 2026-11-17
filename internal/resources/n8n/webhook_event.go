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
	syntheticWebhookPathPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}(?:/[A-Za-z0-9_-]{1,64}){0,7}$`)
	syntheticWebhookSelectorPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._ -]{0,127}$`)
	syntheticWebhookEmailPattern    = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.invalid$`)
	// Publishing is allowed only for pinned built-in action nodes plus one
	// Webhook trigger. Unknown/community node definitions fail closed because
	// workflow JSON does not prove that their type has no trigger behavior.
	syntheticWebhookNodeAllowlist = map[string]struct{}{
		"n8n-nodes-base.webhook":          {},
		"n8n-nodes-base.httpRequest":      {},
		"n8n-nodes-base.googleSheets":     {},
		"n8n-nodes-base.gmail":            {},
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

const dispatchSyntheticWebhookProgram = `const fs=require("fs"),http=require("http"),crypto=require("crypto");const x=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));if(!x||typeof x.path!=="string"||!x.path.startsWith("/webhook/")||!["GET","POST","PUT","PATCH","DELETE","HEAD"].includes(x.method)||!x.event||typeof x.event!=="object"||Array.isArray(x.event))process.exit(2);const body=Buffer.from(JSON.stringify(x.event));if(body.length>65536)process.exit(2);function send(attempt){const hash=crypto.createHash("sha256");let size=0,failed=false;const request=http.request({host:"127.0.0.1",port:5678,path:x.path,method:x.method,headers:{"content-type":"application/json","content-length":body.length}},response=>{response.on("data",chunk=>{size+=chunk.length;if(size>1048576){failed=true;request.destroy();return}hash.update(chunk)});response.on("end",()=>{if(failed)return;if(response.statusCode===404&&attempt<20){setTimeout(()=>send(attempt+1),500);return}process.stdout.write(JSON.stringify({httpStatus:response.statusCode,responseBytes:size,responseSha256:hash.digest("hex")}))})});request.on("error",()=>{process.exitCode=1});request.end(body)}send(0);`

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
	Published               bool   `json:"published"`
	CleanupRequired         bool   `json:"cleanupRequired,omitempty"`
	FailureClass            string `json:"failureClass,omitempty"`
	FailureDiagnosticSHA256 string `json:"failureDiagnosticSha256,omitempty"`
	FailureDiagnosticBytes  int    `json:"failureDiagnosticBytes,omitempty"`
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
	if runtime.AccessMode != "host-loopback" && runtime.AccessMode != "owner-cli" {
		return SyntheticWebhookReceipt{
			SandboxID: canonical, RuntimeID: runtimeID, OperationID: operationID.String(),
			Status: "unsupported-access-mode", PublicationState: "inactive",
		}, errors.New("synthetic webhook dispatch requires an owner-managed local execution path")
	}
	if runtime.AccessMode == "host-loopback" && strings.TrimSpace(runtime.Endpoint) == "" {
		return SyntheticWebhookReceipt{}, errors.New("synthetic webhook dispatch requires an active loopback endpoint")
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
		setSyntheticWebhookFailure(&baseReceipt, "tmpfs-prepare-error", []byte(err.Error()))
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
		setSyntheticWebhookFailure(&baseReceipt, "workflow-export-error", []byte(err.Error()))
		return baseReceipt, errors.New("owner-imported workflow lookup failed")
	}
	exportBytes, exceeded, err := runner.RunStdoutLimitQuiet(ctx, maxSyntheticWebhookExportBytes, "exec", "--user", "node", containerName, "node", "-e", readSyntheticWorkflowProgram, stagePath)
	if err != nil || exceeded || len(exportBytes) > maxSyntheticWebhookExportBytes {
		class := "workflow-export-read-error"
		if exceeded || len(exportBytes) > maxSyntheticWebhookExportBytes {
			class = "workflow-export-read-limit"
		}
		setSyntheticWebhookFailure(&baseReceipt, class, boundedSyntheticWebhookError(err))
		return baseReceipt, errors.New("owner-imported workflow lookup exceeded its bounded export")
	}
	route, err := validateExportedSyntheticWebhook(exportBytes, workflowID)
	if err != nil {
		setSyntheticWebhookFailure(&baseReceipt, "workflow-route-invalid", []byte(err.Error()))
		cleanupErr := attemptUnpublish()
		baseReceipt.CleanupRequired = cleanupErr != nil
		return baseReceipt, err
	}

	publishCtx, cancel := context.WithTimeout(ctx, syntheticWorkflowRunTTL)
	defer cancel()
	var publishStdout, publishStderr []byte
	var publishExceeded bool
	var publishErr error
	if diagnosticRunner, ok := o.runner().(quietDiagnosticCommandRunner); ok {
		publishStdout, publishStderr, publishExceeded, publishErr = diagnosticRunner.RunStdoutStderrLimitQuiet(publishCtx, 8<<10, 8<<10, "exec", "--user", "node", containerName, "n8n", "publish:workflow", "--id="+workflowID)
	} else {
		_, publishErr = runner.RunQuiet(publishCtx, "exec", "--user", "node", containerName, "n8n", "publish:workflow", "--id="+workflowID)
	}
	if publishErr != nil || publishExceeded {
		failureClass := classifySyntheticWebhookPublishFailure(publishStdout, publishStderr, publishExceeded)
		setSyntheticWebhookFailure(&baseReceipt, failureClass, append(publishStdout, publishStderr...))
		if baseReceipt.FailureDiagnosticSHA256 == "" {
			setSyntheticWebhookFailure(&baseReceipt, failureClass, boundedSyntheticWebhookError(publishErr))
		}
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
	var requestErr error
	if runtime.AccessMode == "owner-cli" {
		var eventCleanupRequired bool
		baseReceipt.HTTPStatus, baseReceipt.ResponseBytes, baseReceipt.ResponseSHA256, eventCleanupRequired, requestErr = dispatchSyntheticWebhookInside(requestCtx, runner, containerName, workflowID, route, eventJSON)
		baseReceipt.CleanupRequired = eventCleanupRequired
	} else {
		var request *http.Request
		request, requestErr = http.NewRequestWithContext(requestCtx, route.Method, strings.TrimRight(runtime.Endpoint, "/")+route.Path, bytes.NewReader(eventJSON))
		if requestErr == nil {
			request.Header.Set("Content-Type", "application/json")
			var response *http.Response
			response, requestErr = o.webhookClient().Do(request)
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
		}
	}
	cleanupErr := attemptUnpublish()
	baseReceipt.CleanupRequired = baseReceipt.CleanupRequired || cleanupErr != nil
	if requestErr != nil {
		setSyntheticWebhookFailure(&baseReceipt, "webhook-dispatch-error", []byte(requestErr.Error()))
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

func setSyntheticWebhookFailure(receipt *SyntheticWebhookReceipt, failureClass string, diagnostic []byte) {
	if receipt == nil || failureClass == "" {
		return
	}
	receipt.FailureClass = failureClass
	if len(diagnostic) > 8192 {
		diagnostic = diagnostic[:8192]
	}
	if len(diagnostic) == 0 {
		return
	}
	digest := sha256.Sum256(diagnostic)
	receipt.FailureDiagnosticSHA256 = hex.EncodeToString(digest[:])
	receipt.FailureDiagnosticBytes = len(diagnostic)
}

func classifySyntheticWebhookPublishFailure(stdout, stderr []byte, exceeded bool) string {
	if exceeded {
		return "workflow-publish-output-limit"
	}
	diagnostic := strings.ToLower(string(append(append([]byte(nil), stdout...), stderr...)))
	switch {
	case strings.Contains(diagnostic, "unknown command"), strings.Contains(diagnostic, "command not found"), strings.Contains(diagnostic, "unrecognized command"):
		return "workflow-publish-cli-unavailable"
	case strings.Contains(diagnostic, "not found"), strings.Contains(diagnostic, "does not exist"):
		return "workflow-publish-workflow-missing"
	case strings.Contains(diagnostic, "project"), strings.Contains(diagnostic, "owner"), strings.Contains(diagnostic, "permission"):
		return "workflow-publish-ownership-rejected"
	default:
		return "workflow-publish-runtime-error"
	}
}

func boundedSyntheticWebhookError(err error) []byte {
	if err == nil {
		return nil
	}
	message := err.Error()
	if len(message) > 8192 {
		message = message[:8192]
	}
	return []byte(message)
}

func dispatchSyntheticWebhookInside(ctx context.Context, runner quietCommandRunner, containerName, workflowID string, route syntheticWebhookRoute, eventJSON []byte) (httpStatus, responseBytes int, responseSHA string, cleanupRequired bool, resultErr error) {
	event, err := validateSyntheticWebhookEvent(eventJSON)
	if err != nil {
		return 0, 0, "", false, err
	}
	input, err := json.Marshal(struct {
		Path   string         `json:"path"`
		Method string         `json:"method"`
		Event  map[string]any `json:"event"`
	}{Path: route.Path, Method: route.Method, Event: event})
	if err != nil || len(input) > MaxSyntheticWebhookEventBytes+1024 {
		return 0, 0, "", false, errors.New("synthetic webhook dispatch input exceeded its bounded size")
	}
	stagePath := "/tmp/.vrooli-webhook-event-" + workflowID + "-" + uuid.NewString() + ".json"
	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "rm", "-f", "--", stagePath); err != nil && !isNotFound(err) {
		return 0, 0, "", false, errors.New("could not prepare task tmpfs for webhook event")
	}
	defer func() {
		if cleanupErr := removeWorkflowInput(context.Background(), runner, containerName, stagePath); cleanupErr != nil {
			cleanupRequired = true
			if resultErr == nil {
				resultErr = errors.New("synthetic webhook event completed but tmpfs cleanup failed")
			}
		}
	}()
	if _, err := runner.RunInputQuiet(ctx, input, "exec", "--interactive", "--user", "node", containerName, "node", "-e", stageWorkflowFileProgram, stagePath); err != nil {
		return 0, 0, "", false, errors.New("could not stage bounded synthetic webhook event")
	}
	output, exceeded, err := runner.RunStdoutLimitQuiet(ctx, maxHTTPMockOutputBytes, "exec", "--user", "node", containerName, "node", "-e", dispatchSyntheticWebhookProgram, stagePath)
	if err != nil || exceeded {
		return 0, 0, "", false, errors.New("owner-local webhook dispatch failed or exceeded its output bound")
	}
	var receipt struct {
		HTTPStatus     int    `json:"httpStatus"`
		ResponseBytes  int    `json:"responseBytes"`
		ResponseSHA256 string `json:"responseSha256"`
	}
	if err := json.Unmarshal(output, &receipt); err != nil || receipt.HTTPStatus < 100 || receipt.HTTPStatus > 599 || receipt.ResponseBytes < 0 || receipt.ResponseBytes > maxSyntheticWebhookResponseBytes || !validHTTPMockDigest(receipt.ResponseSHA256) {
		return 0, 0, "", false, errors.New("owner-local webhook dispatch returned an invalid bounded receipt")
	}
	return receipt.HTTPStatus, receipt.ResponseBytes, receipt.ResponseSHA256, false, nil
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
	responseNodeCount := 0
	responseNodeMode := false
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
		if node.Type == "n8n-nodes-base.googleSheets" || node.Type == "n8n-nodes-base.gmail" {
			if err := validateSyntheticWebhookIntegrationNode(node.Type, node.Parameters); err != nil {
				return syntheticWebhookRoute{}, err
			}
		}
		if node.Type == "n8n-nodes-base.respondToWebhook" {
			responseNodeCount++
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
		responseNodeMode = responseMode == "responseNode"
		if multiple || !syntheticWebhookMethodAllowed(method) || !syntheticWebhookPathPattern.MatchString(path) || auth != "" && auth != "none" || responseMode != "lastNode" && !responseNodeMode {
			return syntheticWebhookRoute{}, errors.New("workflow Webhook must use one static unauthenticated supported-method route with a supported response mode")
		}
		route = syntheticWebhookRoute{Method: method, Path: "/webhook/" + path}
	}
	if webhookCount != 1 {
		return syntheticWebhookRoute{}, errors.New("workflow must contain exactly one enabled Webhook trigger")
	}
	if responseNodeMode && responseNodeCount == 0 {
		return syntheticWebhookRoute{}, errors.New("responseNode mode requires at least one enabled Respond to Webhook node")
	}
	return route, nil
}

// Native service nodes in the webhook owner profile are admitted by static
// operation only. Their real vendor APIs are replaced with exact task routes;
// wildcard credentials, selectors and arbitrary API operations are rejected.
func validateSyntheticWebhookIntegrationNode(nodeType string, parameters map[string]any) error {
	resource, _ := parameters["resource"].(string)
	operation, _ := parameters["operation"].(string)
	switch nodeType {
	case "n8n-nodes-base.googleSheets":
		if resource != "sheet" || operation != "append" {
			return errors.New("synthetic webhook Google Sheets supports only Sheet Append")
		}
		documentID := syntheticSelector(parameters["documentId"])
		sheetName := syntheticSelector(parameters["sheetName"])
		if !syntheticWebhookSelectorPattern.MatchString(documentID) || !syntheticWebhookSelectorPattern.MatchString(sheetName) {
			return errors.New("synthetic webhook Google Sheets requires bounded static document and sheet selectors")
		}
	case "n8n-nodes-base.gmail":
		if resource != "message" || operation != "send" {
			return errors.New("synthetic webhook Gmail supports only Message Send")
		}
		recipient, _ := parameters["sendTo"].(string)
		subject, _ := parameters["subject"].(string)
		message, _ := parameters["message"].(string)
		emailType, _ := parameters["emailType"].(string)
		if !syntheticWebhookEmailPattern.MatchString(recipient) || len(recipient) > 254 || len(subject) == 0 || len(subject) > 256 || strings.HasPrefix(subject, "=") || len(message) == 0 || len(message) > 8192 || strings.HasPrefix(message, "=") || emailType != "text" && emailType != "html" {
			return errors.New("synthetic webhook Gmail requires bounded static synthetic recipient and message fields")
		}
	default:
		return errors.New("native service node is outside the synthetic webhook profile")
	}
	return nil
}

func syntheticSelector(value any) string {
	if direct, ok := value.(string); ok {
		return direct
	}
	if locator, ok := value.(map[string]any); ok {
		selected, _ := locator["value"].(string)
		return selected
	}
	return ""
}

func syntheticWebhookMethodAllowed(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
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
