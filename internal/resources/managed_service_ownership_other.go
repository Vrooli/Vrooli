//go:build !linux

package resources

func managedServiceExecutableMatchesArtifact(ManagedServiceState) bool {
	return false
}
