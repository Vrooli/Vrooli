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
	MinClassifiedShare        float64
}

func DefaultValidityConfig() ValidityConfig {
	return ValidityConfig{MinSampleMeaningful: 5, MaxFingerprintBucketShare: 0.90, MinClassifiedShare: 0.90}
}

func (c ValidityConfig) normalized() ValidityConfig {
	defaults := DefaultValidityConfig()
	if c.MinSampleMeaningful <= 0 {
		c.MinSampleMeaningful = defaults.MinSampleMeaningful
	}
	if c.MaxFingerprintBucketShare <= 0 || c.MaxFingerprintBucketShare > 1 {
		c.MaxFingerprintBucketShare = defaults.MaxFingerprintBucketShare
	}
	if c.MinClassifiedShare <= 0 || c.MinClassifiedShare > 1 {
		c.MinClassifiedShare = defaults.MinClassifiedShare
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
	ClassifiedBase           int64   `json:"classifiedBase"`
	UnclassifiedCount        int64   `json:"unclassifiedCount"`
	UnclassifiedShare        float64 `json:"unclassifiedShare"`
	MinimumClassifiedShare   float64 `json:"minimumClassifiedShare"`
}

func assessValidity(sampleSize, largestFingerprintBucket int64, config ValidityConfig) Validity {
	config = config.normalized()
	validity := Validity{Availability: availability.New(availability.Available, ""), SampleSize: sampleSize, LargestFingerprintBucket: largestFingerprintBucket, ClassifiedBase: sampleSize, MinimumClassifiedShare: config.MinClassifiedShare}
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

func withDenominatorValidity(validity Validity, total, classified, unclassified int64, config ValidityConfig) Validity {
	config = config.normalized()
	validity.ClassifiedBase = classified
	validity.UnclassifiedCount = unclassified
	if total > 0 {
		validity.UnclassifiedShare = float64(unclassified) / float64(total)
	}
	validity.MinimumClassifiedShare = config.MinClassifiedShare
	if classified <= 0 {
		validity.Availability = availability.New(availability.Unavailable, "no classified invocation evidence is available")
		return validity
	}
	if total > 0 && float64(classified)/float64(total) < config.MinClassifiedShare {
		validity.Availability = availability.New(availability.Unreliable, fmt.Sprintf("classified share %.1f%% is below the minimum %.1f%%", float64(classified)/float64(total)*100, config.MinClassifiedShare*100))
	}
	return validity
}
