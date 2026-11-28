package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/agenttranscript"
)

// TestReviewHandoffNativeResumeReceiptLocalIntegration follows the production
// seams without starting a harness: native source lines enter through the
// shared dialect parser and ConversationStore, stable selected IDs are
// resolved, and the existing review payload is submitted to a fake local
// adapter carrying shared native-resume metadata. The fake persists native
// target lines and projects parsed events into a second store. It is test
// evidence only; it does not claim a live target receipt.
func TestReviewHandoffNativeResumeReceiptLocalIntegration(t *testing.T) {
	ctx := context.Background()
	harnesses := []string{agenttranscript.HarnessClaude, agenttranscript.HarnessCodex, agenttranscript.HarnessGrok, agenttranscript.HarnessOpenCode}
	const clue = "opportunity-code: amber-otter-731"
	for _, sourceHarness := range harnesses {
		for _, targetHarness := range harnesses {
			t.Run(sourceHarness+"-to-"+targetHarness, func(t *testing.T) {
				sourceStore := NewConversationStore()
				sourceSession := "source-" + sourceHarness
				var selectedIDs []string
				for i, line := range localSourceLines(sourceHarness, clue) {
					messages := agenttranscript.ParseMessages(sourceHarness, line)
					if len(messages) != 1 {
						t.Fatalf("source %s record %d parsed to %#v", sourceHarness, i, messages)
					}
					message := messages[0]
					event, result := sourceStore.AppendNativeEvent(ctx, sourceSession, sourceHarness+"_tailer", message.Role, message.Text, &NativeProvenance{
						Provider: sourceHarness, SessionID: "native-source", TurnID: "source-turn", MessageID: fmt.Sprintf("source-%d", i),
					})
					if !result.Appended {
						t.Fatalf("persist source event %d: %+v", i, result)
					}
					selectedIDs = append(selectedIDs, event.ID)
				}
				selected, missing, err := sourceStore.ResolveEvents(ctx, sourceSession, selectedIDs)
				if err != nil || len(missing) != 0 || len(selected) != len(selectedIDs) {
					t.Fatalf("resolve reviewed selection: selected=%d missing=%v err=%v", len(selected), missing, err)
				}
				var payloadParts []string
				for _, event := range selected {
					payloadParts = append(payloadParts, string(event.Role)+": "+event.Text)
				}
				selectedPayload := strings.Join(payloadParts, "\n")
				if !strings.Contains(selectedPayload, clue) || !strings.Contains(selectedPayload, "Tool call") {
					t.Fatalf("reviewed selection lost text or tool call: %q", selectedPayload)
				}

				format, ok := agenttranscript.FormatFor(targetHarness)
				if !ok || (format.ResumeFlag == "" && format.ResumeCommand == "") {
					t.Fatalf("%s has no shared native-resume descriptor: %+v", targetHarness, format)
				}
				marker := "WC2-LOCAL-" + sourceHarness + "-" + targetHarness
				targetSession := "fake-local-target-" + sourceHarness + "-" + targetHarness
				request := localNativeResumeRequest{
					Harness: targetHarness, SessionID: targetSession,
					ResumeFlag: format.ResumeFlag, ResumeCommand: format.ResumeCommand,
					IdempotencyKey: "handoff/" + marker, Marker: marker,
					Payload: "Use this selected context to answer the question. Include the receipt marker in your next answer: " + marker + "\n" + selectedPayload,
				}
				adapter := newLocalNativeResumeAdapter(NewConversationStore())
				adapter.failOnceAfterDelivery = true
				if err := adapter.Submit(ctx, request); !errors.Is(err, errLocalPartialWrite) {
					t.Fatalf("first submit = %v, want injected partial-write error", err)
				}
				partial := agenttranscript.InspectHandoffTranscript(targetHarness, marker, adapter.transcripts[targetSession])
				if !partial.Delivered || partial.Consumed {
					t.Fatalf("partial write receipt = %+v, want delivered only", partial)
				}
				partialStore := adapter.store.ListSession(ctx, targetSession)
				if countConversationRole(partialStore.Events, ConversationRoleUser) != 1 || countConversationRole(partialStore.Events, ConversationRoleAssistant) != 0 {
					t.Fatalf("partial target projection = %#v", partialStore.Events)
				}

				if err := adapter.Submit(ctx, request); err != nil {
					t.Fatalf("retry after persisted user marker: %v", err)
				}
				afterRetry := agenttranscript.InspectHandoffTranscript(targetHarness, marker, adapter.transcripts[targetSession])
				if !afterRetry.Delivered || !afterRetry.Consumed {
					t.Fatalf("receipt after later native assistant marker = %+v", afterRetry)
				}
				state := adapter.store.ListSession(ctx, targetSession)
				if countConversationRole(state.Events, ConversationRoleUser) != 1 || countConversationRole(state.Events, ConversationRoleAssistant) != 1 {
					t.Fatalf("retry duplicated native target messages: %#v", state.Events)
				}
				if !strings.Contains(state.Events[1].Text, "amber-otter-731") || !strings.Contains(state.Events[1].Text, marker) {
					t.Fatalf("local target did not answer from selected context and echo marker: %#v", state.Events[1])
				}

				lineCount := len(adapter.transcripts[targetSession])
				if err := adapter.Submit(ctx, request); err != nil {
					t.Fatalf("idempotent retry: %v", err)
				}
				if got := len(adapter.transcripts[targetSession]); got != lineCount {
					t.Fatalf("completed retry wrote %d more transcript lines", got-lineCount)
				}
				conflicting := request
				conflicting.Payload += " changed"
				if err := adapter.Submit(ctx, conflicting); err == nil {
					t.Fatal("reusing an idempotency key with a changed payload unexpectedly succeeded")
				}
				if got := len(adapter.transcripts[targetSession]); got != lineCount {
					t.Fatalf("conflicting retry wrote %d transcript lines", got-lineCount)
				}
			})
		}
	}
}

