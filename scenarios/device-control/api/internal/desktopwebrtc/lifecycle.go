package desktopwebrtc

import (
	"context"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// LifecycleInventory reports every session owned by this manager. Active
// sessions, pending capture cancellation, in-flight provider calls, and
// failed cleanup remain visible blockers until the owner has a terminal
// receipt.
func (m *Manager) LifecycleInventory(ctx context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
	if err := ctx.Err(); err != nil {
		return nil, []*commonv1.LifecycleBlocker{{Code: "desktop_webrtc_inventory_unavailable", Message: err.Error(), Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true}}, false
	}
	if m == nil {
		return []*commonv1.LifecycleWorkItem{{Id: "owner/desktop-webrtc", Kind: "desktop-webrtc", Status: "idle"}}, nil, true
	}
	if !m.mu.TryLock() {
		return []*commonv1.LifecycleWorkItem{{Id: "owner/desktop-webrtc", Kind: "desktop-webrtc", Status: "unknown"}}, []*commonv1.LifecycleBlocker{{Code: "desktop_webrtc_inventory_busy", Message: "desktop WebRTC owner state is being updated and could not be snapshotted", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true}}, false
	}
	defer m.mu.Unlock()

	if len(m.sessions) == 0 {
		return []*commonv1.LifecycleWorkItem{{Id: "owner/desktop-webrtc", Kind: "desktop-webrtc", Status: "idle"}}, nil, true
	}
	work := make([]*commonv1.LifecycleWorkItem, 0, len(m.sessions))
	blockers := make([]*commonv1.LifecycleBlocker, 0, len(m.sessions))
	complete := true
	for id, s := range m.sessions {
		if s == nil || s.session == nil {
			complete = false
			work = append(work, &commonv1.LifecycleWorkItem{Id: "desktop-webrtc/" + id, Kind: "desktop-webrtc", Status: "unknown"})
			blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_webrtc_owner_state_unknown", Message: "desktop WebRTC session " + id + " has no complete owner state", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
			continue
		}

		status := "active"
		disposition := commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING
		message := "desktop WebRTC session " + id + " remains open"
		switch {
		case s.cleanupError != "":
			status = "uncertain"
			disposition = commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN
			message = "desktop WebRTC session " + id + " cleanup is unconfirmed: " + s.cleanupError
		case s.revoked && (s.cleanupPending || s.inFlight != 0 || captureRunning(s.captureDone)):
			status = "stopping"
			message = "desktop WebRTC session " + id + " cleanup has not reached a terminal receipt"
		case s.revoked:
			status = "completed"
		}
		work = append(work, &commonv1.LifecycleWorkItem{Id: "desktop-webrtc/" + id, Kind: "desktop-webrtc", Status: status})
		if status != "completed" {
			blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_webrtc_work_pending", Message: message, Disposition: disposition, Observed: true})
		}
	}
	return work, blockers, complete
}

func captureRunning(done <-chan struct{}) bool {
	if done == nil {
		return false
	}
	select {
	case <-done:
		return false
	default:
		return true
	}
}
