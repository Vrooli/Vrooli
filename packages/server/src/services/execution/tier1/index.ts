export * from "./coordination/index.js";
export * from "./intelligence/index.js";
export * from "./organization/index.js";
export * from "./state/index.js";
export * from "./tierOneCoordinator.js";
export * from "./tierOneFactory.js";

// MetacognitiveMonitor functionality now provided by emergent agents - see monitoring/README.md
// export { AgentDeploymentService } from "../cross-cutting/agents/agentDeploymentService.js";

export type {
    MetacognitiveReflection,
    StrategySelectionResult, SwarmAgent, SwarmConfig, SwarmDecision, SwarmInitParams, SwarmPerformance, SwarmState, SwarmTeam, TeamFormation
} from "@vrooli/shared";

