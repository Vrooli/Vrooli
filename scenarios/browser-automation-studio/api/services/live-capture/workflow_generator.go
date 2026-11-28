// Package livecapture provides business logic for live capture mode functionality.
// This includes converting recorded actions to workflows, action merging, and smart wait insertion.
package livecapture

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/actions"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

// WorkflowGenerator converts recorded actions into workflow definitions.
type WorkflowGenerator struct{}

// NewWorkflowGenerator creates a new workflow generator.
func NewWorkflowGenerator() *WorkflowGenerator {
	return &WorkflowGenerator{}
}

// GenerateWorkflow derives typed candidates; unsupported observations never become clicks.
func (g *WorkflowGenerator) GenerateWorkflow(recorded []driver.RecordedAction) (*basworkflows.WorkflowDefinitionV2, error) {
	return g.GenerateWorkflowWithPages(recorded, nil)
}

// GenerateWorkflowWithPages derives tab switches from the recording session's
// existing page tracker. A popup created by the immediately preceding observed
// action is selected rather than opened a second time.
func (g *WorkflowGenerator) GenerateWorkflowWithPages(recorded []driver.RecordedAction, pages []*domain.Page) (*basworkflows.WorkflowDefinitionV2, error) {
	actions, err := prepareRecordedActions(recorded)
	if err != nil {
		return nil, err
	}
	actions = MergeConsecutiveActions(actions)
	tabState, err := newWorkflowTabState(actions, pages)
	if err != nil {
		return nil, err
	}
	flow := &basworkflows.WorkflowDefinitionV2{}
	appendNode := func(action *basactions.ActionDefinition) {
		index := len(flow.Nodes)
		node := &basworkflows.WorkflowNodeV2{Id: fmt.Sprintf("node_%d", index+1), Action: action, Position: &basbase.NodePosition{X: 250, Y: float64(100 + index*120)}}
		if index > 0 {
			flow.Edges = append(flow.Edges, &basworkflows.WorkflowEdgeV2{Id: fmt.Sprintf("edge_%d", index), Source: flow.Nodes[index-1].Id, Target: node.Id})
		}
		flow.Nodes = append(flow.Nodes, node)
	}
	for index, recordedAction := range actions {
		if err := tabState.validateTargetAt(index, recordedAction); err != nil {
			return nil, err
		}
		if index > 0 {
			if err := tabState.transition(index, recordedAction, appendNode); err != nil {
				return nil, err
			}
			appendFramePathTransition(appendNode, actions[index-1].FramePath, recordedAction.FramePath)
			if wait := analyzeTransitionForWait(actions[index-1], recordedAction); wait != nil {
				appendNode(waitAction(wait))
			}
		}
		action, err := recordedActionDefinition(recordedAction)
		if err != nil {
			return nil, fmt.Errorf("recorded action %d (%s): %w", index+1, recordedAction.ActionType, err)
		}
		appendNode(action)
	}
	return flow, nil
}

type workflowTabState struct {
	targets  []string
	pages    map[string]*domain.Page
	actions  []driver.RecordedAction
	closures []*domain.Page
	open     []string
	active   string
}

func newWorkflowTabState(actions []driver.RecordedAction, pages []*domain.Page) (*workflowTabState, error) {
	targets, pageByID, err := resolveRecordedPageTargets(actions, pages)
	if err != nil {
		return nil, err
	}
	state := &workflowTabState{targets: targets, pages: pageByID, actions: actions}
	for _, page := range pages {
		if page != nil && page.Status == domain.PageStatusClosed && page.ClosedAt != nil {
			state.closures = append(state.closures, page)
		}
	}
	sort.Slice(state.closures, func(i, j int) bool {
		return state.closures[i].ClosedAt.Before(*state.closures[j].ClosedAt)
	})
	if len(targets) > 0 {
		state.active = targets[0]
		state.open = []string{state.active}
	}
	return state, nil
}

