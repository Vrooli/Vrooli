package session

import (
	"context"
	"errors"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

// ObserveAccessibilityTree authorizes one logical observe operation against
// this session's current page and immutable lease before returning its AX tree.
func (s *Session) ObserveAccessibilityTree(ctx context.Context, expectedPageID string) (*driver.AccessibilityTreeResponse, error) {
	if s.isClosed() {
		return nil, errors.New("session closed")
	}
	if expectedPageID == "" {
		expectedPageID = s.currentDriverPageID()
	}
	if err := s.authorizeActionRequestForPage(ctx, sessionpermission.ActionRequest{Kind: "observe"}, expectedPageID); err != nil {
		return nil, err
	}
	return s.client.GetAccessibilityTree(ctx, s.id, s.executionID, s.leaseID, expectedPageID)
}
