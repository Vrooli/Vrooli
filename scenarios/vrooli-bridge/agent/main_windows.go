//go:build windows

package main

import (
	"context"
	"log"

	"golang.org/x/sys/windows/svc"
)

// runWindowsService enters the SCM dispatcher only when the executable was
// launched by Windows Services. Console invocations, including `service
// install|status|uninstall`, continue through the ordinary main path.
func runWindowsService(args []string) (bool, error) {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return false, err
	}
	if !isService {
		return false, nil
	}
	name := "vrooli-bridge-agent"
	for _, arg := range args {
		if arg == "--provision-helper" {
			name = "vrooli-bridge-provisioner"
			break
		}
	}
	return true, svc.Run(name, windowsService{args: args})
}

type windowsService struct {
	args []string
}

func (s windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, statuses chan<- svc.Status) (bool, uint32) {
	statuses <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- runWithContext(ctx, s.args) }()

	statuses <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case err := <-done:
			if err != nil {
				log.Printf("Windows service stopped with error: %v", err)
				return false, 1
			}
			return false, 0
		case request := <-requests:
			switch request.Cmd {
			case svc.Stop, svc.Shutdown:
				statuses <- svc.Status{State: svc.StopPending}
				cancel()
				if err := <-done; err != nil {
					log.Printf("Windows service stopped with error: %v", err)
					return false, 1
				}
				return false, 0
			}
		}
	}
}