func (s *workflowTabState) transition(index int, action driver.RecordedAction, appendNode func(*basactions.ActionDefinition)) error {
	if err := s.applyRecordedClosures(index, action.Timestamp, appendNode); err != nil {
		return err
	}
	target := s.targets[index]
	if target == s.active {
		return nil
	}
	if tabIndex := pageIndex(s.open, target); tabIndex >= 0 {
		appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, int32(tabIndex), ""))
		s.active = target
		return nil
	}

	page := s.pages[target]
	if page == nil {
		return fmt.Errorf("recorded action %d: target page %q is not replayable", index+1, target)
	}
	if page.OpenerID != nil {
		if popupCreatedByPreviousAction(page, s.targets, s.actions, index) {
			s.open = append(s.open, target)
			appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, int32(len(s.open)-1), ""))
			s.active = target
			return nil
		}
		return fmt.Errorf("recorded action %d: popup page %q has ambiguous opener timing", index+1, target)
	}

	url := action.URL
	if url == "" {
		url = page.URL
	}
	s.open = append(s.open, target)
	appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN, 0, url))
	s.active = target
	return nil
}

func (s *workflowTabState) validateTargetAt(index int, action driver.RecordedAction) error {
	target := s.targets[index]
	page := s.pages[target]
	if page == nil {
		return nil
	}
	actionAt := parseRecordedActionTime(action)
	if !page.CreatedAt.IsZero() && !actionAt.IsZero() && actionAt.Before(page.CreatedAt) {
		return fmt.Errorf("recorded action %d: target page %q did not exist yet", index+1, target)
	}
	if page.Status == domain.PageStatusClosed && (page.ClosedAt == nil || actionAt.IsZero() || !actionAt.Before(*page.ClosedAt)) {
		return fmt.Errorf("recorded action %d: target page %q is not replayable", index+1, target)
	}
	return nil
}

// applyRecordedClosures replays page closes that are causally between adjacent
// recorded actions. A close with an ambiguous timestamp fails closed while its
// page is part of the replay stack.
func (s *workflowTabState) applyRecordedClosures(index int, timestamp string, appendNode func(*basactions.ActionDefinition)) error {
	if index <= 0 {
		return nil
	}
	previousAt := parseRecordedActionTime(s.actions[index-1])
	currentAt, _ := time.Parse(time.RFC3339Nano, timestamp)
	if previousAt.IsZero() || currentAt.IsZero() {
		for _, page := range s.closures {
			if pageIndex(s.open, page.ID.String()) >= 0 {
				return fmt.Errorf("recorded close for page %q is ambiguous relative to action %d", page.ID, index+1)
			}
		}
		return nil
	}
	for _, page := range s.closures {
		pageID := page.ID.String()
		closedAt := *page.ClosedAt
		if closedAt.After(currentAt) {
			continue
		}
		tabIndex := pageIndex(s.open, pageID)
		if tabIndex < 0 {
			continue // Pages are opened at first use, so an unused closed page has no replay effect.
		}
		if !closedAt.After(previousAt) || !closedAt.Before(currentAt) {
			return fmt.Errorf("recorded close for page %q is ambiguous relative to action %d", pageID, index+1)
		}
		for _, other := range s.closures {
			if other.ID != page.ID && other.ClosedAt.Equal(closedAt) && pageIndex(s.open, other.ID.String()) >= 0 {
				return fmt.Errorf("recorded closes for pages %q and %q share an ambiguous timestamp", pageID, other.ID)
			}
		}
		if len(s.open) == 1 {
			return fmt.Errorf("recorded close would remove the last replay tab before action %d", index+1)
		}
		appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE, int32(tabIndex), ""))
		s.open = append(s.open[:tabIndex], s.open[tabIndex+1:]...)
		if s.active == pageID {
			nextIndex := tabIndex
			if nextIndex >= len(s.open) {
				nextIndex = len(s.open) - 1
			}
			s.active = s.open[nextIndex]
		}
	}
	return nil
}

func popupCreatedByPreviousAction(page *domain.Page, targets []string, actions []driver.RecordedAction, index int) bool {
	if page == nil || page.OpenerID == nil || page.CreatedAt.IsZero() || index <= 0 || targets[index-1] != page.OpenerID.String() {
		return false
	}
	if index >= len(actions) || (actions[index-1].ActionType != "click" && actions[index-1].ActionType != "keyboard") {
		return false
	}
	previousTime := parseRecordedActionTime(actions[index-1])
	currentTime := parseRecordedActionTime(actions[index])
	return !previousTime.IsZero() && !currentTime.IsZero() && page.CreatedAt.After(previousTime) &&
		(page.CreatedAt.Before(currentTime) || page.CreatedAt.Equal(currentTime))
}

