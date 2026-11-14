package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPhysicalDiskCounterParsingExcludesUnselectedAndRejectsPartial(t *testing.T) {
	data := "259 0 nvme0n1 10 0 100 0 20 0 200 0 0 0 0\n259 1 nvme0n1p1 10 0 999 0 20 0 999 0 0 0 0\n8 0 sda 10 0 50 0 20 0 25 0 0 0 0\n"
	got, err := parsePhysicalDiskCounters(data, map[string]bool{"nvme0n1": true, "sda": true})
	if err != nil || got["nvme0n1"].readSectors != 100 || got["sda"].writeSectors != 25 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := parsePhysicalDiskCounters(data, map[string]bool{"nvme0n1": true, "missing": true}); err == nil {
		t.Fatal("partial inventory accepted")
	}
}

func TestMeasuredProviderRequiresConsecutiveValidCounters(t *testing.T) {
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	samples := []diskCounters{{"sda": {100, 200}}, {"sda": {200, 400}}, {"sda": {1, 2}}, {"other": {3, 4}}}
	p := NewMeasuredProvider()
	p.now = func() time.Time { return now }
	p.read = func() (diskCounters, error) { sample := samples[0]; samples = samples[1:]; return sample, nil }
	first, _ := p.GetInfrastructureMonitorData(context.Background())
	if first.StorageIO.ReadRate.Status != "not_yet_sampled" {
		t.Fatalf("first = %+v", first.StorageIO.ReadRate)
	}
	now = now.Add(2 * time.Second)
	second, _ := p.GetInfrastructureMonitorData(context.Background())
	if second.StorageIO.ReadRate.Status != "measured" || second.StorageIO.ReadRate.Value != 0.0256 || second.StorageIO.WriteRate.Value != 0.0512 {
		t.Fatalf("second = %+v", second.StorageIO)
	}
	now = now.Add(time.Second)
	reset, _ := p.GetInfrastructureMonitorData(context.Background())
	if reset.StorageIO.ReadRate.Status != "not_yet_sampled" {
		t.Fatalf("reset = %+v", reset.StorageIO.ReadRate)
	}
	now = now.Add(time.Second)
	changed, _ := p.GetInfrastructureMonitorData(context.Background())
	if changed.StorageIO.ReadRate.Status != "not_yet_sampled" {
		t.Fatalf("inventory change = %+v", changed.StorageIO.ReadRate)
	}
	p.read = func() (diskCounters, error) { return diskCounters{}, errDiskUnsupported }
	now = now.Add(time.Second)
	unsupported, _ := p.GetInfrastructureMonitorData(context.Background())
	if unsupported.StorageIO.ReadRate.Status != "unsupported" {
		t.Fatalf("unsupported = %+v", unsupported.StorageIO.ReadRate)
	}
	p.read = func() (diskCounters, error) { return diskCounters{}, errors.New("permission denied") }
	failed, _ := p.GetInfrastructureMonitorData(context.Background())
	if failed.StorageIO.ReadRate.Status != "failed" {
		t.Fatalf("failed = %+v", failed.StorageIO.ReadRate)
	}
}

func TestMeasuredProviderSharesObservationAcrossRapidReaders(t *testing.T) {
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	calls := 0
	p := NewMeasuredProvider()
	p.now = func() time.Time { return now }
	p.read = func() (diskCounters, error) {
		calls++
		return diskCounters{"sda": {uint64(calls * 100), uint64(calls * 200)}}, nil
	}
	first, _ := p.GetInfrastructureMonitorData(context.Background())
	now = now.Add(500 * time.Millisecond)
	rapid, _ := p.GetInfrastructureMonitorData(context.Background())
	if calls != 1 || rapid.StorageIO.ReadRate.Status != "not_yet_sampled" || !rapid.StorageIO.ReadRate.ObservedAt.Equal(first.StorageIO.ReadRate.ObservedAt) {
		t.Fatalf("rapid reader did not share observation: calls=%d result=%+v", calls, rapid.StorageIO)
	}
	now = now.Add(500 * time.Millisecond)
	measured, _ := p.GetInfrastructureMonitorData(context.Background())
	if calls != 2 || measured.StorageIO.ReadRate.Status != "measured" || measured.StorageIO.ReadRate.Value != 0.0512 {
		t.Fatalf("second observation = %+v, calls=%d", measured.StorageIO, calls)
	}
}

func TestDiskCounterDeltasRejectsIndividualResetHiddenByOtherDevice(t *testing.T) {
	previous := diskCounters{"a": {100, 100}, "b": {100, 100}}
	current := diskCounters{"a": {1, 1}, "b": {500, 500}}
	if _, _, ok := diskCounterDeltas(previous, current); ok {
		t.Fatal("individual reset was hidden by aggregate growth")
	}
}
