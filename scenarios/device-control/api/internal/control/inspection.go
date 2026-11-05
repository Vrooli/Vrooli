package control

import (
	"context"
	"fmt"
	"time"

	"device-control/strategy"
)

func (s *Service) InspectDevice(ctx context.Context, deviceID, transport string) (strategy.DeviceInspection, error) {
	adapter, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return strategy.DeviceInspection{}, fmt.Errorf("device transport is unknown")
	}
	inspector, ok := adapter.(strategy.Inspector)
	if !ok {
		return strategy.DeviceInspection{}, fmt.Errorf("selected transport does not support read-only runtime inspection")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return inspector.Inspect(probeCtx)
}
