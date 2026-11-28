package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This opt-in owner proof runs public-derived #17164 on pinned n8n using only
// task-private HTTP fixtures. It seeds a method defect in the synthetic Gemini
// request, records the native failure, restores the public method, and verifies
// the Sheets/Slack handoff routes. It does not call Workspace Sandbox.
func TestLivePublic17164NativeOwnerRepairRun(t *testing.T) {
	if os.Getenv("F7_LIVE_PUBLIC_17164_OWNER") != "1" {
		t.Skip("set F7_LIVE_PUBLIC_17164_OWNER=1 to run the isolated n8n owner integration")
	}
	outPath := strings.TrimSpace(os.Getenv("F7_NATIVE_17164_RECEIPT_OUT"))
	if outPath == "" {
		t.Fatal("F7_NATIVE_17164_RECEIPT_OUT must name the content-free receipt path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	binary := strings.TrimSpace(os.Getenv("F7_VROOLI_BIN"))
	if binary == "" {
		binary = "/home/matthalloran8/.vrooli/bin/vrooli"
	}
	owner := N8NWorkflowRuntimeCLI{Binary: binary, Environment: os.Environ}
	sandboxID := uuid.NewString()
	publicJSON := readFixture(t, "public-workflows/17164.json")
	profileJSON := readFixture(t, "public-workflow-17164-profile.json")
	sourceHash := sha256.Sum256([]byte(publicJSON))
	if hex.EncodeToString(sourceHash[:]) != "2788a548183fb6a9cf96f9c16a590ab7da4e87ae7850fac7c0dd014e6194f715" {
		t.Fatalf("sanitized public #17164 digest changed: %x", sourceHash)
	}
	workflow, _, _, err := normalizeWorkflow(publicJSON)
	if err != nil || len(workflow.Nodes) != 9 || !nativeScheduledWorkflowSupported(workflow) {
		t.Fatalf("public #17164 is no longer supported by bounded native schedule execution: nodes=%d err=%v", len(workflow.Nodes), err)
	}
	simulator, err := NewDescriptionSimulator(profileJSON, false)
	if err != nil {
		t.Fatalf("load public #17164 API descriptions: %v", err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil || len(profile.Operations) == 0 {
		t.Fatalf("derive exact task HTTP mock profile: operations=%d err=%v", len(profile.Operations), err)
	}
	fixtures := make([]HTTPMockRoute, 0, len(profile.Operations)+1)
	addresses := make([]HTTPMockOperationAddress, 0, len(profile.Operations)+1)
	var modelRoute *HTTPMockRoute
	var rssURL, sheetsURL, slackURL string
	for _, operation := range profile.Operations {
		route := operation.Route
		fixtures = append(fixtures, route)
		addressURL, addressErr := syntheticHTTPRouteURL(route, "http", "18080")
		if addressErr != nil {
			t.Fatalf("compile exact #17164 companion route: %v", addressErr)
		}
		addresses = append(addresses, HTTPMockOperationAddress{Selector: operation.Selector, Method: route.Method, URL: addressURL})
		selector := operation.Selector
		switch {
		case selector.NodeType == "n8n-nodes-base.rssFeedRead" && selector.InputMatch["feedUrl"] == "https://feeds.synthetic.invalid/one":
			rssURL, addressErr = syntheticHTTPRouteURL(route, "https", "")
		case selector.NodeType == "n8n-nodes-base.googleSheets" && selector.Operation == "append":
			sheetsURL, addressErr = syntheticHTTPRouteURL(route, "https", "")
		case selector.NodeType == "n8n-nodes-base.slack" && selector.Operation == "post":
			slackURL, addressErr = syntheticHTTPRouteURL(route, "https", "")
		case route.Host == "generativelanguage-googleapis-co.synthetic.invalid" && strings.EqualFold(route.Method, http.MethodPost):
			routeCopy := route
			modelRoute = &routeCopy
		}
		if addressErr != nil {
			t.Fatalf("compile #17164 task workflow URL: %v", addressErr)
		}
	}
	if modelRoute == nil || rssURL == "" || sheetsURL == "" || slackURL == "" {
		t.Fatalf("synthetic profile does not bind all expected #17164 routes: rss=%t sheets=%t slack=%t model=%t", rssURL != "", sheetsURL != "", slackURL != "", modelRoute != nil)
	}
	faultRoute := *modelRoute
	faultRoute.Method = http.MethodGet
	faultRoute.Status = http.StatusServiceUnavailable
	faultRoute.Body = `{"error":"synthetic seeded method defect"}`
	fixtures = append(fixtures, faultRoute)
	faultURL, err := syntheticHTTPRouteURL(faultRoute, "http", "18080")
	if err != nil {
		t.Fatalf("compile seeded #17164 route: %v", err)
	}
	faultAddress := HTTPMockOperationAddress{Method: faultRoute.Method, URL: faultURL}
	addresses = append(addresses, faultAddress)
	config := struct {
		Name     string          `json:"name"`
		Version  string          `json:"version"`
		Fixtures []HTTPMockRoute `json:"fixtures"`
	}{Name: taskSandboxProfileName, Version: taskSandboxProfileVersion, Fixtures: fixtures}
	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	profileHash := sha256.Sum256(configJSON)

	buildTaskCopy := func(modelMethod string) (string, error) {
		var doc map[string]any
		if err := json.Unmarshal([]byte(publicJSON), &doc); err != nil {
			return "", err
		}
		nodes, _ := doc["nodes"].([]any)
		for _, rawNode := range nodes {
			node, _ := rawNode.(map[string]any)
			if node == nil {
				continue
			}
			kind, _ := node["type"].(string)
			switch kind {
			case "n8n-nodes-base.rssFeedRead":
				node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
				node["parameters"] = map[string]any{"authentication": "none", "method": "GET", "url": rssURL, "options": map[string]any{}}
			case "n8n-nodes-base.googleSheets":
				node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
				node["parameters"] = map[string]any{"authentication": "none", "method": "POST", "url": sheetsURL, "sendBody": true, "specifyBody": "json", "jsonBody": "={{ $json }}", "options": map[string]any{}}
			case "n8n-nodes-base.slack":
				node["type"], node["typeVersion"] = "n8n-nodes-base.httpRequest", 4.2
				node["parameters"] = map[string]any{"authentication": "none", "method": "POST", "url": slackURL, "sendBody": true, "specifyBody": "json", "jsonBody": "={{ {title: $('Parse Social Posts').item.json.title, link: $('Parse Social Posts').item.json.link, twitter: $('Parse Social Posts').item.json.twitter, linkedin: $('Parse Social Posts').item.json.linkedin} }}", "options": map[string]any{}}
			case "n8n-nodes-base.httpRequest":
				parameters, _ := node["parameters"].(map[string]any)
				if parameters == nil {
					return "", fmt.Errorf("public model node has no parameter object")
				}
				parameters["authentication"] = "none"
				delete(parameters, "nodeCredentialType")
				parameters["method"] = modelMethod
			}
		}
		encoded, err := json.Marshal(doc)
		if err != nil {
			return "", err
		}
		rewritten, rewriteErr := rewriteWorkflowHTTPRequestsForSandbox(string(encoded), addresses)
		if rewriteErr != nil {
			routes := make([]string, 0, len(addresses))
			for _, address := range addresses {
				routes = append(routes, address.Method+" "+address.URL)
			}
			return "", fmt.Errorf("rewrite synthetic task copy (%d route candidates: %s): %w", len(routes), strings.Join(routes, "; "), rewriteErr)
		}
		return rewritten, nil
	}

	seededWorkflow, err := buildTaskCopy(http.MethodGet)
	if err != nil {
		t.Fatalf("build seeded native #17164 task copy: %v", err)
	}

	lease, err := owner.Start(ctx, sandboxID)
	if err != nil {
		t.Fatalf("start pinned n8n runtime: %v", err)
	}
	var companionID string
	runtimeStopped, companionStopped := false, false
	var cleanupErr error
	phase := "companion-start"
	receipt := map[string]any{
		"kind":                       "native-n8n-resource-owner-defect-repair-attempt",
		"workspaceSandboxGateCredit": false,
		"publicWorkflowId":           "17164",
		"sourceSha256":               hex.EncodeToString(sourceHash[:]),
		"taskScopeId":                sandboxID,
		"runtimeId":                  lease.RuntimeID,
		"runtimeVersion":             "2.48.0",
		"profileSha256":              hex.EncodeToString(profileHash[:]),
		"fixtureCount":               len(fixtures),
		"externalEffects":            false,
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanupCancel()
		if companionID != "" {
			hitsJSON, hitsErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			var hits HTTPMockRouteHitsReceipt
			if hitsErr == nil && json.Unmarshal(hitsJSON, &hits) == nil && hits.Status == "available" {
				receipt["routeHitsAtTermination"] = hits
			} else {
				receipt["routeHitsReadStatus"] = "unavailable"
			}
		}
		if companionID != "" && !companionStopped {
			_, stopErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
			cleanupErr = errors.Join(cleanupErr, stopErr)
			companionStopped = stopErr == nil
		}
		if !runtimeStopped {
			stopErr := owner.Stop(cleanupCtx, lease)
			cleanupErr = errors.Join(cleanupErr, stopErr)
			runtimeStopped = stopErr == nil
		}
		statusJSON, statusErr := owner.run(cleanupCtx, nil, "resource", "n8n", "runtime", "status", "--sandbox-id", sandboxID, "--json")
		var status struct {
			Status string `json:"status"`
		}
		if statusErr == nil {
			statusErr = json.Unmarshal(statusJSON, &status)
		}
		receipt["phaseAtTermination"] = phase
		receipt["cleanup"] = map[string]any{
			"runtimeStopCommandSucceeded":   runtimeStopped,
			"companionStopCommandSucceeded": companionStopped,
			"ownerStatus":                   status.Status,
			"ownerStatusReadSucceeded":      statusErr == nil,
		}
		receipt["cleanupVerified"] = cleanupErr == nil && statusErr == nil && status.Status == "stopped"
		encoded, encodeErr := json.MarshalIndent(receipt, "", "  ")
		if encodeErr == nil {
			writeErr := os.WriteFile(outPath, append(encoded, '\n'), 0o600)
			if writeErr == nil {
				writeErr = os.Chmod(outPath, 0o600)
			}
			if writeErr != nil {
				t.Errorf("write content-free native repair attempt receipt: %v", writeErr)
			}
		} else {
			t.Errorf("encode content-free native repair attempt receipt: %v", encodeErr)
		}
		if cleanupErr != nil || statusErr != nil || status.Status != "stopped" {
			t.Errorf("isolated owner cleanup/status incomplete: runtimeStopped=%t companionStopped=%t status=%s cleanupErr=%v statusErr=%v", runtimeStopped, companionStopped, status.Status, cleanupErr, statusErr)
		}
	}()
	startJSON, err := owner.run(ctx, configJSON, "resource", "n8n", "runtime", "companion-start-http-mock", "--sandbox-id", sandboxID, "--config-stdin", "--json")
	if err != nil {
		t.Fatalf("start task-local RSS/Gemini/Sheets/Slack mock: %v", err)
	}
	var companion struct {
		CompanionRuntimeID string `json:"companionRuntimeId"`
		Status             string `json:"status"`
		NoEgressVerified   bool   `json:"noEgressVerified"`
		FixtureCount       int    `json:"fixtureCount"`
		ConfigSHA256       string `json:"configSha256"`
	}
	if err := json.Unmarshal(startJSON, &companion); err != nil || companion.Status != "attached" || !companion.NoEgressVerified || companion.FixtureCount != len(fixtures) || companion.ConfigSHA256 != hex.EncodeToString(profileHash[:]) {
		t.Fatalf("task mock receipt invalid: err=%v", err)
	}
	companionID = companion.CompanionRuntimeID
	receipt["noEgressVerified"] = companion.NoEgressVerified
	receipt["companionRuntimeId"] = companion.CompanionRuntimeID
	phase = "seeded-execution"

	seededHash := sha256.Sum256([]byte(seededWorkflow))
	seededStarted := time.Now()
	seededEvidence, seededErr := owner.ImportAndExecuteScheduled(ctx, lease, seededWorkflow)
	seededElapsed := time.Since(seededStarted).Milliseconds()
	if seededErr == nil || !strings.Contains(seededErr.Error(), "failureClass=") || !strings.Contains(seededErr.Error(), "diagnosticSha256=") {
		t.Fatalf("seeded native method defect did not fail with bounded diagnostic evidence: evidence=%+v err=%v", seededEvidence, seededErr)
	}
	failureClass := strings.SplitN(strings.SplitN(seededErr.Error(), "failureClass=", 2)[1], " ", 2)[0]
	failureDigest := strings.SplitN(strings.SplitN(seededErr.Error(), "diagnosticSha256=", 2)[1], " ", 2)[0]
	if len(failureDigest) != 64 {
		t.Fatalf("seeded failure diagnostic is not SHA-256: %s", failureDigest)
	}
	receipt["seededFailure"] = map[string]any{"taskCopySha256": hex.EncodeToString(seededHash[:]), "failureClass": failureClass, "diagnosticSha256": failureDigest, "elapsedMs": seededElapsed, "triggerKind": seededEvidence.TriggerKind, "dispatchMode": seededEvidence.DispatchMode}
	seedHitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read seeded failure route hits: %v", err)
	}
	var seedHits HTTPMockRouteHitsReceipt
	if err := json.Unmarshal(seedHitsJSON, &seedHits); err != nil || seedHits.Status != "available" {
		t.Fatalf("seeded failure route evidence is invalid: err=%v", err)
	}
	receipt["seededFailure"].(map[string]any)["routeHits"] = seedHits
	seededFaultRouteHits := exactHTTPMockRouteHitCount(seedHits, faultRoute)
	receipt["seededFailure"].(map[string]any)["exactFaultRouteHits"] = seededFaultRouteHits
	if !seededNativeFaultRouteSatisfied(seedHits, faultRoute) {
		t.Fatalf("seeded n8n error did not reach the exact Gemini fault route; routeHits=%d", seededFaultRouteHits)
	}

	correctedWorkflow, err := buildTaskCopy(http.MethodPost)
	if err != nil {
		t.Fatalf("build repaired native #17164 task copy: %v", err)
	}
	correctedHash := sha256.Sum256([]byte(correctedWorkflow))
	phase = "repaired-execution"
	correctedStarted := time.Now()
	execution, correctedErr := owner.ImportAndExecuteScheduled(ctx, lease, correctedWorkflow)
	correctedElapsed := time.Since(correctedStarted).Milliseconds()
	if correctedErr != nil {
		receipt["repairedExecutionFailure"] = map[string]any{"taskCopySha256": hex.EncodeToString(correctedHash[:]), "elapsedMs": correctedElapsed, "diagnostic": contentFreeN8NFailureEvidence(correctedErr)}
		t.Fatalf("execute repaired public #17164 graph on pinned n8n: %v", correctedErr)
	}
	if execution.Status != "completed" || execution.TriggerKind != string(TriggerScheduled) || execution.DispatchMode != "manual-one-shot" || execution.NodeCount < 9 || execution.ItemCount < 2 || execution.FailedRuns != 0 {
		t.Fatalf("native repaired schedule execution is incomplete: %+v", execution)
	}
	finalHitsJSON, err := owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-route-hits", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("read final exact route counts: %v", err)
	}
	var finalHits HTTPMockRouteHitsReceipt
	if err := json.Unmarshal(finalHitsJSON, &finalHits); err != nil || finalHits.Status != "available" || finalHits.TotalHits == 0 {
		t.Fatalf("final route receipt is invalid: err=%v", err)
	}
	receipt["repairedExecution"] = map[string]any{"taskCopySha256": hex.EncodeToString(correctedHash[:]), "status": execution.Status, "elapsedMs": correctedElapsed, "triggerKind": execution.TriggerKind, "dispatchMode": execution.DispatchMode, "nodeCount": execution.NodeCount, "runCount": execution.RunCount, "itemCount": execution.ItemCount, "failedRuns": execution.FailedRuns, "outputSha256": execution.OutputSHA, "outputBytes": execution.OutputBytes}
	receipt["finalRouteHits"] = finalHits
	receipt["simulatedServices"] = []string{"RSS feed", "Gemini generation", "Google Sheets append", "Slack post"}
	if err := owner.Stop(ctx, lease); err != nil {
		t.Fatalf("stop isolated n8n runtime: %v", err)
	}
	runtimeStopped = true
	_, err = owner.run(ctx, nil, "resource", "n8n", "runtime", "companion-stop-http-mock", "--sandbox-id", sandboxID, "--companion-runtime-id", companionID, "--json")
	if err != nil {
		t.Fatalf("stop task HTTP mock: %v", err)
	}
	companionStopped, companionID = true, ""
	phase = "completed"
}

func contentFreeN8NFailureEvidence(err error) map[string]string {
	result := map[string]string{"kind": "failure-without-bounded-diagnostic"}
	if err == nil {
		return result
	}
	text := err.Error()
	if strings.Contains(text, "failureClass=") {
		class := strings.SplitN(strings.SplitN(text, "failureClass=", 2)[1], " ", 2)
		if len(class) > 0 && class[0] != "" && len(class[0]) <= 64 {
			valid := true
			for _, char := range class[0] {
				if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_') {
					valid = false
					break
				}
			}
			if valid {
				result["failureClass"] = class[0]
			}
		}
	}
	if marker := "diagnosticSha256="; strings.Contains(text, marker) {
		digest := strings.SplitN(strings.SplitN(text, marker, 2)[1], " ", 2)[0]
		decoded, decodeErr := hex.DecodeString(digest)
		if decodeErr == nil && len(decoded) == sha256.Size {
			result["diagnosticSha256"] = strings.ToLower(digest)
			delete(result, "kind")
		}
	}
	return result
}

func TestContentFreeN8NFailureEvidenceKeepsOnlyBoundedEnvelope(t *testing.T) {
	secret := "api-key=synthetic-secret"
	plain, err := json.Marshal(contentFreeN8NFailureEvidence(errors.New(secret)))
	if err != nil || strings.Contains(string(plain), secret) || strings.Contains(string(plain), "synthetic-secret") {
		t.Fatalf("stderr-only detail escaped content-free receipt: %s err=%v", plain, err)
	}

	digest := strings.Repeat("a", sha256.Size*2)
	bounded, err := json.Marshal(contentFreeN8NFailureEvidence(errors.New("failureClass=cli-stderr-diagnostic diagnosticSha256=" + digest + " capturedDiagnosticBytes=112 diagnosticTruncated=false")))
	if err != nil || !strings.Contains(string(bounded), `"failureClass":"cli-stderr-diagnostic"`) || !strings.Contains(string(bounded), digest) || strings.Contains(string(bounded), "capturedDiagnosticBytes") {
		t.Fatalf("bounded failure envelope was not selectively retained: %s err=%v", bounded, err)
	}

	invalid, err := json.Marshal(contentFreeN8NFailureEvidence(errors.New("failureClass=../../secret diagnosticSha256=not-a-digest")))
	if err != nil || strings.Contains(string(invalid), "../../secret") || strings.Contains(string(invalid), "not-a-digest") {
		t.Fatalf("invalid failure metadata escaped content-free receipt: %s err=%v", invalid, err)
	}
}
