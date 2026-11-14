package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// runSetupBroker is a focused project-setup operation. It deliberately avoids
// host-requirement apply, credentials, resources and global completion markers.
func (s *setupService) runSetupBroker(root string, opts Options, stdout, stderr io.Writer) error {
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
	status := s.deps.inspectPrivilegeBroker()
	if !opts.DryRun {
		executable, err := s.deps.osExecutable()
		if err != nil {
			return err
		}
		status, err = s.deps.installPrivilegeBroker(context.Background(), executable, opts, stdout, stderr)
		if err != nil {
			return err
		}
	}
	if opts.JSON {
		if err := json.NewEncoder(stdout).Encode(status); err != nil {
			return err
		}
	} else {
		renderPrivilegeBrokerStatus(stdout, status)
		if opts.DryRun {
			fmt.Fprintln(stdout, "Dry run: setup would install/upgrade the broker and its saved local-service policy; other host requirements are excluded.")
		}
	}
	if !opts.DryRun && (!status.Available || status.UpgradeRequired) {
		return fmt.Errorf("privilege broker remains unavailable: %s; use `vrooli setup broker --sudo-mode=ask`", status.Reason)
	}
	return nil
}
