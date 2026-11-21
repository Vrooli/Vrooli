package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/setupprivilege"
)

// runSetupBroker is a focused project-setup operation. It deliberately avoids
// host-requirement apply, credentials, resources and global completion markers.
func (s *setupService) runSetupBroker(root string, opts Options, stdout, stderr io.Writer) error {
	if err := opts.Context.Err(); err != nil {
		return err
	}
	if err := s.deps.currentHost().ValidateSetup(); err != nil {
		return err
	}
	if _, err := s.deps.loadProject(root); err != nil {
		return err
	}
	var status privilegebroker.SetupStatus
	if opts.DryRun {
		status = s.deps.inspectPrivilegeBroker()
	} else {
		executable, err := s.deps.osExecutable()
		if err != nil {
			return err
		}
		var owned *setupprivilege.Session
		if opts.setupPrivilegeSession == nil && shouldCreateSetupPrivilegeSession(opts.SudoMode, s.deps.elevationFacts()) {
			owned = setupprivilege.NewSession(opts.Context, executable, stdout, stderr)
			opts.setupPrivilegeSession = owned
		}
		status, err = s.deps.installPrivilegeBroker(opts.Context, executable, opts, stdout, stderr)
		if owned != nil {
			err = errors.Join(err, owned.Close())
		}
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
		return fmt.Errorf("privilege broker requires setup: %s; use `%s`", status.Reason, privilegebroker.SetupBrokerRecoveryCommand)
	}
	return nil
}
