package sysmounts

// VolumeIdentity is metadata returned by the platform volume adapter.
type VolumeIdentity struct {
	Label  string
	UUID   string
	Model  string
	Serial string
	// Filesystem is the on-disk filesystem type as reported by the platform
	// adapter. It is the only way to learn the filesystem of a volume that is
	// not mounted, where no mount entry exists to read it from. Empty when the
	// adapter cannot determine it.
	Filesystem string
}