func parseRecordedActionTime(action driver.RecordedAction) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, action.Timestamp)
	return parsed
}

func pageIndex(pages []string, target string) int {
	for index, page := range pages {
		if page == target {
			return index
		}
	}
	return -1
}

func tabSwitchAction(action basactions.TabSwitchAction, index int32, url string) *basactions.ActionDefinition {
	params := &basactions.TabSwitchParams{Action: action}
	label := "Switch tab"
	if action == basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN {
		label = "Open tab"
		params.Url = proto.String(url)
	} else {
		params.Index = proto.Int32(index)
	}
	return &basactions.ActionDefinition{
		Type:     basactions.ActionType_ACTION_TYPE_TAB_SWITCH,
		Params:   &basactions.ActionDefinition_TabSwitch{TabSwitch: params},
		Metadata: &basactions.ActionMetadata{Label: proto.String(label)},
	}
}

func appendFramePathTransition(appendNode func(*basactions.ActionDefinition), from, to []string) {
	shared := 0
	for shared < len(from) && shared < len(to) && from[shared] == to[shared] {
		shared++
	}
	for index := len(from); index > shared; index-- {
		appendNode(frameSwitchAction(basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_PARENT, ""))
	}
	for _, selector := range to[shared:] {
		appendNode(frameSwitchAction(basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER, selector))
	}
}

func frameSwitchAction(action basactions.FrameSwitchAction, selector string) *basactions.ActionDefinition {
	label := "Return to parent frame"
	params := &basactions.FrameSwitchParams{Action: action}
	if action == basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER {
		label = "Enter recorded frame"
		params.Selector = proto.String(selector)
	}
	return &basactions.ActionDefinition{
		Type:   basactions.ActionType_ACTION_TYPE_FRAME_SWITCH,
		Params: &basactions.ActionDefinition_FrameSwitch{FrameSwitch: params},
		Metadata: &basactions.ActionMetadata{
			Label: proto.String(label),
		},
	}
}

// MergeConsecutiveActions coalesces full input/scroll snapshots only within one
// target. It never writes through the recorded payload maps.
func MergeConsecutiveActions(recorded []driver.RecordedAction) []driver.RecordedAction {
	if recorded == nil {
		return nil
	}
	merged := make([]driver.RecordedAction, 0, len(recorded))
	for index, action := range recorded {
		if action.ActionType == "focus" && index+1 < len(recorded) && recorded[index+1].ActionType == "type" && sameTarget(action, recorded[index+1]) {
			continue
		}
		if len(merged) > 0 {
			previous := merged[len(merged)-1]
			if previous.ActionType == action.ActionType && sameTarget(previous, action) && coalescibleSnapshots(previous, action) {
				merged[len(merged)-1] = action
				continue
			}
		}
		merged = append(merged, action)
	}
	return merged
}

func sameTarget(a, b driver.RecordedAction) bool {
	if a.PageID != b.PageID || a.DriverPageID != b.DriverPageID || a.FrameID != b.FrameID || a.URL != b.URL || !slices.Equal(a.FramePath, b.FramePath) {
		return false
	}
	if a.Selector == nil || b.Selector == nil {
		return a.Selector == nil && b.Selector == nil
	}
	return a.Selector.Primary == b.Selector.Primary
}

func coalescibleSnapshots(previous, next driver.RecordedAction) bool {
	switch next.ActionType {
	case "type":
		_, old := previous.Payload["text"].(string)
		_, current := next.Payload["text"].(string)
		return old && current && previous.Payload["submit"] != true
	case "scroll":
		_, oldX := previous.Payload["scrollX"]
		_, oldY := previous.Payload["scrollY"]
		_, newX := next.Payload["scrollX"]
		_, newY := next.Payload["scrollY"]
		return (oldX || oldY) && oldX == newX && oldY == newY
	default:
		return false
	}
}

