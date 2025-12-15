//go:build legacydb
// +build legacydb

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// Helper to create a RecordedAction with common fields
func createTestAction(actionType string, selector string, timestamp time.Time, url string) RecordedAction {
	action := RecordedAction{
		ID:         "test-id",
		SessionID:  "test-session",
		ActionType: actionType,
		Timestamp:  timestamp.Format(time.RFC3339Nano),
		URL:        url,
		Confidence: 0.9,
	}

	if selector != "" {
		action.Selector = &SelectorSet{
			Primary: selector,
			Candidates: []SelectorCandidate{
				{Type: "css", Value: selector, Confidence: 0.9},
			},
		}
	}

	return action
}

func TestAnalyzeTransitionForWait_ClickTriggersWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] click action triggers wait for next selector", func(t *testing.T) {
		now := time.Now()

		current := createTestAction("click", "#submit-btn", now, "https://example.com")
		next := createTestAction("type", "#username", now.Add(100*time.Millisecond), "https://example.com")

		result := analyzeTransitionForWait(current, next)

		require.NotNil(t, result, "Should insert wait after click before type")
		assert.Equal(t, "selector", result.WaitType)
		assert.Equal(t, "#username", result.Selector)
		assert.Equal(t, 10000, result.TimeoutMs)
	})
}

func TestAnalyzeTransitionForWait_NavigateTriggersWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] navigate action triggers wait for next selector", func(t *testing.T) {
		now := time.Now()

		current := createTestAction("navigate", "", now, "https://example.com")
		next := createTestAction("click", "#login-btn", now.Add(200*time.Millisecond), "https://example.com/login")

		result := analyzeTransitionForWait(current, next)

		require.NotNil(t, result, "Should insert wait after navigate")
		assert.Equal(t, "selector", result.WaitType)
		assert.Equal(t, "#login-btn", result.Selector)
	})
}

func TestAnalyzeTransitionForWait_URLChangeTriggersWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] URL change triggers wait for next selector", func(t *testing.T) {
		now := time.Now()

		current := createTestAction("click", "#link", now, "https://example.com/page1")
		next := createTestAction("click", "#button", now.Add(100*time.Millisecond), "https://example.com/page2")

		result := analyzeTransitionForWait(current, next)

		require.NotNil(t, result, "Should insert wait when URL changes")
		assert.Equal(t, "selector", result.WaitType)
		assert.Equal(t, "#button", result.Selector)
	})
}

func TestAnalyzeTransitionForWait_LargeTimeGapTriggersWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] >500ms time gap triggers wait for next selector", func(t *testing.T) {
		now := time.Now()

		// Scroll doesn't normally trigger changes, but large time gap should
		current := createTestAction("scroll", "", now, "https://example.com")
		current.Payload = map[string]interface{}{"scrollY": 500.0}

		next := createTestAction("click", "#lazy-loaded-btn", now.Add(800*time.Millisecond), "https://example.com")

		result := analyzeTransitionForWait(current, next)

		require.NotNil(t, result, "Should insert wait for large time gap")
		assert.Equal(t, "selector", result.WaitType)
		assert.Equal(t, "#lazy-loaded-btn", result.Selector)
	})
}

func TestAnalyzeTransitionForWait_VeryLargeGapInsertsTimeout(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] >2s time gap inserts timeout wait even without selector", func(t *testing.T) {
		now := time.Now()

		current := createTestAction("click", "#btn", now, "https://example.com")
		// Next action is navigate without selector
		next := createTestAction("navigate", "", now.Add(3*time.Second), "https://example.com/page2")

		result := analyzeTransitionForWait(current, next)

		require.NotNil(t, result, "Should insert timeout wait for large gap")
		assert.Equal(t, "timeout", result.WaitType)
		// Should be half the observed gap, capped at 5000
		assert.Equal(t, 1500, result.TimeoutMs)
	})
}

