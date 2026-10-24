package livecapture

import (
	"fmt"
	"slices"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/actions"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

type timelineWorkflowStep struct {
	entry     *bastimeline.TimelineEntry
	page      string
	framePath []string
	url       string
	at        time.Time
	kind      string
	selector  string
}

// GenerateWorkflowFromTimelineEntries keeps driver-buffered proto entries
// canonical through workflow planning and clones only at the workflow boundary.
func (g *WorkflowGenerator) GenerateWorkflowFromTimelineEntries(entries []*bastimeline.TimelineEntry, pages []*domain.Page) (*basworkflows.WorkflowDefinitionV2, error) {
	return generateWorkflowFromTimelineEntries(entries, pages)
}

func generateWorkflowFromTimelineEntries(entries []*bastimeline.TimelineEntry, pages []*domain.Page) (*basworkflows.WorkflowDefinitionV2, error) {
	steps := make([]timelineWorkflowStep, 0, len(entries))
	pageByDriver := make(map[string]*domain.Page, len(pages))
	var initial *domain.Page
	for _, page := range pages {
		if page == nil {
			continue
		}
		if page.IsInitial {
			if initial != nil && initial.ID != page.ID {
				return nil, fmt.Errorf("recording has duplicate initial page bindings")
			}
			initial = page
		}
		if page.DriverPageID != "" {
			if pageByDriver[page.DriverPageID] != nil {
				return nil, fmt.Errorf("recording has duplicate bindings for driver page %q", page.DriverPageID)
			}
			pageByDriver[page.DriverPageID] = page
		}
	}
	for index, entry := range entries {
		if entry == nil {
			continue
		}
		kind := enums.ActionTypeToString(entry.GetAction().GetType())
		if entry.GetAction().GetType() == basactions.ActionType_ACTION_TYPE_INPUT {
			kind = "type" // The workflow planner registry uses the capture alias.
		}
		path := slices.Clone(entry.GetTelemetry().GetFramePath())
		if entry.GetTelemetry().GetFrameId() != "" && len(path) == 0 {
			return nil, fmt.Errorf("recorded entry %d: frame replay requires a captured frame selector path", index+1)
		}
		if len(path) > 32 {
			return nil, fmt.Errorf("recorded entry %d: frame selector path exceeds 32 levels", index+1)
		}
		for _, selector := range path {
			if selector == "" {
				return nil, fmt.Errorf("recorded entry %d: frame selector path contains an empty selector", index+1)
			}
		}
		page := pageByDriver[entry.GetTelemetry().GetDriverPageId()]
		if entry.GetTelemetry().GetDriverPageId() != "" && page == nil && len(pageByDriver) > 0 {
			return nil, fmt.Errorf("recorded entry %d: driver page %q has no logical page binding", index+1, entry.GetTelemetry().GetDriverPageId())
		}
		if page == nil && len(pageByDriver) > 1 {
			return nil, fmt.Errorf("recorded entry %d: multi-page recording is missing its logical page identity", index+1)
		}
		if page == nil && index == 0 && kind == "navigate" {
			page = initial
		}
		pageKey := "initial"
		if page != nil {
			pageKey = page.ID.String()
		} else if entry.GetTelemetry().GetDriverPageId() != "" {
			pageKey = "unbound:" + entry.GetTelemetry().GetDriverPageId()
		}
		at := time.Time{}
		if entry.GetTimestamp() != nil {
			at = entry.GetTimestamp().AsTime()
		}
		step := timelineWorkflowStep{entry: entry, page: pageKey, framePath: path, url: entry.GetTelemetry().GetUrl(), at: at, kind: kind, selector: timelineSelector(entry.GetAction())}
		if err := validateTimelineStepPage(step, page, index, at); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("no timeline entries to convert")
	}
	flow := &basworkflows.WorkflowDefinitionV2{}
	appendNode := func(action *basactions.ActionDefinition) {
		i := len(flow.Nodes)
		node := &basworkflows.WorkflowNodeV2{Id: fmt.Sprintf("node_%d", i+1), Action: action, Position: &basbase.NodePosition{X: 250, Y: float64(100 + i*120)}}
		if i > 0 {
			flow.Edges = append(flow.Edges, &basworkflows.WorkflowEdgeV2{Id: fmt.Sprintf("edge_%d", i), Source: flow.Nodes[i-1].Id, Target: node.Id})
		}
		flow.Nodes = append(flow.Nodes, node)
	}
	pageState := newTimelinePageState(pages, steps)
	var previous *timelineWorkflowStep
	var previousFrame []string
	for index := range steps {
		step := &steps[index]
		if previous != nil {
			if err := pageState.transition(index, *previous, *step, appendNode); err != nil {
				return nil, err
			}
			appendFramePathTransition(appendNode, previousFrame, step.framePath)
			if wait := timelineTransitionWait(*previous, *step); wait != nil {
				appendNode(waitAction(wait))
			}
		}
		action, err := timelineWorkflowAction(step.entry)
		if err != nil {
			return nil, fmt.Errorf("recorded entry %d (%s): %w", index+1, step.kind, err)
		}
		appendNode(action)
		previous, previousFrame = step, step.framePath
	}
	return flow, nil
}

func ApplyTimelineEntryRange(entries []*bastimeline.TimelineEntry, start, end int) []*bastimeline.TimelineEntry {
	if start < 0 {
		start = 0
	}
	if end >= len(entries) {
		end = len(entries) - 1
	}
	if start <= end && start < len(entries) {
		return entries[start : end+1]
	}
	return entries
}

func timelineWorkflowAction(entry *bastimeline.TimelineEntry) (*basactions.ActionDefinition, error) {
	action := proto.Clone(entry.GetAction()).(*basactions.ActionDefinition)
	switch action.Type {
	case basactions.ActionType_ACTION_TYPE_CLICK, basactions.ActionType_ACTION_TYPE_INPUT,
		basactions.ActionType_ACTION_TYPE_NAVIGATE, basactions.ActionType_ACTION_TYPE_SCROLL,
		basactions.ActionType_ACTION_TYPE_SELECT, basactions.ActionType_ACTION_TYPE_FOCUS,
		basactions.ActionType_ACTION_TYPE_BLUR, basactions.ActionType_ACTION_TYPE_HOVER,
		basactions.ActionType_ACTION_TYPE_KEYBOARD, basactions.ActionType_ACTION_TYPE_DRAG_DROP,
		basactions.ActionType_ACTION_TYPE_WAIT, basactions.ActionType_ACTION_TYPE_ASSERT,
		basactions.ActionType_ACTION_TYPE_SCREENSHOT:
	default:
		return nil, fmt.Errorf("unsupported recorded action type %q", enums.ActionTypeToString(action.Type))
	}
	if action.Type == basactions.ActionType_ACTION_TYPE_INPUT {
		params := action.GetInput()
		if params == nil {
			return nil, fmt.Errorf("input observation has no typed input parameters")
		}
		params.ClearFirst = proto.Bool(true)
	}
	if action.Metadata == nil {
		action.Metadata = &basactions.ActionMetadata{}
	}
	if action.Metadata.Label == nil {
		label := enums.ActionTypeToString(action.Type)
		action.Metadata.Label = &label
	}
	return action, nil
}

func timelineSelector(action *basactions.ActionDefinition) string {
	switch params := action.GetParams().(type) {
	case *basactions.ActionDefinition_Click:
		return params.Click.GetSelector()
	case *basactions.ActionDefinition_Input:
		return params.Input.GetSelector()
	case *basactions.ActionDefinition_Hover:
		return params.Hover.GetSelector()
	case *basactions.ActionDefinition_Focus:
		return params.Focus.GetSelector()
	case *basactions.ActionDefinition_Blur:
		return params.Blur.GetSelector()
	case *basactions.ActionDefinition_SelectOption:
		return params.SelectOption.GetSelector()
	case *basactions.ActionDefinition_Assert:
		return params.Assert.GetSelector()
	case *basactions.ActionDefinition_Scroll:
		return params.Scroll.GetSelector()
	default:
		return ""
	}
}

func validateTimelineStepPage(step timelineWorkflowStep, page *domain.Page, index int, at time.Time) error {
	if page == nil {
		return nil
	}
	if !page.CreatedAt.IsZero() && !at.IsZero() && at.Before(page.CreatedAt) {
		return fmt.Errorf("recorded entry %d: target page %q did not exist yet", index+1, page.ID)
	}
	if page.Status == domain.PageStatusClosed && (page.ClosedAt == nil || at.IsZero() || !at.Before(*page.ClosedAt)) {
		return fmt.Errorf("recorded entry %d: target page %q is not replayable", index+1, page.ID)
	}
	return nil
}

func timelineTransitionWait(current, next timelineWorkflowStep) *WaitTemplate {
	changes := actions.TriggersDOMChanges(actions.ActionType(current.kind))
	gap := next.at.Sub(current.at).Milliseconds()
	if actions.NeedsSelectorWait(actions.ActionType(next.kind)) && next.selector != "" && (changes || current.url != next.url || gap > 500) {
		return &WaitTemplate{WaitType: "selector", Selector: next.selector, TimeoutMs: 10000, Label: "Wait for " + next.selector}
	}
	if gap > 2000 {
		duration := gap / 2
		if duration > 5000 {
			duration = 5000
		}
		return &WaitTemplate{WaitType: "timeout", TimeoutMs: int(duration), Label: "Wait for page to stabilize"}
	}
	return nil
}

type timelinePageState struct {
	pages  map[string]*domain.Page
	steps  []timelineWorkflowStep
	open   []string
	active string
	closed []*domain.Page
}

func newTimelinePageState(pages []*domain.Page, steps []timelineWorkflowStep) *timelinePageState {
	s := &timelinePageState{pages: make(map[string]*domain.Page), steps: steps}
	for _, p := range pages {
		if p != nil {
			s.pages[p.ID.String()] = p
			if p.Status == domain.PageStatusClosed && p.ClosedAt != nil {
				s.closed = append(s.closed, p)
			}
		}
	}
	slices.SortFunc(s.closed, func(a, b *domain.Page) int { return a.ClosedAt.Compare(*b.ClosedAt) })
	if len(steps) > 0 {
		s.active = steps[0].page
		s.open = []string{s.active}
	}
	return s
}

func (s *timelinePageState) transition(index int, previous, next timelineWorkflowStep, appendNode func(*basactions.ActionDefinition)) error {
	if err := s.applyClosures(index, previous.at, next.at, appendNode); err != nil {
		return err
	}
	if next.page == s.active {
		return nil
	}
	if tab := pageIndex(s.open, next.page); tab >= 0 {
		appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, int32(tab), ""))
		s.active = next.page
		return nil
	}
	p := s.pages[next.page]
	if p == nil {
		return fmt.Errorf("recorded entry %d: target page %q is not replayable", index+1, next.page)
	}
	if p.OpenerID != nil {
		createdByPrevious := index > 0 && previous.page == p.OpenerID.String() && (previous.kind == "click" || previous.kind == "keyboard") && !p.CreatedAt.IsZero() && !previous.at.IsZero() && !next.at.IsZero() && p.CreatedAt.After(previous.at) && p.CreatedAt.Before(next.at)
		if !createdByPrevious {
			return fmt.Errorf("recorded entry %d: popup page %q has ambiguous opener timing", index+1, next.page)
		}
		s.open = append(s.open, next.page)
		appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH, int32(len(s.open)-1), ""))
		s.active = next.page
		return nil
	}
	url := next.url
	if url == "" {
		url = p.URL
	}
	s.open = append(s.open, next.page)
	appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN, 0, url))
	s.active = next.page
	return nil
}