// ApplyActionRange returns the requested action subset, clamping indices to the available actions.
func ApplyActionRange(actions []driver.RecordedAction, start, end int) []driver.RecordedAction {
	if len(actions) == 0 {
		return actions
	}

	if start < 0 {
		start = 0
	}
	if end >= len(actions) {
		end = len(actions) - 1
	}
	if start <= end && start < len(actions) {
		return actions[start : end+1]
	}
	return actions
}

// WaitTemplate describes a wait node to be inserted between actions.
type WaitTemplate struct {
	WaitType  string // "selector" or "timeout"
	Selector  string // For selector waits
	TimeoutMs int    // Timeout for selector waits, or duration for timeout waits
	Label     string // Human-readable label
}

// analyzeTransitionForWait examines two consecutive actions and determines
// if a wait node should be inserted between them.
// Returns nil if no wait is needed.
func analyzeTransitionForWait(current, next driver.RecordedAction) *WaitTemplate {
	// Check if the next action needs its selector to exist (uses action registry)
	if actions.NeedsSelectorWait(actions.ActionType(next.ActionType)) && next.Selector != nil && next.Selector.Primary != "" {
		// If current action might trigger DOM changes, add a wait (uses action registry)
		triggersChanges := actions.TriggersDOMChanges(actions.ActionType(current.ActionType))

		// Check for URL change (indicates navigation happened)
		urlChanged := current.URL != next.URL

		// Check for significant time gap (>500ms suggests async activity)
		var timeDiff int64
		if current.Timestamp != "" && next.Timestamp != "" {
			currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
			nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
			if err1 == nil && err2 == nil {
				timeDiff = nextTime.Sub(currentTime).Milliseconds()
			}
		}
		significantGap := timeDiff > 500

		// Insert wait if any condition is met
		if triggersChanges || urlChanged || significantGap {
			label := fmt.Sprintf("Wait for %s", describeElement(next))
			return &WaitTemplate{
				WaitType:  "selector",
				Selector:  next.Selector.Primary,
				TimeoutMs: 10000, // 10 second default timeout
				Label:     label,
			}
		}
	}

	// Check for large time gaps that suggest async operations even without selector needs
	if current.Timestamp != "" && next.Timestamp != "" {
		currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
		nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
		if err1 == nil && err2 == nil {
			timeDiff := nextTime.Sub(currentTime).Milliseconds()
			// If gap > 2 seconds, insert a proportional wait (capped at 5 seconds)
			if timeDiff > 2000 {
				waitDuration := timeDiff / 2 // Wait for half the observed gap
				if waitDuration > 5000 {
					waitDuration = 5000
				}
				return &WaitTemplate{
					WaitType:  "timeout",
					TimeoutMs: int(waitDuration),
					Label:     "Wait for page to stabilize",
				}
			}
		}
	}

	return nil
}

// describeElement creates a human-readable description of an element for labels.
func describeElement(action driver.RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 15)
			return fmt.Sprintf("\"%s\"", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return action.ElementMeta.AriaLabel
		}
		if action.ElementMeta.TagName != "" {
			return action.ElementMeta.TagName
		}
	}
	return "element"
}

func waitAction(template *WaitTemplate) *basactions.ActionDefinition {
	params := &basactions.WaitParams{TimeoutMs: proto.Int32(int32(template.TimeoutMs))}
	if template.WaitType == "selector" {
		params.WaitFor = &basactions.WaitParams_Selector{Selector: template.Selector}
		params.State = basactions.WaitState_WAIT_STATE_VISIBLE.Enum()
	} else {
		params.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: int32(template.TimeoutMs)}
	}
	return &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_WAIT, Params: &basactions.ActionDefinition_Wait{Wait: params}, Metadata: &basactions.ActionMetadata{Label: proto.String(template.Label)}}
}

// generateClickLabel creates a readable label for a click action.
func generateClickLabel(action driver.RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 20)
			return fmt.Sprintf("Click: %s", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return fmt.Sprintf("Click: %s", action.ElementMeta.AriaLabel)
		}
		return fmt.Sprintf("Click %s", action.ElementMeta.TagName)
	}
	return "Click element"
}

// truncateString truncates a string to maxLen and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractHostname extracts the hostname from a URL (or truncates if too long).
func extractHostname(urlStr string) string {
	if len(urlStr) > 50 {
		return urlStr[:50] + "..."
	}
	return urlStr
}
