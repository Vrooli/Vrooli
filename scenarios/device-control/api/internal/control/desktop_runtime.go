package control

import (
	"context"
	"time"

	"device-control/internal/desktophelper"
	"github.com/vrooli/api-core/owneridentity"
)

// RunDesktopOwner is an optional API-lifetime task. It provisions the managed
// helper's public configuration and keeps grant status live. Every restart
// creates a new admission owner; old grants are deliberately not reconstructed.
// The public HTTP SessionService does not gain desktop authority by enabling it.
func (s *Service) RunDesktopOwner(ctx context.Context, bootstrapPath string, report func(error)) {
	run := func() error {
		config, err := desktophelper.LoadOwnerBootstrap(bootstrapPath)
		if err != nil {
			return err
		}
		owner, err := s.ProvisionDesktopAdmission(config.HelperConfigPath, config.DeviceID, config.Helper)
		if err != nil {
			return err
		}
		owner.webSubject = config.OperatorSubject
		owner.webIdentity = owneridentity.NewClient(owneridentity.Config{Resolver: desktopIdentityResolver{}})
		listener, closeListener, err := desktophelper.ListenOwner(bootstrapPath)
		if err != nil {
			return err
		}
		defer closeListener()
		return owner.serveDesktopOwner(ctx, listener, config.Helper)
	}
	for ctx.Err() == nil {
		err := run()
		if ctx.Err() != nil {
			return
		}
		if err != nil && report != nil {
			report(err)
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
