//go:build !linux

package hostwireless

import (
	"context"
	"errors"
)

func runNativeWirelessOperation(_ context.Context, operation string, req WirelessOperationRequest, _ bool) (WirelessOperationResult, error) {
	return WirelessOperationResult{Operation: operation, Profile: req.Profile, State: "unsupported", Checks: []WirelessCheck{{Name: "platform", Detail: "reviewed wireless transaction requires Linux amd64 Ubuntu Noble"}}}, errors.New("wireless activation is unsupported on this platform")
}
