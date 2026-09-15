package codecs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

// TestCodexParseTranscriptLineSurfacesGoalOnLivePath proves the live codec path
// (not only the import seam) recognizes a Codex goal event and sets Result.Goal,
// so a Codex goal run can reach a goal terminal under Agent Manager.
func TestCodexParseTranscriptLineSurfacesGoalOnLivePath(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "codex_goal_updated.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	c := NewCodexForTest()
	parser := c.NewTranscriptParser()
	result := parser.ParseTranscriptLine(uuid.New(), string(raw))
	if result.Goal == nil {
		t.Fatal("live ParseTranscriptLine did not surface the goal marker")
	}
	if result.Goal.Objective != "complete the governed plan" || result.Goal.Status != runner.GoalStatusBudgetLimited {
		t.Fatalf("goal = %+v", result.Goal)
	}
}

// A tool request records intent. Only the matched accepted result can report
// its effect, even when the parser already knows the installed objective.
func TestCodexParseTranscriptLineDoesNotCompleteFromUpdateGoalRequest(t *testing.T) {
	c := NewCodexForTest()
	parser := c.NewTranscriptParser()
	runID := uuid.New()
	// The goal is installed with an active thread_goal_updated carrying the
	// objective.
	parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"finish the plan","status":"active"}}}`)
	line := `{"type":"response_item","payload":{"type":"custom_tool_call","name":"exec","input":"const r = await tools.update_goal({status:\"complete\"})"}}`
	result := parser.ParseTranscriptLine(runID, line)
	if result.Goal != nil || result.Terminal != nil {
		t.Fatalf("update_goal request became accepted completion: %+v", result)
	}
}

// TestCodexParseTranscriptLineSurfacesCompleteGoal proves a completion status is
// surfaced with the complete vocabulary on the live path.
func TestCodexParseTranscriptLineSurfacesCompleteGoal(t *testing.T) {
	c := NewCodexForTest()
	parser := c.NewTranscriptParser()
	line := `{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"finish the plan","status":"complete"}}}`
	result := parser.ParseTranscriptLine(uuid.New(), line)
	if result.Goal == nil || result.Goal.Status != runner.GoalStatusComplete {
		t.Fatalf("goal = %+v, want complete", result.Goal)
	}
}

// Missing parser history must not turn intent into a fabricated terminal.
func TestCodexParseTranscriptLineDoesNotCompleteFromRequestWithoutInstall(t *testing.T) {
	c := NewCodexForTest()
	parser := c.NewTranscriptParser()
	line := `{"type":"response_item","payload":{"type":"custom_tool_call","name":"exec","input":"const r = await tools.update_goal({status:\"complete\"})"}}`
	result := parser.ParseTranscriptLine(uuid.New(), line)
	if result.Goal != nil || result.Terminal != nil {
		t.Fatalf("request became completion without accepted evidence: %+v", result)
	}
}

func TestCodexGoalAcceptedReceiptAcrossConsumeSegments(t *testing.T) {
	path := filepath.Join("testdata", "codex_goal_accepted.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("fixture line count = %d", len(lines))
	}
	boundary := int64(len(strings.Join(lines[:3], "\n")) + 1)
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	cursor, terminal, err := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, EndAt: boundary, ParseFn: parser.ParseTranscriptLine})
	if err != nil || terminal != nil || cursor != boundary {
		t.Fatalf("request segment became terminal: cursor=%d terminal=%+v err=%v", cursor, terminal, err)
	}
	var accepted *runner.GoalMarker
	_, terminal, err = runner.Consume(t.Context(), runner.ConsumeArgs{RunID: runID, Transcript: path, StartAt: cursor, ParseFn: parser.ParseTranscriptLine, OnGoalStatus: func(marker runner.GoalMarker) error { accepted = &marker; return nil }})
	if err != nil || terminal == nil || !terminal.Success || terminal.TerminalReason != "goal_complete" || accepted == nil || accepted.Objective != "complete the fixture" {
		t.Fatalf("matched native receipt across segments was lost: goal=%+v terminal=%+v err=%v", accepted, terminal, err)
	}
	// A rebuilt parser needs the request prefix; a result by itself is unknown.
	fresh := NewCodexForTest().NewTranscriptParser()
	if result := fresh.ParseTranscriptLine(runID, lines[3]); result.Goal != nil {
		t.Fatal("output-only segment fabricated request correlation")
	}
	for _, line := range lines[:3] {
		fresh.ParseTranscriptLine(runID, line)
	}
	if result := fresh.ParseTranscriptLine(runID, lines[3]); result.Goal == nil || result.Goal.Status != runner.GoalStatusComplete {
		t.Fatal("replayed request prefix did not restore goal correlation")
	}
}

func TestCodexGoalEvidenceRejectsRequestsQuotesAndUnmatchedResults(t *testing.T) {
	for name, line := range map[string]string{
		"request_only":      goalRequestLine("call-goal", `const r = await tools.update_goal({status:"complete"}); text(r);`),
		"assistant_quote":   `{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"tools.update_goal({status:'complete'})"}]}}`,
		"user_quote":        `{"type":"event_msg","payload":{"type":"user_message","message":"tools.update_goal({status:'complete'})"}}`,
		"shell_output":      `{"type":"response_item","payload":{"type":"function_call_output","call_id":"shell-call","output":"tools.update_goal({status:'complete'})"}}`,
		"unmatched_receipt": goalReceiptLine("unknown-call", acceptedGoalReceipt()),
		"non_native_alias":  `{"type":"event_msg","payload":{"type":"update_goal","goal":{"objective":"quoted request","status":"complete"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			if result := parser.ParseTranscriptLine(uuid.New(), line); result.Goal != nil || result.Terminal != nil {
				t.Fatalf("unaccepted evidence became terminal: %+v", result)
			}
		})
	}
}

