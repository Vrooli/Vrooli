package androidadb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// OpenReverseTCP keeps route creation and rollback at the ADB owner boundary.
// Existing bindings are never borrowed, replaced, or removed by this operation.
func (a *Adapter) OpenReverseTCP(ctx context.Context, devicePort, hostPort int) (func(context.Context) error, error) {
	if devicePort < 1024 || devicePort > 65535 || hostPort < 1024 || hostPort > 65535 {
		return nil, fmt.Errorf("reverse TCP endpoints must be unprivileged numeric ports")
	}
	device, host := "tcp:localhost:"+strconv.Itoa(devicePort), "tcp:localhost:"+strconv.Itoa(hostPort)
	bindings, err := a.reverseTCPBindings(ctx)
	if err != nil {
		return nil, err
	}
	// A bare tcp:<port> binds all device interfaces on Android 11. Always
	// request localhost explicitly, and refuse aliases using the same port.
	for endpoint := range bindings {
		if !strings.HasPrefix(endpoint, "tcp:") {
			continue
		}
		port, parseErr := strconv.Atoi(endpoint[strings.LastIndex(endpoint, ":")+1:])
		if parseErr != nil {
			return nil, fmt.Errorf("existing reverse TCP endpoint is not recognizable")
		}
		if port == devicePort {
			return nil, fmt.Errorf("reverse TCP device port %d already has a binding; refusing rebind", devicePort)
		}
	}

	cleanup := func(cleanupCtx context.Context) error { return a.RemoveReverseTCP(cleanupCtx, devicePort, hostPort) }

	creator, ok := a.runner.(reverseTCPCreator)
	if !ok {
		return nil, fmt.Errorf("reverse TCP requires the owner protocol client")
	}
	selector := a.serial
	if a.endpoint != "" {
		selector = a.endpoint
	}
	err = creator.createReverseTCP(ctx, selector, devicePort, hostPort)
	if err != nil {
		return cleanup, fmt.Errorf("create reverse TCP mapping: %w", err)
	}
	after, err := a.reverseTCPBindings(ctx)
	if err != nil {
		return cleanup, err
	}
	if after[device] != host && after["tcp:"+strconv.Itoa(devicePort)] != host {
		return cleanup, fmt.Errorf("reverse TCP setup was not verified")
	}
	return cleanup, nil
}

func (a *Adapter) reverseTCPBindings(ctx context.Context) (map[string]string, error) {
	output, err := a.runner.Run(ctx, "adb", a.args("reverse", "--list")...)
	if err != nil {
		return nil, fmt.Errorf("inspect reverse TCP bindings: %w", err)
	}
	if len(output) > 64<<10 {
		return nil, fmt.Errorf("reverse TCP binding list exceeds bounded size")
	}
	bindings := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("reverse TCP binding list is not recognizable")
		}
		if _, exists := bindings[fields[1]]; exists {
			return nil, fmt.Errorf("reverse TCP binding identity is ambiguous")
		}
		bindings[fields[1]] = fields[2]
	}
	return bindings, nil
}

// RemoveReverseTCP is an explicit, lease-scoped recovery operation. It removes
// only the selected port with the exact expected host-loopback destination.
// Android 11 normalizes every resolved listener name to tcp:<port>, even when
// it actually bound localhost. The requested endpoint is not its listed name.
func (a *Adapter) RemoveReverseTCP(ctx context.Context, devicePort, hostPort int) error {
	if devicePort < 1024 || devicePort > 65535 || hostPort < 1024 || hostPort > 65535 {
		return fmt.Errorf("invalid reverse TCP recovery ports")
	}
	current, err := a.reverseTCPBindings(ctx)
	if err != nil {
		return err
	}
	device, host := "tcp:localhost:"+strconv.Itoa(devicePort), "tcp:localhost:"+strconv.Itoa(hostPort)
	key := ""
	for endpoint, target := range current {
		if endpoint != device && endpoint != "tcp:"+strconv.Itoa(devicePort) {
			continue
		}
		if target != host || key != "" {
			return fmt.Errorf("reverse TCP binding changed or ambiguous; refusing to remove an unowned mapping")
		}
		key = endpoint
	}
	if key == "" {
		return nil
	}
	if _, err := a.runner.Run(ctx, "adb", a.args("reverse", "--remove", key)...); err != nil {
		return fmt.Errorf("remove owned reverse TCP mapping: %w", err)
	}
	after, err := a.reverseTCPBindings(ctx)
	if err != nil {
		return err
	}
	for endpoint := range after {
		if strings.HasPrefix(endpoint, "tcp:") {
			port, err := strconv.Atoi(endpoint[strings.LastIndex(endpoint, ":")+1:])
			if err != nil || port == devicePort {
				return fmt.Errorf("reverse TCP mapping removal was not verified")
			}
		}
	}
	return nil
}
