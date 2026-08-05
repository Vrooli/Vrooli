package sysmounts

// VolumeIdentity is metadata returned by the platform volume adapter.
type VolumeIdentity struct {
	Label  string
	UUID   string
	Model  string
	Serial string
}