func TestCodexGoalOnlyMatchingSuccessfulReceiptQualifies(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"accepted":          func(map[string]any) {},
		"rejected":          func(r map[string]any) { r["error"] = "goal update refused" },
		"is_error":          func(r map[string]any) { r["isError"] = true },
		"snake_case_error":  func(r map[string]any) { r["is_error"] = true },
		"unsuccessful":      func(r map[string]any) { r["success"] = false },
		"missing_goal":      func(r map[string]any) { delete(r, "goal") },
		"wrong_status":      func(r map[string]any) { r["goal"].(map[string]any)["status"] = "blocked" },
		"missing_objective": func(r map[string]any) { r["goal"].(map[string]any)["objective"] = "" },
		"wrong_thread":      func(r map[string]any) { r["goal"].(map[string]any)["threadId"] = "other-thread" },
	} {
		t.Run(name, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			runID := uuid.New()
			parser.ParseTranscriptLine(runID, `{"type":"session_meta","payload":{"id":"fixture-thread"}}`)
			request := goalRequestLine("call-goal", `const r = await tools.update_goal({status:"complete"}); text(r);`)
			if result := parser.ParseTranscriptLine(runID, request); result.Goal != nil {
				t.Fatal("request became completion before result")
			}
			receipt := acceptedGoalReceipt()
			mutate(receipt)
			if result := parser.ParseTranscriptLine(runID, goalReceiptLine("other-call", receipt)); result.Goal != nil {
				t.Fatal("unrelated call satisfied pending goal update")
			}
			result := parser.ParseTranscriptLine(runID, goalReceiptLine("call-goal", receipt))
			if (result.Goal != nil) != (name == "accepted") {
				t.Fatalf("receipt %s: goal=%+v", name, result.Goal)
			}
			if duplicate := parser.ParseTranscriptLine(runID, goalReceiptLine("call-goal", acceptedGoalReceipt())); duplicate.Goal != nil {
				t.Fatal("consumed/rejected call accepted a later duplicate result")
			}
		})
	}
}

func TestCodexGoalQuotedOrConditionalScriptCannotSupplyReceipt(t *testing.T) {
	for _, script := range []string{
		`text("tools.update_goal({status:'complete'})")`,
		`if (false) { await tools.update_goal({status:"complete"}); }`,
		`// tools.update_goal({status:"complete"})`,
		`const r = await tools.update_goal({status:"complete"}); text({goal:{status:"complete"}});`,
	} {
		parser := NewCodexForTest().NewTranscriptParser()
		runID := uuid.New()
		for _, line := range []string{goalRequestLine("call-goal", script), goalReceiptLine("call-goal", acceptedGoalReceipt())} {
			if result := parser.ParseTranscriptLine(runID, line); result.Goal != nil {
				t.Fatalf("unsupported script became accepted goal evidence: %q", script)
			}
		}
	}
}

