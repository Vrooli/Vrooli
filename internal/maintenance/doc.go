// Package maintenance owns maintenance observations and bounded cleanup
// operations. Demand-based stopping is an explicit runtime-registry contract:
// only instances carrying the demand-managed supervision policy are eligible,
// and process identity is revalidated immediately before a signal.
package maintenance
