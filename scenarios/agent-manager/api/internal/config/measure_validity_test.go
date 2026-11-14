package config

import "testing"

func TestLoadMeasureValidityConfigUsesDefaultsAndValidatesOverrides(t *testing.T) {
	config, err := LoadMeasureValidityConfig()
	if err != nil || config.MinSampleMeaningful != 5 || config.MaxFingerprintBucketShare != 0.90 {
		t.Fatalf("defaults = %+v, %v", config, err)
	}
	t.Setenv("AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL", "9")
	t.Setenv("AGENT_MANAGER_MEASURE_MAX_FINGERPRINT_BUCKET_SHARE", "0.75")
	config, err = LoadMeasureValidityConfig()
	if err != nil || config.MinSampleMeaningful != 9 || config.MaxFingerprintBucketShare != 0.75 {
		t.Fatalf("overrides = %+v, %v", config, err)
	}
	t.Setenv("AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL", "0")
	if _, err := LoadMeasureValidityConfig(); err == nil {
		t.Fatal("expected invalid minimum sample error")
	}
}