func TestAnalyzeTransitionForWait_NoWaitNeededForSameElement(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] no wait needed for non-triggering actions on close timeline", func(t *testing.T) {
		now := time.Now()

		// Two types on same page, close together, hover doesn't trigger changes
		current := createTestAction("hover", "#menu", now, "https://example.com")
		next := createTestAction("hover", "#submenu", now.Add(50*time.Millisecond), "https://example.com")

		result := analyzeTransitionForWait(current, next)

		assert.Nil(t, result, "Should not insert wait for close hover actions")
	})
}

func TestAnalyzeTransitionForWait_NoWaitForNavigateWithoutSelector(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] no selector wait for actions without selector", func(t *testing.T) {
		now := time.Now()

		current := createTestAction("click", "#btn", now, "https://example.com")
		// Navigate has no selector
		next := createTestAction("navigate", "", now.Add(100*time.Millisecond), "https://example.com/page2")

		result := analyzeTransitionForWait(current, next)

		// No selector wait because next action has no selector
		// And time gap is < 2 seconds so no timeout wait either
		assert.Nil(t, result, "Should not insert wait when next action has no selector")
	})
}

func TestMergeConsecutiveActions_MergesTyping(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] consecutive type actions on same element are merged", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("type", "#input", now, "https://example.com"),
			createTestAction("type", "#input", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("type", "#input", now.Add(200*time.Millisecond), "https://example.com"),
		}

		// Set text payloads
		actions[0].Payload = map[string]interface{}{"text": "hello"}
		actions[1].Payload = map[string]interface{}{"text": " "}
		actions[2].Payload = map[string]interface{}{"text": "world"}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 1, "Should merge into single action")
		assert.Equal(t, "type", result[0].ActionType)
		assert.Equal(t, "hello world", result[0].Payload["text"])
	})
}

func TestMergeConsecutiveActions_MergesScrolls(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] consecutive scroll actions are merged", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("scroll", "", now, "https://example.com"),
			createTestAction("scroll", "", now.Add(50*time.Millisecond), "https://example.com"),
			createTestAction("scroll", "", now.Add(100*time.Millisecond), "https://example.com"),
		}

		actions[0].Payload = map[string]interface{}{"scrollY": 100.0}
		actions[1].Payload = map[string]interface{}{"scrollY": 300.0}
		actions[2].Payload = map[string]interface{}{"scrollY": 500.0}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 1, "Should merge into single scroll")
		assert.Equal(t, 500.0, result[0].Payload["scrollY"], "Should use final scroll position")
	})
}

func TestMergeConsecutiveActions_RemovesFocusBeforeType(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] focus events before type on same element are removed", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("focus", "#input", now, "https://example.com"),
			createTestAction("type", "#input", now.Add(50*time.Millisecond), "https://example.com"),
		}
		actions[1].Payload = map[string]interface{}{"text": "test"}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 1, "Should remove focus before type")
		assert.Equal(t, "type", result[0].ActionType)
	})
}

func TestMergeConsecutiveActions_PreservesDifferentElements(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] type actions on different elements are not merged", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("type", "#username", now, "https://example.com"),
			createTestAction("type", "#password", now.Add(500*time.Millisecond), "https://example.com"),
		}

		actions[0].Payload = map[string]interface{}{"text": "user@test.com"}
		actions[1].Payload = map[string]interface{}{"text": "secret123"}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 2, "Should preserve separate type actions")
		assert.Equal(t, "#username", result[0].Selector.Primary)
		assert.Equal(t, "#password", result[1].Selector.Primary)
	})
}

func TestInsertSmartWaits_InsertsWaitNodes(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] inserts wait nodes between actions that need them", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com"),
			createTestAction("click", "#login-btn", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("type", "#username", now.Add(200*time.Millisecond), "https://example.com"),
		}
		actions[2].Payload = map[string]interface{}{"text": "testuser"}

		nodes, edges := insertSmartWaits(actions)

		// Should have: navigate, wait, click, wait, type = 5 nodes
		// (navigate triggers wait for click, click triggers wait for type)
		require.GreaterOrEqual(t, len(nodes), 4, "Should insert at least one wait node")

		// Verify wait nodes are present
		waitCount := 0
		for _, node := range nodes {
			if node["type"] == "wait" {
				waitCount++
			}
		}
		assert.GreaterOrEqual(t, waitCount, 1, "Should have at least one wait node")

		// Verify edges chain correctly
		assert.Equal(t, len(nodes)-1, len(edges), "Edges should chain all nodes")
	})
}

