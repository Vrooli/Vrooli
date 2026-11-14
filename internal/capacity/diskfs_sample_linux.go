//go:build linux

package capacity

// diskJobFilesystemIdentity retains kernel mount identity for sampling paths
// that may belong to Docker and may not be traversable by the capacity owner.
func diskJobFilesystemIdentity(path string) (string, string, error) {
	return filesystemFromMountInfo(path)
}
