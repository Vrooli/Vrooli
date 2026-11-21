package resources

import (
	"fmt"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/process"
)

// The caller holds the group lifecycle lock. Only the supervised ownership
// inventory is authoritative; GPU process names and open ports are not.
func checkExclusiveResourceGroup(controller *Controller, manifest ResourceManifest) error {
	if controller == nil {
		return fmt.Errorf("exclusive group requires a resource controller")
	}
	records, err := ReadManagedServiceRecords()
	if err != nil {
		return fmt.Errorf("read exclusive group ownership: %w", err)
	}
	for _, record := range records {
		if record.Resource == manifest.Name || !process.IsPIDRunning(record.State.PID) {
			continue
		}
		peer, err := controller.LoadManifest(filepath.Join(controller.Root, "resources", record.Resource, "resource.json"))
		if err != nil {
			return fmt.Errorf("read possible exclusive-group peer %s: %w", record.Resource, err)
		}
		if peer.Lifecycle.ExclusiveGroup != manifest.Lifecycle.ExclusiveGroup {
			continue
		}
		supervisor, _, err := managedServiceSupervisorFor(peer.Name)
		if err != nil {
			return err
		}
		_, running, err := supervisor.Status()
		if err != nil {
			return fmt.Errorf("inspect exclusive-group peer %s: %w", peer.Name, err)
		}
		if running {
			return fmt.Errorf("exclusive group %q is occupied by %s; finish its work and use vrooli resource stop %s before starting %s", manifest.Lifecycle.ExclusiveGroup, peer.Name, peer.Name, manifest.Name)
		}
	}
	return nil
}
