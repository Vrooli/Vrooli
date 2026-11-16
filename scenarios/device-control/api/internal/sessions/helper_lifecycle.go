package sessions

// HelperLifecycleStatus is the durable helper process lifecycle reported by
// DesktopUnixHelper while it drains and releases controller resources.
type HelperLifecycleStatus string

const (
	HelperLifecycleRunning                    HelperLifecycleStatus = "running"
	HelperLifecycleStopping                   HelperLifecycleStatus = "stopping"
	HelperLifecycleControllerCleanupCompleted HelperLifecycleStatus = "controller-cleanup-completed"
	HelperLifecycleUncertain                  HelperLifecycleStatus = "uncertain"
)

func (s HelperLifecycleStatus) Valid() bool {
	switch s {
	case HelperLifecycleRunning, HelperLifecycleStopping, HelperLifecycleControllerCleanupCompleted, HelperLifecycleUncertain:
		return true
	default:
		return false
	}
}

// Transition accepts only owner-observed shutdown milestones. Repeated
// stopping reports are valid because a drain timeout is still in progress.
func (s HelperLifecycleStatus) Transition(next HelperLifecycleStatus) (HelperLifecycleStatus, bool) {
	valid := false
	switch s {
	case "":
		valid = next == HelperLifecycleRunning
	case HelperLifecycleRunning:
		valid = next == HelperLifecycleStopping
	case HelperLifecycleStopping:
		valid = next == HelperLifecycleStopping || next == HelperLifecycleControllerCleanupCompleted || next == HelperLifecycleUncertain
	}
	return next, valid
}
