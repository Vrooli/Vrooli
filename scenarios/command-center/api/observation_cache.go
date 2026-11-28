package main

import "time"

// commitQualifiedObservation keeps the response aligned with the cache's
// last-good ordering. A slower request may finish after a newer producer
// observation has already been qualified and committed by another request.
func commitQualifiedObservation(cache *Cache, key string, source MetricEntry, candidate MetricEntry, observed Envelope, ttl time.Duration) (MetricEntry, Envelope) {
	if cache.Put(key, observed, ttl) {
		return candidate, observed
	}

	latest, _, ok := cache.Get(key)
	if !ok || latest.ObservationAt == nil || observed.ObservationAt == nil || !latest.ObservationAt.After(*observed.ObservationAt) {
		return candidate, observed
	}

	current, _ := qualifyMetricObservation(source, latest, nil)
	return current, latest
}
