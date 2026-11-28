package hostinventory

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/shell"
)

const darwinIOReg = "/usr/sbin/ioreg"

func collectNativeDevices(ctx context.Context, commands CommandRunner) NativeDeviceObservation {
	if commands == nil {
		commands = shell.OSRunner{}
	}
	service, serviceErr := readDarwinRegistry(ctx, commands, "IOService")
	firmware, firmwareErr := readDarwinRegistry(ctx, commands, "IODeviceTree")
	if errors.Is(serviceErr, os.ErrNotExist) && errors.Is(firmwareErr, os.ErrNotExist) {
		return unavailableNativeDevices(darwinRegistryMechanism, IntegrityProbeUnsupported, serviceErr)
	}
	return darwinRegistryObservation(service, serviceErr, firmware, firmwareErr)
}

func readDarwinRegistry(ctx context.Context, commands CommandRunner, plane string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := commands.Run(ctx, darwinIOReg, "-a", "-l", "-i", "-p", plane)
	if err != nil {
		return nil, fmt.Errorf("ioreg %s: %w", plane, err)
	}
	return out, nil
}
