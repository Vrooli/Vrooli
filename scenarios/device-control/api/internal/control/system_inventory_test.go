package control

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type inventoryFixture struct {
	*fakes.Strategy
	inventory strategy.SystemInventory
	called    bool
}

func (f *inventoryFixture) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{{ID: "phone", Serial: "serial", IdentityKey: "serial", IdentityKind: "adb-serial", StrategyID: f.ID(), Transport: "usb", Health: strategy.StatusAvailable}}, nil
}

func (f *inventoryFixture) InspectSystemInventory(context.Context) (strategy.SystemInventory, error) {
	f.called = true
	return f.inventory, nil
}

func TestInspectSystemInventoryRequiresOwningLeaseAndAuditsSummary(t *testing.T) {
	fixture := &inventoryFixture{Strategy: fakes.New("inventory", strategy.StatusAvailable), inventory: strategy.SystemInventory{CPULogicalCores: 8, InstalledPackages: []string{"com.example.app"}, ObservedAt: time.Now().UTC()}}
	db, err := sql.Open("sqlite", "file:system-inventory-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc, err := NewWithDB(strategyregistry.New(fixture), db)
	require.NoError(t, err)
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	_, err = svc.InspectSystemInventory(context.Background(), SystemInventoryRequest{DeviceID: "phone", Actor: "operator"})
	require.ErrorContains(t, err, "lease token are required")
	require.False(t, fixture.called)
	lease, err := svc.Acquire("phone", "operator", time.Minute)
	require.NoError(t, err)
	result, err := svc.InspectSystemInventory(context.Background(), SystemInventoryRequest{DeviceID: "phone", Actor: "operator", LeaseToken: lease.LeaseToken})
	require.NoError(t, err)
	require.Equal(t, uint32(8), result.Inventory.CPULogicalCores)
	require.Equal(t, lease.ID, result.Audit.LeaseID)
	require.Equal(t, "device_system_inventory", result.Audit.Verb)
	require.Equal(t, "1", result.Audit.Details["installed_package_count"])
	require.NotContains(t, result.Audit.Details, "com.example.app")
	records := svc.AuditContext(context.Background())
	require.Len(t, records, 1)
	require.False(t, records[0].Interactive)
	require.NotContains(t, fmt.Sprint(records[0].Details), "com.example.app")
}

func TestInspectSystemInventoryRejectsDifferentLeaseActor(t *testing.T) {
	fixture := &inventoryFixture{Strategy: fakes.New("inventory", strategy.StatusAvailable)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	lease, err := svc.Acquire("phone", "owner", time.Minute)
	require.NoError(t, err)
	_, err = svc.InspectSystemInventory(context.Background(), SystemInventoryRequest{DeviceID: "phone", Actor: "other", LeaseToken: lease.LeaseToken})
	require.ErrorContains(t, err, "does not own")
	require.False(t, fixture.called)
}
