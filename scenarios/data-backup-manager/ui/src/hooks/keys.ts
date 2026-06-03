/**
 * Centralized React Query keys. Keeping every key here means mutations can
 * invalidate the exact queries they affect without each hook re-deriving key
 * shapes (the drift that makes "why didn't the list refresh?" bugs).
 */
export const queryKeys = {
  health: ["health"] as const,
  targets: (owner = "") => ["targets", owner] as const,
  target: (id: string) => ["target", id] as const,
  targetStatus: (owner = "") => ["targetStatus", owner] as const,
  destinations: ["destinations"] as const,
  destination: (id: string) => ["destination", id] as const,
  destinationUsage: (id: string) => ["destinationUsage", id] as const,
  plans: ["plans"] as const,
  plan: (id: string) => ["plan", id] as const,
  runs: (planId = "") => ["runs", planId] as const,
  run: (id: string) => ["run", id] as const,
  runStats: (planId = "") => ["runStats", planId] as const,
  restores: (targetId = "") => ["restores", targetId] as const,
  restore: (id: string) => ["restore", id] as const,
  audits: (targetId = "") => ["audits", targetId] as const,
  audit: (id: string) => ["audit", id] as const,
  targetSuggestions: ["targetSuggestions"] as const,
  destinationSuggestions: ["destinationSuggestions"] as const,
  coverageReport: ["coverageReport"] as const,
};
