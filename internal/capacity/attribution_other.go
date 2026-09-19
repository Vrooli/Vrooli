//go:build !linux

package capacity

// nilCgroupSource is the non-linux cgroup seam: container attribution via
// /proc/<pid>/cgroup is linux-specific, so other platforms resolve no container
// and every PID degrades cleanly to an unknown owner (cross-platform-readiness).
type nilCgroupSource struct{}

func newProcCgroupSource() CgroupSource { return nilCgroupSource{} }

func (nilCgroupSource) ContainerID(int) (string, bool) { return "", false }
