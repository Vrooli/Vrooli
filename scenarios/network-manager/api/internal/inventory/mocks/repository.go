package mocks

import (
	"context"
	"sort"
	"sync"

	"network-manager/internal/inventory"
)

type Repository struct {
	mu      sync.Mutex
	devices map[string]inventory.Device
}

func NewRepository() *Repository {
	return &Repository{devices: map[string]inventory.Device{}}
}

func (r *Repository) SaveDevice(_ context.Context, device inventory.Device) (inventory.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[device.ID] = device
	return device, nil
}

func (r *Repository) GetDevice(_ context.Context, id string) (inventory.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	device, ok := r.devices[id]
	if !ok {
		return inventory.Device{}, inventory.ErrNotFound
	}
	return device, nil
}

func (r *Repository) ListDevices(_ context.Context, group string) ([]inventory.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]inventory.Device, 0, len(r.devices))
	for _, device := range r.devices {
		if group != "" && device.Group != group {
			continue
		}
		out = append(out, device)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *Repository) UpdateGroup(ctx context.Context, id, group string) (inventory.Device, error) {
	device, err := r.GetDevice(ctx, id)
	if err != nil {
		return inventory.Device{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	device.Group = group
	r.devices[id] = device
	return device, nil
}

var _ inventory.Repository = (*Repository)(nil)