func TestInsertSmartWaits_HandlesEmptyActions(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] handles empty action list gracefully", func(t *testing.T) {
		nodes, edges := insertSmartWaits([]RecordedAction{})

		assert.Nil(t, nodes)
		assert.Nil(t, edges)
	})
}

func TestInsertSmartWaits_SingleAction(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] single action produces single node", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("click", "#btn", now, "https://example.com"),
		}

		nodes, edges := insertSmartWaits(actions)

		require.Len(t, nodes, 1, "Should have exactly one node")
		assert.Len(t, edges, 0, "Should have no edges for single node")
		assert.Equal(t, "click", nodes[0]["type"])
	})
}

func TestConvertActionsToWorkflow_Integration(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] full workflow conversion with smart waits", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com/login"),
			createTestAction("click", "#username-field", now.Add(100*time.Millisecond), "https://example.com/login"),
			createTestAction("type", "#username-field", now.Add(150*time.Millisecond), "https://example.com/login"),
			createTestAction("type", "#username-field", now.Add(200*time.Millisecond), "https://example.com/login"),
			createTestAction("click", "#submit-btn", now.Add(300*time.Millisecond), "https://example.com/login"),
		}

		// Set text payloads for type actions
		actions[2].Payload = map[string]interface{}{"text": "test"}
		actions[3].Payload = map[string]interface{}{"text": "user"}

		result := convertActionsToWorkflow(actions)

		require.NotNil(t, result["nodes"])
		require.NotNil(t, result["edges"])

		nodes := result["nodes"].([]map[string]interface{})
		edges := result["edges"].([]map[string]interface{})

		// After merging: navigate, click, type (merged), click = 4 actions
		// Plus wait nodes inserted between triggering actions
		assert.GreaterOrEqual(t, len(nodes), 4, "Should have at least 4 nodes")
		assert.Equal(t, len(nodes)-1, len(edges), "Edges should connect all nodes")

		// Verify at least one wait node exists
		hasWait := false
		for _, node := range nodes {
			if node["type"] == "wait" {
				hasWait = true
				// Verify wait node structure
				data := node["data"].(map[string]interface{})
				assert.NotNil(t, data["label"])
				assert.NotNil(t, data["timeoutMs"])
			}
		}
		assert.True(t, hasWait, "Should have at least one wait node")
	})
}

func TestDescribeElement_WithInnerText(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] describes element using inner text", func(t *testing.T) {
		action := RecordedAction{
			ElementMeta: &ElementMeta{
				InnerText: "Sign In",
				TagName:   "button",
			},
		}

		result := describeElement(action)

		assert.Equal(t, `"Sign In"`, result)
	})
}

func TestDescribeElement_WithAriaLabel(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] falls back to aria-label", func(t *testing.T) {
		action := RecordedAction{
			ElementMeta: &ElementMeta{
				AriaLabel: "Close dialog",
				TagName:   "button",
			},
		}

		result := describeElement(action)

		assert.Equal(t, "Close dialog", result)
	})
}

func TestDescribeElement_WithTagName(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] falls back to tag name", func(t *testing.T) {
		action := RecordedAction{
			ElementMeta: &ElementMeta{
				TagName: "input",
			},
		}

		result := describeElement(action)

		assert.Equal(t, "input", result)
	})
}

func TestDescribeElement_NoMeta(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] returns 'element' when no metadata", func(t *testing.T) {
		action := RecordedAction{}

		result := describeElement(action)

		assert.Equal(t, "element", result)
	})
}

