package infra

import (
	"context"
)

// getAutoLoginUser is the session-posture seam. It remains read-only and is
// kept separate from credential mutation so autoheal cannot mint credentials.
func (c *RDPCheck) getAutoLoginUser() string {
	if c.autoLoginUserProvider != nil {
		return c.autoLoginUserProvider()
	}
	return autoLoginUser()
}

// findGraphicalSessionUser is the action/session seam. Keeping the session
// lookup here prevents recovery execution from growing another copy of the
// display-session discovery logic.
func (c *RDPCheck) findGraphicalSessionUser(ctx context.Context) string {
	return seat0SessionUser(ctx, c.executor)
}
