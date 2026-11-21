package config

import (
	"fmt"
	"os"
	"strconv"
)

type MeasureValidityConfig struct {
	MinSampleMeaningful       int64
	MaxFingerprintBucketShare float64
}

// LoadMeasureValidityConfig keeps the analytical-honesty thresholds in the
// scenario control surface instead of embedding per-measure magic numbers.
func LoadMeasureValidityConfig() (MeasureValidityConfig, error) {
	config := MeasureValidityConfig{MinSampleMeaningful: 5, MaxFingerprintBucketShare: 0.90}
	if raw, configured := os.LookupEnv("AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL"); configured {
		if raw == "" {
			return config, fmt.Errorf("AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL must not be empty when configured")
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return config, fmt.Errorf("AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL must be a positive integer")
		}
		config.MinSampleMeaningful = value
	}
	if raw, configured := os.LookupEnv("AGENT_MANAGER_MEASURE_MAX_FINGERPRINT_BUCKET_SHARE"); configured {
		if raw == "" {
			return config, fmt.Errorf("AGENT_MANAGER_MEASURE_MAX_FINGERPRINT_BUCKET_SHARE must not be empty when configured")
		}
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || value <= 0 || value > 1 {
			return config, fmt.Errorf("AGENT_MANAGER_MEASURE_MAX_FINGERPRINT_BUCKET_SHARE must be in (0,1]")
		}
		config.MaxFingerprintBucketShare = value
	}
	return config, nil
}