type localNativeResumeRequest struct {
	Harness        string
	SessionID      string
	ResumeFlag     string
	ResumeCommand  string
	IdempotencyKey string
	Marker         string
	Payload        string
}

var errLocalPartialWrite = errors.New("injected local target failure after delivery was persisted")

// localNativeResumeAdapter mirrors the production boundary: the request
// names the native resume contract, while all writes remain in-memory and the
// target transcript is parsed using the shared production codec.
type localNativeResumeAdapter struct {
	store                  *ConversationStore
	transcripts            map[string][][]byte
	requests               map[string]string
	progressWritten        map[string]bool
	failOnceAfterDelivery  bool
	failureAlreadyInjected bool
}

func newLocalNativeResumeAdapter(store *ConversationStore) *localNativeResumeAdapter {
	return &localNativeResumeAdapter{
		store: store, transcripts: make(map[string][][]byte), requests: make(map[string]string), progressWritten: make(map[string]bool),
	}
}

func (a *localNativeResumeAdapter) Submit(ctx context.Context, request localNativeResumeRequest) error {
	if request.Harness == "" || request.SessionID == "" || request.IdempotencyKey == "" || request.Marker == "" {
		return errors.New("incomplete local native-resume request")
	}
	if request.ResumeFlag == "" && request.ResumeCommand == "" {
		return errors.New("request has no native resume invocation")
	}
	if prior, ok := a.requests[request.IdempotencyKey]; ok && prior != request.Payload {
		return errors.New("idempotency key reused with different payload")
	}
	a.requests[request.IdempotencyKey] = request.Payload

	if !a.progressWritten[request.SessionID] {
		a.transcripts[request.SessionID] = append(a.transcripts[request.SessionID], []byte(`{"type":"progress","message":"`+request.Marker+`"}`))
		a.progressWritten[request.SessionID] = true
		if got := agenttranscript.InspectHandoffTranscript(request.Harness, request.Marker, a.transcripts[request.SessionID]); got.Delivered || got.Consumed {
			return fmt.Errorf("non-conversational progress record produced receipt: %+v", got)
		}
	}

	if !a.hasMarkerRole(request, "user") {
		if err := a.persistLine(ctx, request, localNativeTextLine(request.Harness, "user", request.Payload)); err != nil {
			return err
		}
	}
	if a.failOnceAfterDelivery && !a.failureAlreadyInjected {
		a.failureAlreadyInjected = true
		return errLocalPartialWrite
	}
	if !a.hasMarkerRole(request, "assistant") {
		clue := extractOpportunityCode(request.Payload)
		answer := "The opportunity code is unavailable from my context."
		if clue != "" {
			answer = "The opportunity code is " + clue + "."
		}
		answer += " Receipt: " + request.Marker
		if err := a.persistLine(ctx, request, localNativeTextLine(request.Harness, "assistant", answer)); err != nil {
			return err
		}
	}
	return nil
}

