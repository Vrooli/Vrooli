package bridge

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/api-core/nodereach"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// SessionClient is the nodereach surface that opens interactive sessions.
// nodereach.Client satisfies it; tests use fakes.
type SessionClient interface {
	Open(ctx context.Context, req nodereach.OpenRequest, timeout time.Duration) (*nodereach.Session, error)
}

// OpenSession opens the node's interactive channel through the Bridge. The
// Bridge resolves the grant; cloud passes only the node id, the bound
// workdir and the presentation size.
func (a *Adapter) OpenSession(ctx context.Context, target identity.TargetRef, spec reach.SessionSpec) (reach.Session, error) {
	if a == nil || a.Client == nil {
		return nil, &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge adapter not configured"}
	}
	opener, ok := a.Client.(SessionClient)
	if !ok {
		return nil, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge client has no session channel"}
	}
	node := strings.TrimSpace(target.NodeID)
	if node == "" {
		return nil, &reach.Error{Kind: reach.KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "target binding has no enrolled node"}
	}
	req := nodereach.OpenRequest{NodeID: node, WorkingDir: target.Locator.Workdir}
	if spec.Cols > 0 {
		req.Width = uint32(spec.Cols)
	}
	if spec.Rows > 0 {
		req.Height = uint32(spec.Rows)
	}
	timeout := a.DefaultTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	session, err := opener.Open(ctx, req, timeout)
	if err != nil {
		return nil, classify(err, target)
	}
	return &bridgeSession{Session: session}, nil
}

type bridgeSession struct{ *nodereach.Session }

func (s *bridgeSession) Resize(rows, cols int) error {
	return s.Session.Resize(uint32(cols), uint32(rows))
}

// Wait drains the read end until the Bridge ends the session; the terminal
// status is available on the underlying session afterwards.
func (s *bridgeSession) Wait() error {
	buf := make([]byte, 256)
	for {
		if _, err := s.Session.Read(buf); err != nil {
			return nil
		}
	}
}