func TestCreateWaitNode_SelectorWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] creates selector-based wait node", func(t *testing.T) {
		template := &WaitTemplate{
			WaitType:  "selector",
			Selector:  "#my-element",
			TimeoutMs: 15000,
			Label:     "Wait for element",
		}

		node := createWaitNode(template, "wait_1", 200.0)

		assert.Equal(t, "wait_1", node["id"])
		assert.Equal(t, "wait", node["type"])

		position := node["position"].(map[string]interface{})
		assert.Equal(t, 250.0, position["x"])
		assert.Equal(t, 200.0, position["y"])

		data := node["data"].(map[string]interface{})
		assert.Equal(t, "Wait for element", data["label"])
		assert.Equal(t, 15000, data["timeoutMs"])
		assert.Equal(t, "#my-element", data["selector"])
	})
}

func TestCreateWaitNode_TimeoutWait(t *testing.T) {
	t.Run("[REQ:BAS-SMART-WAITS] creates timeout-based wait node", func(t *testing.T) {
		template := &WaitTemplate{
			WaitType:  "timeout",
			TimeoutMs: 2000,
			Label:     "Wait for page to stabilize",
		}

		node := createWaitNode(template, "wait_2", 300.0)

		assert.Equal(t, "wait", node["type"])

		data := node["data"].(map[string]interface{})
		assert.Equal(t, "Wait for page to stabilize", data["label"])
		assert.Equal(t, 2000, data["timeoutMs"])
		assert.Nil(t, data["selector"], "Timeout wait should not have selector")
	})
}

// =============================================================================
// Additional Edge Case Tests for mergeConsecutiveActions
// =============================================================================

func TestMergeConsecutiveActions_EmptyInput(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] handles empty action list", func(t *testing.T) {
		result := mergeConsecutiveActions([]RecordedAction{})

		require.Len(t, result, 0, "Should return empty slice for empty input")
	})
}

func TestMergeConsecutiveActions_SingleAction(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] handles single action", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("click", "#btn", now, "https://example.com"),
		}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 1, "Should return single action unchanged")
		assert.Equal(t, "click", result[0].ActionType)
	})
}

func TestMergeConsecutiveActions_FocusNotFollowedByType(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] preserves focus when not followed by type", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("focus", "#input", now, "https://example.com"),
			createTestAction("click", "#submit", now.Add(100*time.Millisecond), "https://example.com"),
		}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 2, "Should preserve both actions")
		assert.Equal(t, "focus", result[0].ActionType)
		assert.Equal(t, "click", result[1].ActionType)
	})
}

func TestMergeConsecutiveActions_FocusOnDifferentElement(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] preserves focus when type is on different element", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("focus", "#input1", now, "https://example.com"),
			createTestAction("type", "#input2", now.Add(100*time.Millisecond), "https://example.com"),
		}
		actions[1].Payload = map[string]interface{}{"text": "hello"}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 2, "Should preserve focus when type is on different element")
		assert.Equal(t, "focus", result[0].ActionType)
		assert.Equal(t, "type", result[1].ActionType)
	})
}

func TestMergeConsecutiveActions_MixedSequence(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] handles mixed action sequence correctly", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com"),
			createTestAction("click", "#login", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("focus", "#username", now.Add(200*time.Millisecond), "https://example.com"),
			createTestAction("type", "#username", now.Add(300*time.Millisecond), "https://example.com"),
			createTestAction("type", "#username", now.Add(400*time.Millisecond), "https://example.com"),
			createTestAction("click", "#password", now.Add(500*time.Millisecond), "https://example.com"),
		}
		actions[3].Payload = map[string]interface{}{"text": "user"}
		actions[4].Payload = map[string]interface{}{"text": "name"}

		result := mergeConsecutiveActions(actions)

		// Expected: navigate, click, type (merged, focus removed), click
		require.Len(t, result, 4, "Should merge types and remove focus")
		assert.Equal(t, "navigate", result[0].ActionType)
		assert.Equal(t, "click", result[1].ActionType)
		assert.Equal(t, "type", result[2].ActionType)
		assert.Equal(t, "username", result[2].Payload["text"], "Should merge type text")
		assert.Equal(t, "click", result[3].ActionType)
	})
}

