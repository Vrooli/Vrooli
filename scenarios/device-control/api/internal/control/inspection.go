package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"device-control/strategy"
	"github.com/google/uuid"
)

type SystemInventoryRequest struct {
	DeviceID, Actor, LeaseToken, Transport string
}

type SystemInventoryResponse struct {
	DeviceID, Transport string
	Inventory           strategy.SystemInventory
	Audit               Audit
}

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

func (s *Service) InspectSystemInventory(ctx context.Context, request SystemInventoryRequest) (SystemInventoryResponse, error) {
	request.DeviceID = strings.TrimSpace(request.DeviceID)
	request.Actor = strings.TrimSpace(request.Actor)
	if request.DeviceID == "" || request.Actor == "" || strings.TrimSpace(request.LeaseToken) == "" {
		return SystemInventoryResponse{}, fmt.Errorf("device, actor, and held lease token are required")
	}
	session, err := s.sessionForLease(ctx, request.DeviceID, request.LeaseToken)
	if err != nil {
		return SystemInventoryResponse{}, fmt.Errorf("validate inventory lease: %w", err)
	}
	if session.Actor != request.Actor {
		return SystemInventoryResponse{}, fmt.Errorf("inventory actor does not own the device lease")
	}
	adapter, ok := s.strategyForFlow(request.DeviceID, request.Transport)
	if !ok {
		return SystemInventoryResponse{}, fmt.Errorf("device transport is unknown")
	}
	transport := request.Transport
	if transport == "" {
		if record, found := s.devices.Get(request.DeviceID); found {
			transport = record.Transport
		}
	}
	inspector, ok := adapter.(strategy.SystemInventoryInspector)
	if !ok {
		return SystemInventoryResponse{}, fmt.Errorf("selected transport does not support Android system inventory")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	inventory, inspectErr := inspector.InspectSystemInventory(probeCtx)
	outcome := "success"
	if inspectErr != nil {
		outcome = "unavailable"
	}
	packagesCount := len(inventory.InstalledPackages)
	audit := Audit{ID: uuid.NewString(), Actor: request.Actor, DeviceID: request.DeviceID, Transport: transport, LeaseID: session.ID, Verb: "device_system_inventory", Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: false, EvidenceBacked: false, Details: map[string]string{"cpu_cores_reported": fmt.Sprint(inventory.CPULogicalCores > 0), "memory_reported": fmt.Sprint(inventory.MemoryTotalBytes != nil), "storage_reported": fmt.Sprint(inventory.DataTotalBytes != nil), "installed_package_count": fmt.Sprint(packagesCount), "packages_truncated": fmt.Sprint(inventory.PackagesTruncated)}}
	s.persistDirectAudit(ctx, audit)
	return SystemInventoryResponse{DeviceID: request.DeviceID, Transport: transport, Inventory: inventory, Audit: audit}, inspectErr
}
