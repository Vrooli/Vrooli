package rehearsals

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestTaskHTTPMockWorkflowCompilerCoversUnlikePublicTriggerShapes(t *testing.T) {
	cases := []struct {
		id      string
		profile string
		trigger TriggerKind
	}{
		{id: "6338", profile: "public-workflow-6338-profile.json", trigger: TriggerManual},
		{id: "16904", profile: "public-workflow-16904-profile.json", trigger: TriggerWebhook},
		{id: "17164", profile: "public-workflow-17164-profile.json", trigger: TriggerScheduled},
		{id: "17831", profile: "public-workflow-17831-profile.json", trigger: TriggerWebhook},
	}
	for _, test := range cases {
		t.Run(test.id, func(t *testing.T) {
			raw := readFixture(t, "public-workflows/"+test.id+".json")
			workflow, _, _, err := normalizeWorkflow(raw)
			if err != nil {
				t.Fatalf("normalize public workflow: %v", err)
			}
			if len(workflow.TriggerKinds) == 0 || workflow.TriggerKinds[0] != test.trigger {
				t.Fatalf("fixture trigger changed: got %v want %s", workflow.TriggerKinds, test.trigger)
			}
			simulator, err := NewDescriptionSimulator(readFixture(t, test.profile), false)
			if err != nil {
				t.Fatalf("load public API descriptions: %v", err)
			}
			profile, err := simulator.ExportHTTPMockProfile()
			if err != nil {
				t.Fatalf("derive synthetic API routes: %v", err)
			}
			addresses, err := HTTPMockOperationAddresses(profile)
			if err != nil {
				t.Fatalf("derive fixed task-local route addresses: %v", err)
			}
			compiled, err := compileTaskHTTPMockWorkflow(raw, workflow, profile, addresses)
			if err != nil {
				t.Fatalf("compile public workflow to exact synthetic routes: %v", err)
			}
			var document struct {
				Nodes []struct {
					ID         string         `json:"id"`
					Name       string         `json:"name"`
					Type       string         `json:"type"`
					Parameters map[string]any `json:"parameters"`
				} `json:"nodes"`
			}
			if err := json.Unmarshal([]byte(compiled), &document); err != nil {
				t.Fatalf("decode task-local copy: %v", err)
			}
			if len(document.Nodes) != len(workflow.Nodes) {
				t.Fatalf("compiler changed graph size: task nodes=%d public nodes=%d", len(document.Nodes), len(workflow.Nodes))
			}
			compiledRequests := 0
			for _, node := range document.Nodes {
				if node.Type != "n8n-nodes-base.httpRequest" {
					continue
				}
				compiledRequests++
				text, ok := node.Parameters["url"].(string)
				if !ok {
					t.Fatalf("compiled HTTP Request %q lacks URL", node.Name)
				}
				if strings.HasPrefix(text, "={{ (") {
					if method, _ := node.Parameters["method"].(string); !strings.EqualFold(method, "GET") {
						t.Fatalf("dynamic selector route must be read-only: %s", method)
					}
					for _, operation := range profile.Operations {
						want, _ := syntheticHTTPRouteURL(operation.Route, "http", "18080")
						if strings.Contains(text, want) {
							continue
						}
						if operation.Selector.NodeType == "n8n-nodes-base.rssFeedRead" {
							t.Fatalf("dynamic RSS route table omitted exact profile route %q", want)
						}
					}
					continue
				}
				target, err := url.Parse(text)
				if err != nil || target.Scheme != "http" || target.Port() != "18080" || !strings.HasSuffix(target.Hostname(), ".synthetic.invalid") {
					t.Fatalf("compiled request escaped fixed task route surface: %q", text)
				}
				method, _ := node.Parameters["method"].(string)
				matched := false
				for _, operation := range profile.Operations {
					want, _ := syntheticHTTPRouteURL(operation.Route, "http", "18080")
					if text == want && strings.EqualFold(method, operation.Route.Method) {
						matched = true
						break
					}
				}
				if !matched {
					t.Fatalf("compiled request is not an exact profile route: %q", text)
				}
			}
			if compiledRequests == 0 {
				t.Fatal("compiler produced no task-local provider requests")
			}
			if test.trigger == TriggerScheduled {
				task, err := rewriteScheduledWorkflowForOneShotDispatch(compiled, workflow)
				if err != nil || !strings.Contains(task, `"type":"n8n-nodes-base.manualTrigger"`) {
					t.Fatalf("schedule CLI task copy was not made CLI-startable: err=%v", err)
				}
				task, err = injectSyntheticInputAfterEntry(task, map[string]any{"feedUrl": "https://feeds.synthetic.invalid/approved.xml"}, TriggerManual)
				if err != nil || !strings.Contains(task, `"name":"feedUrl"`) || !strings.Contains(task, "feeds.synthetic.invalid") {
					t.Fatalf("schedule CLI task copy did not inject the bounded dynamic selector input: err=%v", err)
				}
			}
		})
	}
}

func TestTaskHTTPMockWorkflowCompilerRejectsMissingOrMismatchedOwnerRoutes(t *testing.T) {
	raw := readFixture(t, "public-workflows/6338.json")
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatal(err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-6338-profile.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := simulator.ExportHTTPMockProfile()
	if err != nil {
		t.Fatal(err)
	}
	addresses, err := HTTPMockOperationAddresses(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compileTaskHTTPMockWorkflow(raw, workflow, profile, addresses[:len(addresses)-1]); err == nil {
		t.Fatal("compiler accepted an incomplete task-local route receipt")
	}
	addresses[0].URL = strings.Replace(addresses[0].URL, "/", "/unmatched/", 1)
	if _, err := compileTaskHTTPMockWorkflow(raw, workflow, profile, addresses); err == nil {
		t.Fatal("compiler accepted an owner address that does not match its API fixture")
	}
}
