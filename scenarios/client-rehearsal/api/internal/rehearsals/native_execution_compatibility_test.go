package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type nativeExecutionMatrixCase struct {
	id                string
	expectedSource    string
	trigger           TriggerKind
	route             NativeTriggerRoute
	incompatibility   string
	profile           string
	profileKind       string
	input             map[string]any
	seededFixture     string
	repairProfile     string
	expectedSeededSHA string
	historicalOutcome string
}

func TestNativeExecutionCompatibilityMatrixPreservesTaskInputProvenanceAndHandoffs(t *testing.T) {
	cases := []nativeExecutionMatrixCase{
		{id: "6338", expectedSource: "6d960557d679630dd8ec561ac3ec552c0563990607678fb9027e83220cb08f7b", trigger: TriggerManual, route: NativeRouteManual, profile: "public-workflow-6338-profile.json", input: map[string]any{"caseId": "F7-6338", "contact": map[string]any{"name": "Synthetic contact"}}, seededFixture: "repairs/6338-seeded.json", repairProfile: "public-workflow-6338-profile.json", expectedSeededSHA: "9b13403cf55d1aa2988cb5c31bb631a7425941ea6e0c326352fa66ac12ea8e47"},
		{id: "16904", expectedSource: "dbef2a8dac9eab999cdddd0e5be413c2de5d7b454329b3803160fb906b55e704", trigger: TriggerWebhook, route: NativeRouteWebhook, profile: "public-workflow-16904-profile.json", input: map[string]any{"body": map[string]any{"name": "Synthetic Attendee", "email": "new-attendee@synthetic.invalid", "role": "Participant", "eventId": "synthetic-event-1", "day": "Day 1"}}, seededFixture: "repairs/16904-seeded.json", repairProfile: "public-workflow-16904-profile.json", expectedSeededSHA: "3421fe2a7f0c3ca4aba165b277c95218e67453b8b124e99223daea9cfe0eb549"},
		{id: "16364", expectedSource: "19217657f3142431844d90963efe9354ecac7d4eace685f4c7827ac5164cc295", trigger: TriggerScheduled, route: NativeRouteScheduled, profile: "public-workflow-16364-native-profile.json", profileKind: "virtual-api", input: map[string]any{"batchId": "F7-16364", "runDate": "2026-11-16"}, seededFixture: "repairs/16364-seeded.json", repairProfile: "public_workflow_16364_catalog.json", expectedSeededSHA: "3216c935bc503b4f8b23aeef152ae619a2f644a4b2ed45af8e3670e403f2d3c0"},
		{id: "17164", expectedSource: "2788a548183fb6a9cf96f9c16a590ab7da4e87ae7850fac7c0dd014e6194f715", trigger: TriggerScheduled, route: NativeRouteScheduled, profile: "public-workflow-17164-profile.json", input: map[string]any{"feedUrl": "https://feeds.synthetic.invalid/f7-approved.xml"}},
		{id: "17831", expectedSource: "8184b1c054300842c0dd144eaa137a237b9c017badb537ebc249614bb1eff56a", trigger: TriggerWebhook, route: NativeRouteWebhook, profile: "public-workflow-17831-profile.json", input: map[string]any{"event": map[string]any{"fixture": "F7-17831"}}, historicalOutcome: "retained failed owner dispatch: HTTP 404, zero exact route hits"},
		{id: "1804", expectedSource: "f07ea4384b83db37b3f999bdfce143c83e477ebd3defff2e8f0f02a8eb377ce8", trigger: TriggerPolling, route: NativeRouteUnsupported, incompatibility: "polling triggers require an active poll scheduler; the n8n CLI start route cannot produce a poll event"},
		{id: "16667", expectedSource: "f2b0b6c5c2b1ba9bac9d943ab7b5aeb9b67655b87733bcab2c29c6a35289605a", trigger: TriggerEvent, route: NativeRouteUnsupported, incompatibility: "instance event triggers require an n8n event-bus event that the task owner CLI does not expose"},
		{id: "6920", expectedSource: "5e095fad1bba7bb2868fda9a03c429cdd0aee97d7b347cdb94d3683b2dfb9e53", trigger: TriggerError, route: NativeRouteUnsupported, incompatibility: "Error Trigger requires an upstream failed execution context; it cannot be started as a standalone workflow"},
		{id: "5852", expectedSource: "841b9a2120ef937d578f8279e71aa844edae1b613808945a045a3fdd8204e24d", trigger: TriggerSubworkflow, route: NativeRouteUnsupported, incompatibility: "Execute Workflow Trigger requires a parent workflow execution context"},
		{id: "8011", expectedSource: "43f7814e7d9a0181002b335cffdbb17162f2935d1fa6b97269efe76436591ffb", trigger: TriggerChat, route: NativeRouteUnsupported, incompatibility: "Chat Trigger requires the n8n chat session runtime, which the task owner CLI does not expose"},
	}

	for _, test := range cases {
		t.Run(test.id, func(t *testing.T) {
			raw := readFixture(t, "public-workflows/"+test.id+".json")
			sourceSHA := sha256.Sum256([]byte(raw))
			if got := hex.EncodeToString(sourceSHA[:]); got != test.expectedSource {
				t.Fatalf("frozen sanitized public source provenance changed: got %s want %s", got, test.expectedSource)
			}
			compatibility := nativeTriggerCompatibility(test.trigger)
			if compatibility.Route != test.route || compatibility.Supported != (test.route != NativeRouteUnsupported) || compatibility.Incompatibility != test.incompatibility {
				t.Fatalf("native trigger compatibility drift: got %+v", compatibility)
			}
			workflow, _, _, err := normalizeWorkflow(raw)
			if err != nil {
				if test.route == NativeRouteUnsupported && test.trigger == TriggerChat && strings.Contains(err.Error(), "unsupported executable node") {
					t.Logf("typed source incompatibility retained for public chat workflow: %v", err)
					return
				}
				t.Fatalf("normalize public fixture: %v", err)
			}
			triggerNodeID := ""
			for _, node := range workflow.Nodes {
				if node.Trigger == test.trigger && triggerNodeID == "" {
					triggerNodeID = node.ID
				}
			}
			if triggerNodeID == "" {
				t.Fatalf("public fixture lost expected %s trigger: kinds=%v", test.trigger, workflow.TriggerKinds)
			}
			if test.route == NativeRouteUnsupported {
				return
			}
			simulator, err := NewDescriptionSimulator(readFixture(t, test.profile), false)
			if err != nil {
				t.Fatalf("load frozen owner API profile: %v", err)
			}
			taskCopy := ""
			var addresses []HTTPMockOperationAddress
			if test.profileKind == "virtual-api" {
				virtualProfile, profileErr := simulator.ExportVirtualAPIProfile()
				if profileErr != nil || !workflowUsesExactVirtualAPIRoutes(workflow, virtualProfile) {
					t.Fatalf("compile frozen API description into exact virtual routes: hosts=%v err=%v", virtualProfile.Hosts, profileErr)
				}
				taskCopy, err = rewriteWorkflowHTTPRequestsForVirtualAPI(raw, workflow, virtualProfile)
				if err != nil {
					t.Fatalf("compile provider nodes to exact virtual task routes: %v", err)
				}
			} else {
				profile, profileErr := simulator.ExportHTTPMockProfile()
				if profileErr != nil {
					t.Fatalf("compile frozen API description into exact routes: %v", profileErr)
				}
				addresses, err = HTTPMockOperationAddresses(profile)
				if err != nil {
					t.Fatalf("derive fixed task-local addresses: %v", err)
				}
				taskCopy, err = compileTaskHTTPMockWorkflow(raw, workflow, profile, addresses)
				if err != nil {
					t.Fatalf("compile task workflow to exact API-described routes: %v", err)
				}
			}
			switch test.route {
			case NativeRouteManual:
				taskCopy, err = injectSyntheticInputAfterEntry(taskCopy, test.input, TriggerManual)
			case NativeRouteScheduled:
				taskCopy, err = rewriteScheduledWorkflowForOneShotDispatch(taskCopy, workflow)
				if err == nil {
					taskCopy, err = injectSyntheticInputAfterEntry(taskCopy, test.input, TriggerManual)
				}
			case NativeRouteWebhook:
				_, _, err = normalizeTriggerInput(test.input)
			}
			if err != nil {
				t.Fatalf("prepare trigger-specific task-local input route: %v", err)
			}
			taskSHA := sha256.Sum256([]byte(taskCopy))
			if taskSHA == sourceSHA {
				t.Fatal("native task-local copy did not differ from the frozen public source")
			}
			if test.profileKind == "virtual-api" {
				if !strings.Contains(taskCopy, "sheets.googleapis.com") || !strings.Contains(taskCopy, "gmail.googleapis.com") {
					t.Fatal("VirtualAPI task copy omitted an exact provider host alias")
				}
			} else {
				containsOwnerRoute := false
				for _, address := range addresses {
					containsOwnerRoute = containsOwnerRoute || strings.Contains(taskCopy, address.URL)
				}
				if !containsOwnerRoute {
					t.Fatal("task copy omitted every owner-issued exact route")
				}
			}
			inputJSON, err := json.Marshal(test.input)
			if err != nil {
				t.Fatal(err)
			}
			inputSHA := sha256.Sum256(inputJSON)
			t.Logf("adapter conformance source=%s taskCopy=%s input=%s route=%s historical=%s", hex.EncodeToString(sourceSHA[:]), hex.EncodeToString(taskSHA[:]), hex.EncodeToString(inputSHA[:]), compatibility.Route, test.historicalOutcome)

			if test.seededFixture == "" {
				return
			}
			seeded := readFixture(t, test.seededFixture)
			seedSHA := sha256.Sum256([]byte(seeded))
			if got := hex.EncodeToString(seedSHA[:]); got != test.expectedSeededSHA {
				t.Fatalf("seeded defect provenance changed: got %s want %s", got, test.expectedSeededSHA)
			}
			repairProfile := readFixture(t, test.repairProfile)
			clock := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
			repairService := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return clock })
			input := test.input
			if test.trigger == TriggerManual {
				input = nil // The public reminder's canonical manual contract starts with an empty item.
			}
			failed, err := repairService.CreateAndRun(context.Background(), CreateInput{Title: "Native matrix seeded fault " + test.id, WorkflowJSON: seeded, TriggerNodeID: triggerNodeID, TriggerInput: input, SimulatorProfile: test.repairProfile, SimulatorSourceJSON: repairProfile})
			if err != nil || failed.State != StateFailed {
				t.Fatalf("seeded public defect should produce a failed rehearsal before handoff: state=%s err=%v", failed.State, err)
			}
			repaired, err := repairService.CreateAndRun(context.Background(), CreateInput{Title: "Native matrix repaired handoff " + test.id, WorkflowJSON: raw, TriggerNodeID: triggerNodeID, TriggerInput: input, SimulatorProfile: test.repairProfile, SimulatorSourceJSON: repairProfile, RepairOfID: failed.ID})
			if err != nil || repaired.State != StateNeedsReview || !strings.Contains(repaired.RepairSummary, failed.ID) || !strings.Contains(repaired.HandoffMarkdown, repaired.SourceSHA256) {
				t.Fatalf("corrected public revision did not produce a provenance-linked review handoff: state=%s err=%v repair=%q", repaired.State, err, repaired.RepairSummary)
			}
			t.Logf("repair handoff failedRun=%s seededSource=%s repairedSource=%s handoffSha256=%x", failed.ID, hex.EncodeToString(seedSHA[:]), repaired.SourceSHA256, sha256.Sum256([]byte(repaired.HandoffMarkdown)))
		})
	}
}