func TestMergeConsecutiveActions_ScrollsMergedRegardlessOfPage(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] merges consecutive scrolls (uses final position)", func(t *testing.T) {
		now := time.Now()

		// Note: Current implementation merges all consecutive scrolls regardless of URL
		// This is expected behavior as scroll position typically matters for the final state
		actions := []RecordedAction{
			createTestAction("scroll", "", now, "https://example.com/page1"),
			createTestAction("scroll", "", now.Add(50*time.Millisecond), "https://example.com/page2"),
		}
		actions[0].Payload = map[string]interface{}{"scrollY": 100.0}
		actions[1].Payload = map[string]interface{}{"scrollY": 200.0}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 1, "Should merge consecutive scrolls")
		assert.Equal(t, 200.0, result[0].Payload["scrollY"], "Should use final scroll position")
	})
}

func TestMergeConsecutiveActions_TypesInterruptedByClick(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] does not merge types interrupted by other actions", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("type", "#input", now, "https://example.com"),
			createTestAction("click", "#somewhere", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("type", "#input", now.Add(200*time.Millisecond), "https://example.com"),
		}
		actions[0].Payload = map[string]interface{}{"text": "hello"}
		actions[2].Payload = map[string]interface{}{"text": "world"}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 3, "Should not merge types interrupted by click")
		assert.Equal(t, "hello", result[0].Payload["text"])
		assert.Equal(t, "click", result[1].ActionType)
		assert.Equal(t, "world", result[2].Payload["text"])
	})
}

func TestMergeConsecutiveActions_TypeWithNilPayload(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] handles type actions with nil payload gracefully", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("type", "#input", now, "https://example.com"),
			createTestAction("type", "#input", now.Add(100*time.Millisecond), "https://example.com"),
		}
		// First action has nil payload
		actions[1].Payload = map[string]interface{}{"text": "hello"}

		result := mergeConsecutiveActions(actions)

		// Should still merge, treating nil text as empty string
		require.Len(t, result, 1, "Should merge even with nil payload")
		assert.Equal(t, "type", result[0].ActionType)
	})
}

func TestMergeConsecutiveActions_ConsecutiveClicks(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] does not merge consecutive clicks", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("click", "#btn1", now, "https://example.com"),
			createTestAction("click", "#btn2", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("click", "#btn3", now.Add(200*time.Millisecond), "https://example.com"),
		}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 3, "Should not merge clicks")
		assert.Equal(t, "#btn1", result[0].Selector.Primary)
		assert.Equal(t, "#btn2", result[1].Selector.Primary)
		assert.Equal(t, "#btn3", result[2].Selector.Primary)
	})
}

func TestMergeConsecutiveActions_ScrollAfterNavigate(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] preserves scroll after navigate", func(t *testing.T) {
		now := time.Now()

		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com"),
			createTestAction("scroll", "", now.Add(500*time.Millisecond), "https://example.com"),
		}
		actions[1].Payload = map[string]interface{}{"scrollY": 500.0}

		result := mergeConsecutiveActions(actions)

		require.Len(t, result, 2, "Should preserve both actions")
		assert.Equal(t, "navigate", result[0].ActionType)
		assert.Equal(t, "scroll", result[1].ActionType)
		assert.Equal(t, 500.0, result[1].Payload["scrollY"])
	})
}

func TestSelectorsMatch_BothNil(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] selectorsMatch returns false when both selectors are nil", func(t *testing.T) {
		var sel1 *SelectorSet = nil
		var sel2 *SelectorSet = nil

		result := selectorsMatch(sel1, sel2)

		assert.False(t, result, "Should return false when both selectors are nil")
	})
}

