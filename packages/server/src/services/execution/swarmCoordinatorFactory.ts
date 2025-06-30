/**
 * SwarmCoordinator Factory - Singleton Management for Bull Queue Integration
 * 
 * Provides a singleton SwarmCoordinator instance per worker process to eliminate
 * the overhead of creating full SwarmExecutionService instances per swarm.
 * 
 * This replaces the heavy SwarmExecutionService + NewSwarmStateMachineAdapter pattern
 * with direct SwarmCoordinator usage that implements ManagedTaskStateMachine.
 */

import { type Logger } from "winston";
import { logger as defaultLogger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { getUnifiedEventSystem } from "../events/index.js";
import { SwarmContextManager } from "./shared/SwarmContextManager.js";
import { SwarmCoordinator } from "./tier1/coordination/index.js";
import { createConversationBridge } from "./tier1/intelligence/conversationBridge.js";
import { TierTwoOrchestrator } from "./tier2/index.js";

/**
 * Singleton SwarmCoordinator instance per worker process
 */
let swarmCoordinator: SwarmCoordinator | null = null;

/**
 * Get or create the singleton SwarmCoordinator for this worker process.
 * 
 * This approach eliminates the overhead of creating SwarmExecutionService + all tiers
 * for every swarm execution. Instead, we have one coordinator that can handle
 * multiple concurrent swarms efficiently.
 * 
 * @param logger Optional logger instance (defaults to global logger)
 * @returns Singleton SwarmCoordinator instance
 */
export function getSwarmCoordinator(logger: Logger = defaultLogger): SwarmCoordinator {
    if (!swarmCoordinator) {
        logger.info("[SwarmCoordinatorFactory] Creating singleton SwarmCoordinator");

        // Initialize core dependencies
        const contextManager = new SwarmContextManager(
            CacheService.get().raw() as any, // Redis client
            logger,
        );

        const conversationBridge = createConversationBridge(logger);

        // Create lightweight Tier 2 orchestrator (Tier 3 created on demand)
        const tierTwo = new TierTwoOrchestrator(
            logger,
            getUnifiedEventSystem(),
            null, // TierThreeExecutor created lazily when needed
            contextManager,
        );

        // Create the singleton coordinator
        swarmCoordinator = new SwarmCoordinator(
            logger,
            contextManager,
            conversationBridge,
            tierTwo,
        );

        logger.info("[SwarmCoordinatorFactory] Singleton SwarmCoordinator created successfully");
    }

    return swarmCoordinator;
}

/**
 * Reset the singleton instance (primarily for testing)
 * 
 * @internal
 */
export function resetSwarmCoordinator(): void {
    swarmCoordinator = null;
}

/**
 * Check if singleton coordinator is initialized
 * 
 * @returns True if coordinator exists
 */
export function isSwarmCoordinatorInitialized(): boolean {
    return swarmCoordinator !== null;
}