func (s *timelinePageState) applyClosures(index int, previousAt, currentAt time.Time, appendNode func(*basactions.ActionDefinition)) error {
	if index <= 0 {
		return nil
	}
	if previousAt.IsZero() || currentAt.IsZero() {
		for _, p := range s.closed {
			if pageIndex(s.open, p.ID.String()) >= 0 {
				return fmt.Errorf("recorded close for page %q is ambiguous relative to entry %d", p.ID, index+1)
			}
		}
		return nil
	}
	for _, p := range s.closed {
		id := p.ID.String()
		closed := *p.ClosedAt
		if closed.After(currentAt) {
			continue
		}
		tab := pageIndex(s.open, id)
		if tab < 0 {
			continue
		}
		if !closed.After(previousAt) || !closed.Before(currentAt) {
			return fmt.Errorf("recorded close for page %q is ambiguous relative to entry %d", id, index+1)
		}
		if len(s.open) == 1 {
			return fmt.Errorf("recorded close would remove the last replay tab before entry %d", index+1)
		}
		appendNode(tabSwitchAction(basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE, int32(tab), ""))
		s.open = append(s.open[:tab], s.open[tab+1:]...)
		if s.active == id {
			if tab >= len(s.open) {
				tab = len(s.open) - 1
			}
			s.active = s.open[tab]
		}
	}
	return nil
}