func TestSelectorsMatch_OneNil(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] selectorsMatch returns false when one selector is nil", func(t *testing.T) {
		action1 := createTestAction("type", "#input", time.Now(), "https://example.com")
		var sel2 *SelectorSet = nil

		result := selectorsMatch(action1.Selector, sel2)

		assert.False(t, result, "Should return false when one selector is nil")
	})
}

func TestSelectorsMatch_DifferentPrimary(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] selectorsMatch returns false for different primary selectors", func(t *testing.T) {
		action1 := createTestAction("type", "#input1", time.Now(), "https://example.com")
		action2 := createTestAction("type", "#input2", time.Now(), "https://example.com")

		result := selectorsMatch(action1.Selector, action2.Selector)

		assert.False(t, result, "Should return false for different primary selectors")
	})
}

func TestSelectorsMatch_SamePrimary(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] selectorsMatch returns true for same primary selectors", func(t *testing.T) {
		action1 := createTestAction("type", "#input", time.Now(), "https://example.com")
		action2 := createTestAction("type", "#input", time.Now(), "https://example.com")

		result := selectorsMatch(action1.Selector, action2.Selector)

		assert.True(t, result, "Should return true for same primary selectors")
	})
}

// =============================================================================
// E2E Tests: Record → Generate → Execute Flow
// =============================================================================

// recordModeTestHandler creates a Handler configured for record mode testing
func recordModeTestHandler(t *testing.T, workflowSvc compositeWorkflowService) *Handler {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	hub := wsHub.NewHub(log)

	h := &Handler{
		log:              log,
		wsHub:            hub,
		workflowCatalog:  workflowSvc,
		executionService: workflowSvc,
		exportService:    workflowSvc,
	}

	return h
}

// mockPlaywrightDriver creates a test server that simulates the playwright driver
func mockPlaywrightDriver(t *testing.T, actions []RecordedAction) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && contains(r.URL.Path, "/record/start"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"recording_id": "rec-123",
				"session_id":   "session-456",
				"started_at":   time.Now().Format(time.RFC3339),
			})

		case r.Method == http.MethodPost && contains(r.URL.Path, "/record/stop"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"recording_id": "rec-123",
				"action_count": len(actions),
				"stopped_at":   time.Now().Format(time.RFC3339),
			})

		case r.Method == http.MethodGet && contains(r.URL.Path, "/record/actions"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(GetActionsResponse{
				SessionID: "session-456",
				Actions:   actions,
				Count:     len(actions),
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// recordModeWorkflowServiceMock is a minimal mock for workflow service
type recordModeWorkflowServiceMock struct {
	createdWorkflow *database.Workflow
	createErr       error
}

func (m *recordModeWorkflowServiceMock) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	workflowID := uuid.New()
	m.createdWorkflow = &database.Workflow{
		ID:             workflowID,
		ProjectID:      projectID,
		Name:           name,
		FlowDefinition: flowDefinition,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return m.createdWorkflow, nil
}

// Stub implementations for WorkflowService interface - not used in these tests
func (m *recordModeWorkflowServiceMock) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input workflow.WorkflowUpdateInput) (*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*workflow.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*workflow.WorkflowVersionSummary, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*workflow.ExecutionExportPreview, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	return nil, errors.New("not implemented")
}
func (m *recordModeWorkflowServiceMock) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) CreateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) UpdateProject(ctx context.Context, project *database.Project) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	return nil, nil
}
func (m *recordModeWorkflowServiceMock) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error {
	return nil
}
func (m *recordModeWorkflowServiceMock) CheckAutomationHealth(ctx context.Context) (bool, error) {
	return true, nil
}

// Compile-time check that mock implements interface
var _ compositeWorkflowService = (*recordModeWorkflowServiceMock)(nil)