func TestCodexGoalCorrelationIsScopedAndInvalidatedByNewGoal(t *testing.T) {
	for _, change := range []string{"other_run", "other_thread", "new_goal", "changed_request", "wrong_output_type", "missing_call_id", "rejected_envelope"} {
		t.Run(change, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			runID := uuid.New()
			parser.ParseTranscriptLine(runID, `{"type":"session_meta","payload":{"id":"fixture-thread"}}`)
			parser.ParseTranscriptLine(runID, goalRequestLine("call-goal", `text(await tools.update_goal({status:"complete"}));`))
			output := goalReceiptLine("call-goal", acceptedGoalReceipt())
			switch change {
			case "other_run":
				runID = uuid.New()
			case "other_thread":
				parser.ParseTranscriptLine(runID, `{"type":"session_meta","payload":{"id":"other-thread"}}`)
			case "new_goal":
				parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"new work","status":"active"}}}`)
			case "changed_request":
				parser.ParseTranscriptLine(runID, goalRequestLine("call-goal", `text(await tools.update_goal({status:"blocked"}));`))
			case "wrong_output_type":
				output = strings.Replace(output, "custom_tool_call_output", "function_call_output", 1)
			case "missing_call_id":
				output = goalReceiptLine("", acceptedGoalReceipt())
			case "rejected_envelope":
				output = strings.Replace(output, `"payload":{`, `"payload":{"isError":true,`, 1)
			}
			if result := parser.ParseTranscriptLine(runID, output); result.Goal != nil {
				t.Fatalf("%s reused stale/unqualified correlation: %+v", change, result.Goal)
			}
		})
	}
}

func TestCodexGoalTypedCallAndReceiptCarryAcceptedStatus(t *testing.T) {
	for _, status := range []runner.GoalStatus{runner.GoalStatusComplete, runner.GoalStatusBlocked, runner.GoalStatusBudgetLimited, runner.GoalStatusUsageLimited} {
		t.Run(string(status), func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			runID := uuid.New()
			request := `{"type":"response_item","payload":{"type":"function_call","name":"update_goal","call_id":"call-goal","arguments":"{\"status\":\"` + string(status) + `\"}"}}`
			if result := parser.ParseTranscriptLine(runID, request); result.Goal != nil {
				t.Fatal("typed request emitted completion")
			}
			receipt := acceptedGoalReceipt()
			receipt["goal"].(map[string]any)["status"] = status
			text, _ := json.Marshal(receipt)
			output, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "call-goal", "output": string(text)}})
			result := parser.ParseTranscriptLine(runID, string(output))
			if result.Goal == nil || result.Goal.Status != status || result.Goal.Objective != "complete the fixture" {
				t.Fatalf("accepted typed receipt lost status/objective: %+v", result.Goal)
			}
		})
	}
}

func TestCodexGoalMalformedOrAmbiguousReceiptStaysUnknown(t *testing.T) {
	for name, output := range map[string]any{
		"missing":        nil,
		"plain_error":    "goal update rejected",
		"quoted_receipt": `"{\"goal\":{\"objective\":\"complete the fixture\",\"status\":\"complete\"}}"`,
		"malformed":      `{"goal":`,
		"two_receipts":   []map[string]string{{"type": "input_text", "text": `{"goal":{"threadId":"fixture-thread","objective":"complete the fixture","status":"complete"}}`}, {"type": "input_text", "text": `{"goal":{"threadId":"fixture-thread","objective":"complete the fixture","status":"complete"}}`}},
	} {
		t.Run(name, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			runID := uuid.New()
			parser.ParseTranscriptLine(runID, goalRequestLine("call-goal", `text(await tools.update_goal({status:"complete"}));`))
			data, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "call-goal", "output": output}})
			if result := parser.ParseTranscriptLine(runID, string(data)); result.Goal != nil {
				t.Fatalf("%s receipt became completion: %+v", name, result.Goal)
			}
		})
	}
}

func acceptedGoalReceipt() map[string]any {
	return map[string]any{"goal": map[string]any{"threadId": "fixture-thread", "objective": "complete the fixture", "status": "complete"}}
}

func goalRequestLine(callID, script string) string {
	data, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": callID, "input": script}})
	return string(data)
}

func goalReceiptLine(callID string, receipt map[string]any) string {
	text, _ := json.Marshal(receipt)
	data, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": callID, "output": []map[string]string{{"type": "input_text", "text": string(text)}}}})
	return string(data)
}

func TestCodexDefaultPrivacyStillInvalidatesGoalAcrossUserTurn(t *testing.T) {
	for _, userLine := range []string{
		`{"type":"event_msg","payload":{"type":"user_message","message":"private next task"}}`,
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"private next task"}]}}`,
	} {
		parser := NewCodexForTest().NewTranscriptParser()
		id := uuid.New()
		parsed := parser.ParseTranscriptLine(id, userLine)
		if !parsed.UserTurnStarted || len(parsed.Events) != 0 {
			t.Fatalf("privacy or boundary lost: %+v", parsed)
		}
		path := filepath.Join(t.TempDir(), "native.jsonl")
		body := "{\"type\":\"event_msg\",\"payload\":{\"type\":\"thread_goal_updated\",\"goal\":{\"objective\":\"finish\",\"status\":\"complete\"}}}\n" + userLine + "\n"
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		fresh := NewCodexForTest().NewTranscriptParser()
		_, terminal, err := runner.Consume(t.Context(), runner.ConsumeArgs{RunID: id, Transcript: path, ParseFn: fresh.ParseTranscriptLine})
		if err != nil || terminal != nil {
			t.Fatalf("old completion survived private next turn: terminal=%+v err=%v", terminal, err)
		}
	}
}
