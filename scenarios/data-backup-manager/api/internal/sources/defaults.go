package sources

// NewProductionRegistry wires all six capturers into a Registry using the
// real resource CLIs. The filesystem and sqlite capturers perform pure file
// I/O and ignore runner; the remaining four shell out through runner.
//
// Call this once at process start and share the resulting *Registry.
func NewProductionRegistry(runner CommandRunner) *Registry {
	return NewRegistry(
		newFilesystemCapturer(),
		newSQLiteCapturer(),
		newPostgresCapturer(runner),
		newRedisCapturer(runner),
		newQdrantCapturer(runner),
		newObjectStorageCapturer(runner),
	)
}
