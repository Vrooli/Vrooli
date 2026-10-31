package fetch

// DiskSpaceFunc reports the free bytes available at path's filesystem. The
// production implementation (DefaultDiskAvail) wraps statfs; tests inject a fake.
type DiskSpaceFunc func(path string) (availBytes int64, err error)

// DefaultDiskAvail is the production DiskSpaceFunc (statfs-backed on unix, a
// large-value fallback elsewhere). Exported so wiring can reference it without a
// build-tagged reference.
func DefaultDiskAvail(path string) (int64, error) { return diskAvail(path) }
