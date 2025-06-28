export * from "./coordination/index.js";
export * from "./intelligence/index.js";
export * from "./organization/index.js";
// State management moved to shared/SwarmContextManager - state/ directory removed
// TierOneCoordinator removed - use SwarmCoordinator from coordination/ instead

// Strategic intelligence and monitoring now provided by emergent agents
// See docs/architecture/execution/emergent-capabilities/

export type {
    MetacognitiveReflection,
    StrategySelectionResult, SwarmAgent, SwarmConfig, SwarmDecision, SwarmInitParams, SwarmPerformance, SwarmTeam, TeamFormation,
} from "@vrooli/shared";

