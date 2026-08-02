package measures

import (
	"fmt"

	"agent-manager/internal/availability"
)

// ValidityConfig is the product-owned honesty threshold shared by every
// analytical measure. It deliberately lives at the measure boundary rather
// than being selected by individual rates.
type ValidityConfig struct {
	MinSampleMeaningful       int64
	MaxFingerprintBucketShare float64
}

func DefaultValidityConfig() ValidityConfig {
	return ValidityConfig{MinSampleMeaningful: 5, MaxFingerprintBucketShare: 0.90}
}

func (c ValidityConfig) normalized() ValidityConfig {
	defaults := DefaultValidityConfig()
	if c.MinSampleMeaningful <= 0 {
		c.MinSampleMeaningful = defaults.MinSampleMeaningful
	}
	if c.MaxFingerprintBucketShare <= 0 || c.MaxFingerprintBucketShare > 1 {
		c.MaxFingerprintBucketShare = defaults.MaxFingerprintBucketShare
	}
	return c
}

// Validity is returned with every measure response. A result is unreliable
// when its population is too small or a classifier fingerprint dominates the
// population enough to make a rate an artifact of one repeated observation.
type Validity struct {
	availability.Availability
	SampleSize               int64   `json:"sampleSize"`
	LargestFingerprintBucket int64   `json:"largestFingerprintBucket"`
	LargestFingerprintShare  float64 `json:"largestFingerprintShare"`
}

func assessValidity(sampleSize, largestFingerprintBucket int64, config ValidityConfig) Validity {
	config = config.normalized()
	validity := Validity{Availability: availability.New(availability.Available, ""), SampleSize: sampleSize, LargestFingerprintBucket: largestFingerprintBucket}
	if sampleSize > 0 && largestFingerprintBucket > 0 {
		validity.LargestFingerprintShare = float64(largestFingerprintBucket) / float64(sampleSize)
	}
	if sampleSize < config.MinSampleMeaningful {
		validity.Availability = availability.New(availability.Unreliable, fmt.Sprintf("sample size %d is below the minimum meaningful sample of %d", sampleSize, config.MinSampleMeaningful))
		return validity
	}
	if validity.LargestFingerprintShare > config.MaxFingerprintBucketShare {
		validity.Availability = availability.New(availability.Unreliable, fmt.Sprintf("largest fingerprint bucket holds %.1f%% of the sample, above the %.1f%% threshold", validity.LargestFingerprintShare*100, config.MaxFingerprintBucketShare*100))
	}
	return validity
}
