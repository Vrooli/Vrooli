// Package maintenance owns maintenance observations and bounded cleanup
// operations. Demand-based stopping is an explicit runtime-registry contract:
// only instances carrying the demand-managed supervision policy are eligible,
// and process identity is revalidated immediately before a signal.
// Capacity maintenance distinguishes plans from actual actuator outcomes:
// advisory/off recommendations and skips are observations, applied callbacks
// are effects, and failures remain failures. A planned action is never evidence
// that a resource was unloaded or upshifted.
package maintenance