func TestNativeUnsupportedTriggerCompatibilityFailsBeforeSandboxSideEffects(t *testing.T) {
	for _, item := range []struct {
		id      string
		trigger TriggerKind
	}{
		{id: "1804", trigger: TriggerPolling},
		{id: "16667", trigger: TriggerEvent},
		{id: "6920", trigger: TriggerError},
		{id: "5852", trigger: TriggerSubworkflow},
		{id: "8011", trigger: TriggerChat},
	} {
		compatibility := nativeTriggerCompatibility(item.trigger)
		if compatibility.Supported || compatibility.Route != NativeRouteUnsupported || strings.TrimSpace(compatibility.Incompatibility) == "" {
			t.Errorf("trigger %q lacks an explicit native incompatibility contract: %+v", item.trigger, compatibility)
			continue
		}
		workflowJSON := readFixture(t, "public-workflows/"+item.id+".json")
		if item.trigger == TriggerChat {
			workflowJSON = `{"name":"synthetic chat compatibility probe","nodes":[{"id":"chat","name":"Chat","type":"@n8n/n8n-nodes-langchain.chatTrigger","parameters":{}},{"id":"end","name":"End","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Chat":{"main":[[{"node":"End"}]]}}}`
		}
		workflow, _, _, err := normalizeWorkflow(workflowJSON)
		if err != nil {
			t.Errorf("normalize unsupported trigger fixture #%s: %v", item.id, err)
			continue
		}
		triggerID := ""
		for _, node := range workflow.Nodes {
			if node.Trigger == item.trigger {
				triggerID = node.ID
				break
			}
		}
		if triggerID == "" {
			t.Errorf("public fixture #%s lost trigger family %s", item.id, item.trigger)
			continue
		}
		events := []string{}
		nativeService := NewServiceWithNativeN8N(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Unix(1, 0) }, nil, &taskN8NRuntimeFake{events: &events})
		_, err = nativeService.(*service).CreateAndRun(context.Background(), CreateInput{Title: "Unsupported native trigger test", WorkflowJSON: workflowJSON, TriggerNodeID: triggerID, taskSandbox: &WorkspaceSandboxTaskRequest{TaskID: "unsupported-" + item.id, Name: "unsupported-trigger"}})
		if err == nil || !strings.Contains(err.Error(), compatibility.Incompatibility) || len(events) != 0 {
			t.Errorf("unsupported %s trigger must fail with its typed reason before owner/sandbox calls: err=%v events=%v", item.trigger, err, events)
		}
	}
}
