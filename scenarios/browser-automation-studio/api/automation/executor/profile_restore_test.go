package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

type restorableProfileSession struct {
	engine.EngineSession
	initialID string
	navigated string
	created   []string
	popups    []string
	active    string
}

func (s *restorableProfileSession) Run(_ context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	s.navigated = instruction.Action.GetNavigate().GetUrl()
	return contracts.StepOutcome{Success: true}, nil
}

func (s *restorableProfileSession) CreatePage(_ context.Context, url string) (*driver.CreatePageResponse, error) {
	s.created = append(s.created, url)
	return &driver.CreatePageResponse{DriverPageID: "restored-" + url}, nil
}

func (s *restorableProfileSession) CreatePopupPage(_ context.Context, url, openerID string) (*driver.CreatePageResponse, error) {
	s.popups = append(s.popups, openerID+"->"+url)
	return &driver.CreatePageResponse{DriverPageID: "popup-" + url}, nil
}

func (s *restorableProfileSession) SetActivePage(_ context.Context, pageID string) error {
	s.active = pageID
	return nil
}

func (s *restorableProfileSession) InitialDriverPageID() string { return s.initialID }

func TestMaybeRestoreTabsRestoresAllSavedTabsAndActiveSelection(t *testing.T) {
	session := &restorableProfileSession{initialID: "initial"}
	req := Request{
		RestoreTabs: true,
		Plan:        contracts.ExecutionPlan{ExecutionID: uuid.New()},
		OpenTabs: []persistence.TabState{
			{URL: "https://active.test", IsActive: true, Order: 2, OpenerOrder: intPointer(1)},
			{URL: "https://first.test", Order: 0},
			{URL: "https://second.test", Order: 1},
		},
	}

	got, err := NewSimpleExecutor(nil).maybeRestoreTabs(context.Background(), req, nil, engine.SessionSpec{}, session)
	require.NoError(t, err)
	require.Same(t, session, got)
	require.Equal(t, "https://first.test", session.navigated)
	require.Equal(t, []string{"https://second.test"}, session.created)
	require.Equal(t, []string{"restored-https://second.test->https://active.test"}, session.popups)
	require.Equal(t, "popup-https://active.test", session.active)
}

func TestMaybeRestoreTabsRejectsUnsupportedMultiPageRestore(t *testing.T) {
	plain := &profileRestoreSinglePageSession{}
	req := Request{RestoreTabs: true, OpenTabs: []persistence.TabState{{URL: "https://one.test"}, {URL: "https://two.test", Order: 1}}}

	_, err := NewSimpleExecutor(nil).maybeRestoreTabs(context.Background(), req, nil, engine.SessionSpec{}, plain)
	require.ErrorContains(t, err, "does not support restoring multiple profile tabs")
	require.Empty(t, plain.navigated, "unsupported restore must fail before navigating")
}

type profileRestoreSinglePageSession struct {
	engine.EngineSession
	navigated string
}

func (s *profileRestoreSinglePageSession) Run(_ context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	s.navigated = instruction.Action.GetNavigate().GetUrl()
	return contracts.StepOutcome{Success: true}, nil
}

func intPointer(value int) *int { return &value }