func TestE2E_GenerateWorkflowWithInlineActions(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] generates workflow from inline actions without driver", func(t *testing.T) {
		now := time.Now()

		// Create test actions - simulating a login flow
		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com/login"),
			createTestAction("click", "#username", now.Add(500*time.Millisecond), "https://example.com/login"),
			createTestAction("type", "#username", now.Add(600*time.Millisecond), "https://example.com/login"),
			createTestAction("click", "#password", now.Add(1*time.Second), "https://example.com/login"),
			createTestAction("type", "#password", now.Add(1100*time.Millisecond), "https://example.com/login"),
			createTestAction("click", "#submit", now.Add(2*time.Second), "https://example.com/login"),
		}
		actions[2].Payload = map[string]interface{}{"text": "testuser"}
		actions[4].Payload = map[string]interface{}{"text": "password123"}

		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		// Create request with inline actions
		reqBody := GenerateWorkflowRequest{
			SessionID: "session-123",
			Name:      "Login Workflow",
			Actions:   actions,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session-123/generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		// Add URL parameter
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "session-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "Expected 201 Created, got %d: %s", rr.Code, rr.Body.String())

		var resp GenerateWorkflowResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err, "Failed to parse response")

		assert.Equal(t, "Login Workflow", resp.Name)
		assert.Equal(t, 6, resp.ActionCount, "Should have 6 actions")
		assert.Greater(t, resp.NodeCount, 6, "Should have more nodes than actions due to wait nodes")

		// Verify workflow was created with correct flow definition
		require.NotNil(t, workflowMock.createdWorkflow)
		flowDef := workflowMock.createdWorkflow.FlowDefinition
		nodes := flowDef["nodes"].([]map[string]interface{})
		edges := flowDef["edges"].([]map[string]interface{})

		assert.Greater(t, len(nodes), 0, "Should have nodes")
		assert.Equal(t, len(nodes)-1, len(edges), "Edges should connect all nodes")

		// Verify we have wait nodes inserted
		hasWait := false
		for _, node := range nodes {
			if node["type"] == "wait" {
				hasWait = true
				break
			}
		}
		assert.True(t, hasWait, "Should have wait nodes for smart waits")
	})
}

func TestE2E_GenerateWorkflowFromDriver(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] generates workflow by fetching actions from playwright driver", func(t *testing.T) {
		now := time.Now()

		// Actions that will be returned by mock driver
		actions := []RecordedAction{
			createTestAction("click", "#btn", now, "https://example.com"),
			createTestAction("type", "#input", now.Add(500*time.Millisecond), "https://example.com"),
		}
		actions[1].Payload = map[string]interface{}{"text": "hello world"}

		// Start mock playwright driver
		mockDriver := mockPlaywrightDriver(t, actions)
		defer mockDriver.Close()

		// Set environment variable to point to mock driver
		originalURL := os.Getenv("PLAYWRIGHT_DRIVER_URL")
		os.Setenv("PLAYWRIGHT_DRIVER_URL", mockDriver.URL)
		defer os.Setenv("PLAYWRIGHT_DRIVER_URL", originalURL)

		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		// Create request without inline actions - will fetch from driver
		reqBody := GenerateWorkflowRequest{
			SessionID: "session-456",
			Name:      "Driver Workflow",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session-456/generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "session-456")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "Expected 201 Created, got %d: %s", rr.Code, rr.Body.String())

		var resp GenerateWorkflowResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "Driver Workflow", resp.Name)
		assert.Equal(t, 2, resp.ActionCount)
	})
}

func TestE2E_GenerateWorkflowWithActionRange(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] generates workflow from subset of actions using action range", func(t *testing.T) {
		now := time.Now()

		// 5 actions, but we only want actions 1-3
		actions := []RecordedAction{
			createTestAction("click", "#step1", now, "https://example.com"),
			createTestAction("click", "#step2", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("click", "#step3", now.Add(200*time.Millisecond), "https://example.com"),
			createTestAction("click", "#step4", now.Add(300*time.Millisecond), "https://example.com"),
			createTestAction("click", "#step5", now.Add(400*time.Millisecond), "https://example.com"),
		}

		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		// Request with action range - only include actions 1-3 (indices 1, 2, 3)
		reqBody := GenerateWorkflowRequest{
			SessionID: "session-123",
			Name:      "Ranged Workflow",
			Actions:   actions,
			ActionRange: &struct {
				Start int `json:"start"`
				End   int `json:"end"`
			}{Start: 1, End: 3},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session-123/generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "session-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "Expected 201 Created")

		var resp GenerateWorkflowResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, 3, resp.ActionCount, "Should only include 3 actions from range")
	})
}

