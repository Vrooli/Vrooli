package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const httpMockSandboxID = "5e33a6f0-b1bf-4f5d-9315-95ba60d9e025"

func validHTTPMockConfigJSON() []byte {
	return []byte(`{"name":"workflow-apis","version":"1.0.0","fixtures":[{"host":"CRM.SYNTHETIC.INVALID","method":"post","path":"/v1/contacts","query":"sort=created&limit=2","status":201,"contentType":"application/json","body":"{\"id\":\"synthetic-1\",\"email\":\"person@example.invalid\"}"},{"host":"sheets.synthetic.invalid","method":"GET","path":"/v1/rows","status":200,"contentType":"application/json","body":"[{\"row\":1}]"}]}`)
}

func TestValidateHTTPMockConfigRejectsUnsafeFixtures(t *testing.T) {
	valid := HTTPMockConfig{Name: "workflow-apis", Version: "1.0.0", Fixtures: []HTTPMockFixture{{Host: "CRM.SYNTHETIC.INVALID", Method: "post", Path: "/v1/contacts", Status: 201, ContentType: "application/json", Body: `{"id":"synthetic-1","email":"person@example.invalid"}`}}}
	if err := ValidateHTTPMockConfig(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	invalid := []struct {
		name    string
		fixture HTTPMockFixture
	}{
		{"external hostname", HTTPMockFixture{Host: "api.example.com", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}},
		{"URL instead of alias", HTTPMockFixture{Host: "https://crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}},
		{"port in host", HTTPMockFixture{Host: "crm.synthetic.invalid:18080", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`}},
		{"query path", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x?q=1", Status: 200, Body: `{}`}},
		{"encoded path", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/%2e%2e/admin", Status: 200, Body: `{}`}},
		{"traversal path", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/../admin", Status: 200, Body: `{}`}},
		{"redirect", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 302, Body: `{}`}},
		{"secret field", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{"access_token":"synthetic"}`}},
		{"credential-shaped value", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, ContentType: "text/plain", Body: `Authorization: Bearer abcdefghijklmnop`}},
		{"real email", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{"email":"person@example.com"}`}},
		{"reserved health route", HTTPMockFixture{Host: "crm.synthetic.invalid", Method: "GET", Path: "/__vrooli_health", Status: 200, Body: `{}`}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			config := HTTPMockConfig{Name: "workflow-apis", Version: "1.0.0", Fixtures: []HTTPMockFixture{test.fixture}}
			if err := ValidateHTTPMockConfig(config); err == nil {
				t.Fatal("unsafe fixture was accepted")
			}
		})
	}
	duplicate := HTTPMockConfig{Name: "workflow-apis", Version: "1.0.0", Fixtures: []HTTPMockFixture{
		{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/x", Status: 200, Body: `{}`},
		{Host: "CRM.synthetic.invalid", Method: "get", Path: "/v1/x", Status: 201, Body: `{}`},
	}}
	if err := ValidateHTTPMockConfig(duplicate); err == nil {
		t.Fatal("duplicate normalized route was accepted")
	}
	large := HTTPMockConfig{Name: "workflow-apis", Version: "1.0.0", Fixtures: []HTTPMockFixture{{Host: "crm.synthetic.invalid", Method: "GET", Path: "/large", Status: 200, ContentType: "text/plain", Body: strings.Repeat("x", maxSyntheticHTTPMockBodyBytes+1)}}}
	if err := ValidateHTTPMockConfig(large); err == nil {
		t.Fatal("oversized fixture body was accepted")
	}
}

func TestHTTPMockQueryNormalizationMatchesSandboxExactQueryContract(t *testing.T) {
	config, canonical, _, err := parseHTTPMockConfig(validHTTPMockConfigJSON())
	if err != nil || config.Fixtures[0].Query != "limit=2&sort=created" || !bytes.Contains(canonical, []byte(`"query":"limit=2\u0026sort=created"`)) {
		t.Fatalf("public config was not normalized before owner routing: fixture=%+v err=%v json=%s", config.Fixtures[0], err, canonical)
	}
	got, err := normalizeHTTPMockQuery("sort=created&limit=2")
	if err != nil || got != "limit=2&sort=created" {
		t.Fatalf("normalized query = %q, %v", got, err)
	}
	got, err = normalizeHTTPMockQuery("q=a%20b")
	if err != nil || got != "q=a+b" {
		t.Fatalf("space encoding normalization = %q, %v", got, err)
	}
	for _, query := range []string{
		"id=1&id=2",
		"q=%xx",
		"q=x;y",
		"access_token=synthetic",
		"q=Bearer%20abcdefghijklmnop",
		"email=person%40example.com",
		"q=" + strings.Repeat("x", 513),
		"q=" + strings.Repeat("x", maxSyntheticHTTPMockQueryBytes),
		"invalid key=x",
	} {
		if got, err := normalizeHTTPMockQuery(query); err == nil {
			t.Errorf("unsafe query accepted (%d bytes): %q", len(query), got)
		}
	}
	var maxPairs strings.Builder
	for i := range maxSyntheticHTTPMockQueryPairs + 1 {
		if i > 0 {
			maxPairs.WriteByte('&')
		}
		fmt.Fprintf(&maxPairs, "k%d=v", i)
	}
	if _, err := normalizeHTTPMockQuery(maxPairs.String()); err == nil {
		t.Fatal("query over the unique-pair cap was accepted")
	}
}

func TestHTTPMockRoutesDistinguishCanonicalQueriesAndRejectEquivalentDuplicates(t *testing.T) {
	config := HTTPMockConfig{Name: "workflow-apis", Version: "1.0.0", Fixtures: []HTTPMockFixture{
		{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/contacts", Query: "sort=created&limit=2", Status: 200, Body: `[]`},
		{Host: "crm.synthetic.invalid", Method: "GET", Path: "/v1/contacts", Query: "sort=updated&limit=2", Status: 200, Body: `[]`},
	}}
	if err := ValidateHTTPMockConfig(config); err != nil {
		t.Fatalf("distinct exact queries on the same route were rejected: %v", err)
	}
	if config.Fixtures[0].Query != "limit=2&sort=created" {
		t.Fatalf("fixture query was not canonicalized: %q", config.Fixtures[0].Query)
	}
	config.Fixtures[1].Query = "limit=2&sort=created"
	if err := ValidateHTTPMockConfig(config); err == nil {
		t.Fatal("equivalent normalized query routes were not treated as duplicates")
	}
	if !strings.Contains(syntheticHTTPMockServerProgram, "canonicalQuery(target.slice(queryIndex+1))") || !strings.Contains(syntheticHTTPMockServerProgram, `f.path+"?"+(f.query||"")`) || !strings.Contains(canonicalHTTPMockQueryProgram, "values.has(key)") {
		t.Fatal("fixed server does not match normalized exact queries or reject duplicate request keys")
	}
}

func TestParseHTTPMockConfigRejectsOversizeDuplicateKeysAndUnknownFields(t *testing.T) {
	if _, _, _, err := parseHTTPMockConfig([]byte(`{"name":"a","name":"b"}`)); err == nil {
		t.Fatal("duplicate JSON keys were accepted")
	}
	if _, _, _, err := parseHTTPMockConfig([]byte(`{"name":"workflow-apis","version":"1.0.0","fixtures":[],"image":"evil"}`)); err == nil {
		t.Fatal("unknown config field was accepted")
	}
	if _, _, _, err := parseHTTPMockConfig(make([]byte, MaxSyntheticHTTPMockConfigBytes+1)); err == nil {
		t.Fatal("oversized config was accepted")
	}
}

func TestStartHTTPMockUsesOnlyPinnedImageFixedSourceAndOwnedInternalNetwork(t *testing.T) {
	created, running, serverStarted, removed := false, false, false, false
	var mockRuntimeID, configDigest, aliasLabel string
	inspectedImage := ImageRef
	var staged []byte
	createCount := 0
	containerName := httpMockContainerName(httpMockSandboxID)
	_, taskNetwork := objectNames(httpMockSandboxID)
	runner := &fakeRunner{}
	runner.call = func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "info --format {{.DockerRootDir}}":
			return []byte("/var/lib/docker"), nil
		case len(args) >= 2 && args[0] == "container" && args[1] == "inspect":
			if args[2] == containerName || args[2] == "httpmock-container" {
				if !created {
					return nil, errors.New("No such container")
				}
				return httpMockContainerJSONWithImage(httpMockSandboxID, mockRuntimeID, configDigest, aliasLabel, running, inspectedImage), nil
			}
			name, _ := objectNames(httpMockSandboxID)
			if args[2] == name {
				return syntheticN8nInspect(httpMockSandboxID), nil
			}
		case len(args) >= 2 && args[0] == "network" && args[1] == "inspect":
			return httpMockNetworkJSON(httpMockSandboxID), nil
		case len(args) > 0 && args[0] == "create":
			createCount++
			for _, arg := range args {
				if arg == "n8nio/n8n:latest" || arg == "--publish" || arg == "--volume" {
					t.Fatalf("unsafe create arg %q: %#v", arg, args)
				}
			}
			if !containsArgPair(args, "--network", taskNetwork) || !containsArg(args, "--read-only") || !containsArgPair(args, "--cap-drop", "ALL") || containsArg(args, "--publish") {
				t.Fatalf("container does not meet private runtime contract: %#v", args)
			}
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "--label" {
					key, value, ok := strings.Cut(args[i+1], "=")
					if !ok {
						continue
					}
					switch key {
					case labelRuntimeID:
						mockRuntimeID = value
					case labelHTTPMockConfigHash:
						configDigest = value
					case labelHTTPMockAliases:
						aliasLabel = value
					}
				}
			}
			if args[len(args)-1] != httpMockKeepaliveProgram || !containsArgPair(args, "--entrypoint", "node") {
				t.Fatalf("container did not use the fixed keepalive source: %#v", args)
			}
			if !containsArg(args, ImageRef) {
				t.Fatalf("container did not use pinned n8n image: %#v", args)
			}
			created = true
			return []byte("httpmock-container"), nil
		case len(args) > 0 && args[0] == "start":
			running = true
			return nil, nil
		case len(args) > 0 && args[0] == "exec" && containsArg(args, "--detach"):
			if !containsArg(args, syntheticHTTPMockServerProgram) {
				t.Fatalf("owner passed a non-fixed server source: %#v", args)
			}
			serverStarted = true
			return nil, nil
		case len(args) > 0 && args[0] == "exec":
			if !serverStarted {
				return nil, errors.New("server not started")
			}
			return []byte("ready"), nil
		case len(args) > 0 && args[0] == "stop":
			running, serverStarted = false, false
			return nil, nil
		case len(args) > 0 && args[0] == "rm":
			if !containsArg(args, "--volumes") {
				t.Fatalf("owner did not remove ephemeral volumes: %#v", args)
			}
			created, removed = false, true
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", joined)
		}
		return nil, fmt.Errorf("unexpected command: %s", joined)
	}
	runner.inputCall = func(_ context.Context, data []byte, args ...string) ([]byte, error) {
		if !containsArgPair(args, "--interactive", "--user") || !containsArg(args, stageHTTPMockProgram) {
			t.Fatalf("fixture config was not staged by the fixed stdin writer: %#v", args)
		}
		staged = append([]byte(nil), data...)
		return nil, nil
	}
	owner := &Owner{Runner: runner, Image: ImageRef}
	receipt, err := owner.StartHTTPMock(context.Background(), httpMockSandboxID, validHTTPMockConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "attached" || !receipt.NoEgressVerified || receipt.Port != syntheticHTTPMockPort || receipt.CompanionRuntimeID != mockRuntimeID || len(receipt.NetworkAliases) != 2 || !serverStarted || len(staged) == 0 {
		t.Fatalf("unexpected owner receipt/state: receipt=%+v server=%t staged=%d", receipt, serverStarted, len(staged))
	}
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "pull" {
			t.Fatalf("HTTP mock implicitly pulled an image: %#v", runner.calls)
		}
	}
	if strings.Contains(syntheticHTTPMockServerProgram, "fetch(") || strings.Contains(syntheticHTTPMockServerProgram, "http.request(") || !strings.Contains(syntheticHTTPMockServerProgram, "no synthetic fixture") {
		t.Fatal("fixed HTTP server has forwarding behavior or no fail-closed unmatched route")
	}
	if _, err := owner.StartHTTPMock(context.Background(), httpMockSandboxID, validHTTPMockConfigJSON()); err != nil {
		t.Fatalf("idempotent start: %v", err)
	}
	if createCount != 1 {
		t.Fatalf("same config created %d HTTP mock containers", createCount)
	}
	inspectedImage = "foreign/image:latest"
	if _, err := owner.StopHTTPMock(context.Background(), httpMockSandboxID, receipt.CompanionRuntimeID); !errors.Is(err, ErrForeignObject) {
		t.Fatalf("stop accepted a foreign image with copied owner labels: %v", err)
	}
	if removed {
		t.Fatal("stop removed the object after its inspected image failed owner verification")
	}
	inspectedImage = ImageRef
	if _, err := owner.StopHTTPMock(context.Background(), httpMockSandboxID, receipt.CompanionRuntimeID); err != nil {
		t.Fatalf("stop HTTP mock: %v", err)
	}
	if !removed {
		t.Fatal("owned HTTP mock container was not removed")
	}
	if _, err := owner.StopHTTPMock(context.Background(), httpMockSandboxID, receipt.CompanionRuntimeID); err != nil {
		t.Fatalf("idempotent stop: %v", err)
	}
}

func httpMockContainerJSON(sandboxID, runtimeID, digest, aliasList string, running bool) []byte {
	return httpMockContainerJSONWithImage(sandboxID, runtimeID, digest, aliasList, running, ImageRef)
}

func httpMockContainerJSONWithImage(sandboxID, runtimeID, digest, aliasList string, running bool, image string) []byte {
	_, network := objectNames(sandboxID)
	aliases := append(parseHTTPMockAliases(aliasList), httpMockContainerName(sandboxID))
	labels := map[string]string{labelResource: "n8n", labelSandboxID: sandboxID, labelRuntimeID: runtimeID, labelObjectKind: "http-mock", labelCompanionResource: "n8n-http-mock", labelSandboxOwner: sandboxID, labelSandboxRunID: runtimeID, labelSandboxDecl: "true", labelHTTPMockConfigHash: digest, labelHTTPMockAliases: aliasList}
	value := map[string]any{"Id": "httpmock-container", "Config": map[string]any{"Image": image, "Labels": labels}, "State": map[string]any{"Running": running}, "NetworkSettings": map[string]any{"Ports": map[string]any{"18080/tcp": nil}, "Networks": map[string]any{network: map[string]any{"Aliases": aliases}}}}
	data, _ := json.Marshal([]any{value})
	return data
}

func httpMockNetworkJSON(sandboxID string) []byte {
	_, network := objectNames(sandboxID)
	value := []map[string]any{{"Name": network, "Driver": "bridge", "Internal": true, "Labels": map[string]string{labelResource: "n8n", labelSandboxID: sandboxID, labelRuntimeID: runtimeIDFor(sandboxID), labelObjectKind: "network"}, "Containers": map[string]any{}}}
	data, _ := json.Marshal(value)
	return data
}
