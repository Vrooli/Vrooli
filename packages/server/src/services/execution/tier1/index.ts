/**
 * Tier 1: Coordination Intelligence
 * 
 * This tier provides strategic coordination and metacognitive capabilities
 * for autonomous swarm operations. It implements the OODA loop (Observe,
 * Orient, Decide, Act) for high-level decision making and adaptation.
 */

// Main coordinator
export { TierOneCoordinator } from "./TierOneCoordinator.js";

// Core coordination
export { SwarmStateMachine } from "./coordination/swarmStateMachine.js";

// Organization and resource management
export { TeamManager } from "./organization/teamManager.js";
export { ResourceManager } from "./organization/resourceManager.js";

// Intelligence components
export { StrategyEngine } from "./intelligence/strategyEngine.js";
export { MetacognitiveMonitor } from "./intelligence/metacognitiveMonitor.js";

// State management
export { type ISwarmStateStore, InMemorySwarmStateStore } from "./state/swarmStateStore.js";
export { RedisSwarmStateStore } from "./state/redisSwarmStateStore.js";
export { SwarmStateStoreFactory } from "./state/swarmStateStoreFactory.js";

// Re-export types
export type {
    SwarmInitParams,
    SwarmState,
    SwarmConfig,
    SwarmTeam,
    SwarmAgent,
    TeamFormation,
    MetacognitiveReflection,
    StrategySelectionResult,
    SwarmDecision,
    SwarmPerformance,
} from "@vrooli/shared";