func TestE2E_GenerateWorkflowNoActionsFromDriver(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] returns error when driver returns no actions", func(t *testing.T) {
		// Mock driver that returns empty actions
		emptyActionsDriver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && contains(r.URL.Path, "/record/actions") {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(GetActionsResponse{
					SessionID: "session-456",
					Actions:   []RecordedAction{},
					Count:     0,
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer emptyActionsDriver.Close()

		// Set driver URL
		originalURL := os.Getenv("PLAYWRIGHT_DRIVER_URL")
		os.Setenv("PLAYWRIGHT_DRIVER_URL", emptyActionsDriver.URL)
		defer os.Setenv("PLAYWRIGHT_DRIVER_URL", originalURL)

		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		// Request without inline actions - will fetch from driver which returns empty
		reqBody := GenerateWorkflowRequest{
			SessionID: "session-456",
			Name:      "Empty Workflow",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session-456/generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "session-456")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code, "Should return 400 for empty actions from driver")

		var errResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &errResp)
		if details, ok := errResp["details"].(map[string]interface{}); ok {
			assert.Contains(t, details["error"], "No actions", "Error should mention no actions")
		}
	})
}

func TestE2E_GenerateWorkflowMissingSessionID(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] returns error when session ID is missing", func(t *testing.T) {
		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		reqBody := GenerateWorkflowRequest{
			Name: "Test Workflow",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live//generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		// Empty session ID
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code, "Should return 400 for missing session ID")
	})
}

func TestE2E_WorkflowGeneration_NodeTypes(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] generates correct node types for each action type", func(t *testing.T) {
		now := time.Now()

		// Test all supported action types
		actions := []RecordedAction{
			createTestAction("navigate", "", now, "https://example.com"),
			createTestAction("click", "#btn", now.Add(100*time.Millisecond), "https://example.com"),
			createTestAction("type", "#input", now.Add(200*time.Millisecond), "https://example.com"),
			createTestAction("scroll", "", now.Add(300*time.Millisecond), "https://example.com"),
			createTestAction("hover", "#menu", now.Add(400*time.Millisecond), "https://example.com"),
		}
		// Set required payload for type action
		actions[2].Payload = map[string]interface{}{"text": "test"}
		// Set navigate target
		actions[0].Payload = map[string]interface{}{"url": "https://example.com"}
		// Set scroll payload
		actions[3].Payload = map[string]interface{}{"scrollY": 500.0}

		workflowMock := &recordModeWorkflowServiceMock{}
		handler := recordModeTestHandler(t, workflowMock)

		reqBody := GenerateWorkflowRequest{
			SessionID: "session-123",
			Name:      "Multi-Type Workflow",
			Actions:   actions,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/live/session-123/generate-workflow", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("sessionId", "session-123")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()
		handler.GenerateWorkflowFromRecording(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code)

		// Verify generated nodes contain expected types
		flowDef := workflowMock.createdWorkflow.FlowDefinition
		nodes := flowDef["nodes"].([]map[string]interface{})

		nodeTypes := make(map[string]bool)
		for _, node := range nodes {
			nodeType := node["type"].(string)
			nodeTypes[nodeType] = true
		}

		assert.True(t, nodeTypes["navigate"], "Should have navigate node")
		assert.True(t, nodeTypes["click"], "Should have click node")
		assert.True(t, nodeTypes["type"], "Should have type node")
		assert.True(t, nodeTypes["scroll"], "Should have scroll node")
		assert.True(t, nodeTypes["wait"], "Should have wait nodes inserted")
	})
}
