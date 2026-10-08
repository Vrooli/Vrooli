// Package cleanupmanager contains the small, scenario-neutral pieces shared
// by disk-pressure reporters. The HTTP clients remain owned by their callers;
// threshold classification is shared so reporters cannot drift apart.
package cleanupmanager

import (
	"encoding/json"
	"fmt"
	"os"
)

// Band is the portable pressure vocabulary sent to storage-manager.
type Band string

// Thresholds is the repository recovery contract consumed by pressure
// reporters. Keeping floor and rate settings together prevents a sender from
// applying a different emergency policy than the controller.
type Thresholds struct {
	Warning, High, Critical float64
	FloorBytes              int64
	FastFillPercent         float64
}

type recoveryContract struct {
	Storage struct {
		Recovery struct {
			FloorBytes      int64   `json:"floor_bytes"`
			FastFillPercent float64 `json:"fast_fill_percent"`
			Bands           struct {
				Warning  float64 `json:"warning"`
				High     float64 `json:"high"`
				Critical float64 `json:"critical"`
			} `json:"bands"`
		} `json:"recovery"`
	} `json:"storage"`
}

// LoadThresholds reads the recovery thresholds from a repository contract.
// Callers pass the contract path resolved by their owning runtime; a missing
// or malformed contract is an error rather than an implicit emergency policy.
func LoadThresholds(contractPath string) (Thresholds, error) {
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		return Thresholds{}, fmt.Errorf("read repository contract: %w", err)
	}
	var contract recoveryContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return Thresholds{}, fmt.Errorf("decode repository contract: %w", err)
	}
	thresholds := Thresholds{
		Warning:         contract.Storage.Recovery.Bands.Warning,
		High:            contract.Storage.Recovery.Bands.High,
		Critical:        contract.Storage.Recovery.Bands.Critical,
		FloorBytes:      contract.Storage.Recovery.FloorBytes,
		FastFillPercent: contract.Storage.Recovery.FastFillPercent,
	}
	if err := thresholds.Validate(); err != nil {
		return Thresholds{}, err
	}
	return thresholds, nil
}

const (
	BandNormal   Band = "normal"
	BandWarning  Band = "warning"
	BandHigh     Band = "high"
	BandCritical Band = "critical"
)

// Classify maps a measured percentage to the first threshold it has crossed.
// Thresholds are ordered warning <= high <= critical; invalid ordering fails
// closed to normal rather than manufacturing an escalation.
func Classify(usedPercent float64, availableBytes int64, thresholds Thresholds) Band {
	if thresholds.Warning < 0 || thresholds.Warning > thresholds.High || thresholds.High > thresholds.Critical || availableBytes < 0 {
		return BandNormal
	}
	if thresholds.FloorBytes > 0 && availableBytes < thresholds.FloorBytes {
		return BandCritical
	}
	switch {
	case usedPercent >= thresholds.Critical:
		return BandCritical
	case usedPercent >= thresholds.High:
		return BandHigh
	case usedPercent >= thresholds.Warning:
		return BandWarning
	default:
		return BandNormal
	}
}

// Validate returns a useful error for configuration loaders and tests.
func (t Thresholds) Validate() error {
	if t.Warning < 0 || t.Warning > t.High || t.High > t.Critical || t.Critical > 100 {
		return fmt.Errorf("pressure thresholds must satisfy 0 <= warning <= high <= critical <= 100")
	}
	if t.FloorBytes < 0 || t.FastFillPercent < 0 {
		return fmt.Errorf("pressure floor and fast-fill values cannot be negative")
	}
	return nil
}