func (a *localNativeResumeAdapter) persistLine(ctx context.Context, request localNativeResumeRequest, line []byte) error {
	a.transcripts[request.SessionID] = append(a.transcripts[request.SessionID], line)
	for i, message := range agenttranscript.ParseMessages(request.Harness, line) {
		_, result := a.store.AppendNativeEvent(ctx, request.SessionID, request.Harness+"_tailer", message.Role, message.Text, &NativeProvenance{
			Provider: request.Harness, SessionID: request.SessionID, TurnID: "target-turn", MessageID: fmt.Sprintf("native-%d-%s", len(a.transcripts[request.SessionID]), message.Role), BoundaryID: fmt.Sprint(i),
		})
		if !result.Appended {
			return fmt.Errorf("persist target native event: %+v", result)
		}
	}
	return nil
}

func (a *localNativeResumeAdapter) hasMarkerRole(request localNativeResumeRequest, role string) bool {
	for _, message := range a.store.ListSession(context.Background(), request.SessionID).Events {
		if string(message.Role) == role && strings.Contains(message.Text, request.Marker) {
			return true
		}
	}
	return false
}

func countConversationRole(events []ConversationEvent, role ConversationRole) int {
	count := 0
	for _, event := range events {
		if event.Role == role {
			count++
		}
	}
	return count
}

func extractOpportunityCode(payload string) string {
	const prefix = "opportunity-code: "
	start := strings.Index(payload, prefix)
	if start < 0 {
		return ""
	}
	value := payload[start+len(prefix):]
	if end := strings.IndexAny(value, " \n\r\t]"); end >= 0 {
		value = value[:end]
	}
	return value
}

func localSourceLines(harness, clue string) [][]byte {
	user := "Selected context: " + clue
	switch harness {
	case agenttranscript.HarnessClaude:
		return [][]byte{
			[]byte(fmt.Sprintf(`{"type":"user","message":{"content":[{"type":"text","text":%q}]}}`, user)),
			[]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"source-tool","name":"Read","input":{"path":"` + clue + `"}}]}}`),
		}
	case agenttranscript.HarnessCodex:
		return [][]byte{
			[]byte(fmt.Sprintf(`{"type":"event_msg","payload":{"type":"user_message","text":%q}}`, user)),
			[]byte(`{"type":"response_item","payload":{"type":"function_call","call_id":"source-tool","name":"read","arguments":"{\"path\":\"` + clue + `\"}"}}`),
		}
	case agenttranscript.HarnessGrok:
		return [][]byte{
			[]byte(fmt.Sprintf(`{"method":"session/update","params":{"sessionId":"source","update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":%q}}}}`, user)),
			[]byte(`{"method":"session/update","params":{"sessionId":"source","update":{"sessionUpdate":"tool_call","toolCallId":"source-tool","title":"Read","rawInput":{"path":"` + clue + `"}}}}`),
		}
	case agenttranscript.HarnessOpenCode:
		return [][]byte{
			[]byte(fmt.Sprintf(`{"info":{"role":"user"},"parts":[{"type":"text","text":%q}]}`, user)),
			[]byte(`{"info":{"role":"assistant"},"parts":[{"type":"tool","callID":"source-tool","tool":"read","input":{"path":"` + clue + `"}}]}`),
		}
	default:
		return nil
	}
}

func localNativeTextLine(harness, role, text string) []byte {
	switch harness {
	case agenttranscript.HarnessClaude:
		return []byte(fmt.Sprintf(`{"type":%q,"message":{"content":[{"type":"text","text":%q}]}}`, role, text))
	case agenttranscript.HarnessCodex:
		if role == "user" {
			return []byte(fmt.Sprintf(`{"type":"event_msg","payload":{"type":"user_message","text":%q}}`, text))
		}
		return []byte(fmt.Sprintf(`{"type":"response_item","payload":{"type":"agent_message","text":%q}}`, text))
	case agenttranscript.HarnessGrok:
		kind := "agent_message_chunk"
		if role == "user" {
			kind = "user_message_chunk"
		}
		return []byte(fmt.Sprintf(`{"method":"session/update","params":{"sessionId":"local-target","update":{"sessionUpdate":%q,"content":{"type":"text","text":%q}}}}`, kind, text))
	case agenttranscript.HarnessOpenCode:
		return []byte(fmt.Sprintf(`{"info":{"role":%q},"parts":[{"type":"text","text":%q}]}`, role, text))
	default:
		return nil
	}
}
