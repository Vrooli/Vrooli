/**
 * Tier 1: Coordination Intelligence
 * 
 * This tier provides strategic coordination and metacognitive capabilities
 * for autonomous swarm operations. It implements the OODA loop (Observe,
 * Orient, Decide, Act) for high-level decision making and adaptation.
 */
export { SwarmStateMachine } from "./coordination/swarmStateMachine.js";
export { MetacognitiveMonitorAdapter as MetacognitiveMonitor } from "../monitoring/adapters/MetacognitiveMonitorAdapter.js";
export { StrategyEngine } from "./intelligence/strategyEngine.js";
export { ResourceManager } from "./organization/resourceManager.js";
export { TeamManager } from "./organization/teamManager.js";
export { RedisSwarmStateStore } from "./state/redisSwarmStateStore.js";
export { InMemorySwarmStateStore, type ISwarmStateStore } from "./state/swarmStateStore.js";
export { SwarmStateStoreFactory } from "./state/swarmStateStoreFactory.js";
export { TierOneCoordinator } from "./tierOneCoordinator.js";
export { TierOneFactory } from "./tierOneFactory.js";

// Re-export types
export type {
    MetacognitiveReflection,
    StrategySelectionResult, SwarmAgent, SwarmConfig, SwarmDecision, SwarmInitParams, SwarmPerformance, SwarmState, SwarmTeam, TeamFormation
} from "@vrooli/shared";

