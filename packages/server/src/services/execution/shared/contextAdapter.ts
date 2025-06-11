/**
 * Context Adapter
 * 
 * Provides backward compatibility adapters to gradually migrate from the existing
 * tier-specific context implementations to the unified context system.
 * 
 * This allows for incremental migration without breaking existing code.
 */

import { type Logger } from "winston";
import {
    type CoordinationContext,
    type ProcessContext, 
    type ExecutionContext,
    type BaseContext,
} from "@vrooli/shared/dist/execution/types/context.js";
import {
    UnifiedContextManager,
    type CoordinationContextView,
    type ProcessContextView,
    type ExecutionContextView,
} from "./unifiedContext.js";
import { type ProcessRunContext } from "../tier2/context/contextManager.js";
import { ExecutionRunContext, type ExecutionRunContextConfig } from "../tier3/context/runContext.js";

/**
 * Adapters for converting between legacy context types and unified context
 */

export class ContextAdapter {
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Convert legacy CoordinationContext to UnifiedContextManager
     */
    coordinationToUnified(legacyContext: CoordinationContext): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            id: legacyContext.id,
            timestamp: legacyContext.timestamp,
            metadata: legacyContext.metadata,
            swarmId: legacyContext.swarmId,
            sharedMemory: legacyContext.sharedMemory.blackboard,
            blackboard: legacyContext.sharedMemory.blackboard,
            states: {
                coordination: legacyContext.coordinationState,
            },
            config: {
                tier: 1,
                isolationLevel: "scope",
                persistenceLevel: "permanent",
                securityLevel: "private",
                maxVariableSize: 10 * 1024 * 1024,
                maxScopeDepth: 5,
            },
        });
    }

    /**
     * Convert UnifiedContextManager to legacy CoordinationContext
     */
    unifiedToCoordination(unifiedContext: UnifiedContextManager): CoordinationContext {
        const view = unifiedContext.getCoordinationView();
        
        return {
            id: view.id,
            tier: 1,
            timestamp: view.timestamp,
            metadata: view.metadata,
            swarmId: view.swarmId,
            conversationId: view.metadata.sessionId,
            teams: view.metadata.tags.filter(tag => tag.startsWith("team:")),
            sharedMemory: {
                blackboard: view.blackboard,
                decisions: [],
                consensus: [],
                conflicts: [],
            },
            coordinationState: view.coordinationState,
        };
    }

    /**
     * Convert legacy ProcessContext to UnifiedContextManager
     */
    processToUnified(legacyContext: ProcessContext): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            id: legacyContext.id,
            timestamp: legacyContext.timestamp,
            metadata: legacyContext.metadata,
            runId: legacyContext.runId,
            variables: new Map(Object.entries(legacyContext.processMemory.variables)),
            states: {
                navigation: legacyContext.navigationState,
                orchestration: legacyContext.orchestrationState,
            },
            performance: {
                stepDurations: legacyContext.processMemory.performanceData.stepDurations,
                resourceUsage: legacyContext.processMemory.performanceData.resourceUsage,
                bottlenecks: legacyContext.processMemory.performanceData.bottlenecks,
                efficiencyScore: legacyContext.processMemory.performanceData.efficiencyScore,
                memoryUsage: 0,
                contextAccessCount: 0,
            },
            config: {
                tier: 2,
                isolationLevel: "variable",
                persistenceLevel: "checkpoint",
                securityLevel: "private",
                maxVariableSize: 1024 * 1024,
                maxScopeDepth: 10,
            },
        });
    }

    /**
     * Convert UnifiedContextManager to legacy ProcessContext
     */
    unifiedToProcess(unifiedContext: UnifiedContextManager): ProcessContext {
        const view = unifiedContext.getProcessView();
        
        return {
            id: view.id,
            tier: 2,
            timestamp: view.timestamp,
            metadata: view.metadata,
            runId: view.runId,
            routineId: view.routineId || "unknown",
            navigationState: view.navigationState,
            processMemory: {
                variables: Object.fromEntries(view.variables),
                checkpoints: [],
                optimizations: [],
                performanceData: {
                    stepDurations: view.performance.stepDurations,
                    resourceUsage: view.performance.resourceUsage,
                    bottlenecks: view.performance.bottlenecks,
                    efficiencyScore: view.performance.efficiencyScore,
                },
            },
            orchestrationState: view.orchestrationState,
        };
    }

    /**
     * Convert legacy ExecutionContext to UnifiedContextManager
     */
    executionToUnified(legacyContext: ExecutionContext): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            id: legacyContext.id,
            timestamp: legacyContext.timestamp,
            metadata: legacyContext.metadata,
            stepId: legacyContext.stepId,
            inputs: legacyContext.executionMemory.inputs,
            outputs: legacyContext.executionMemory.outputs,
            toolCalls: legacyContext.executionMemory.toolCalls.map(call => ({
                id: call.id,
                timestamp: call.timestamp,
                toolName: call.toolName,
                parameters: call.parameters,
                result: call.result,
                duration: call.duration,
                success: call.success !== false,
            })),
            states: {
                adaptation: legacyContext.adaptationState,
            },
            learningData: legacyContext.executionMemory.learningData ? 
                legacyContext.executionMemory.learningData.patterns?.map((pattern: any) => ({
                    id: pattern.id || `pattern-${Date.now()}`,
                    timestamp: new Date(),
                    pattern: pattern.description || pattern,
                    confidence: pattern.confidence || 0.5,
                    frequency: pattern.frequency || 1,
                    applicability: pattern.applicability || [],
                })) || [] : [],
            config: {
                tier: 3,
                isolationLevel: "full",
                persistenceLevel: "memory",
                securityLevel: "public",
                maxVariableSize: 512 * 1024,
                maxScopeDepth: 3,
            },
        });
    }

    /**
     * Convert UnifiedContextManager to legacy ExecutionContext
     */
    unifiedToExecution(unifiedContext: UnifiedContextManager): ExecutionContext {
        const view = unifiedContext.getExecutionView();
        
        return {
            id: view.id,
            tier: 3,
            timestamp: view.timestamp,
            metadata: view.metadata,
            stepId: view.stepId,
            strategyType: view.metadata.tags.find(tag => tag.startsWith("strategy:"))?.split(":")[1] || "unknown",
            executionMemory: {
                inputs: view.inputs,
                outputs: view.outputs,
                toolCalls: view.toolCalls.map(call => ({
                    id: call.id,
                    timestamp: call.timestamp,
                    toolName: call.toolName,
                    parameters: call.parameters,
                    result: call.result,
                    duration: call.duration,
                    success: call.success,
                })),
                strategyData: {},
                learningData: {
                    patterns: view.learningData.map(learning => ({
                        id: learning.id,
                        description: learning.pattern,
                        confidence: learning.confidence,
                        frequency: learning.frequency,
                        applicability: learning.applicability,
                    })),
                    adaptations: [],
                    optimizations: [],
                },
            },
            adaptationState: view.adaptationState,
        };
    }

    /**
     * Convert Tier 2 ProcessRunContext to UnifiedContextManager
     */
    processRunContextToUnified(processContext: ProcessRunContext): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            variables: new Map(Object.entries(processContext.variables)),
            scopes: processContext.scopes,
            blackboard: processContext.blackboard,
            config: {
                tier: 2,
                isolationLevel: "variable",
                persistenceLevel: "checkpoint",
                securityLevel: "private",
                maxVariableSize: 1024 * 1024,
                maxScopeDepth: 10,
            },
        });
    }

    /**
     * Convert UnifiedContextManager to Tier 2 ProcessRunContext
     */
    unifiedToProcessRunContext(unifiedContext: UnifiedContextManager): ProcessRunContext {
        const view = unifiedContext.getProcessView();
        
        return {
            variables: Object.fromEntries(view.variables),
            blackboard: Object.fromEntries(view.variables), // Simplified mapping
            scopes: [...view.scopes],
        };
    }

    /**
     * Convert Tier 3 ExecutionRunContext to UnifiedContextManager
     */
    executionRunContextToUnified(executionContext: ExecutionRunContext): UnifiedContextManager {
        return new UnifiedContextManager(this.logger, {
            id: `ctx-${executionContext.runId}`,
            stepId: executionContext.currentStepId,
            runId: executionContext.runId,
            metadata: {
                userId: executionContext.userData.id,
                sessionId: executionContext.runId, // Use runId as session for simplicity
                requestId: executionContext.runId,
                tags: [`routine:${executionContext.routineId}`],
            },
            inputs: executionContext.getAllMetadata(),
            outputs: {},
            config: {
                tier: 3,
                isolationLevel: "full",
                persistenceLevel: "memory",
                securityLevel: "public",
                maxVariableSize: 512 * 1024,
                maxScopeDepth: 3,
            },
        });
    }

    /**
     * Convert UnifiedContextManager to Tier 3 ExecutionRunContext
     */
    unifiedToExecutionRunContext(unifiedContext: UnifiedContextManager): ExecutionRunContext {
        const view = unifiedContext.getExecutionView();
        
        const config: ExecutionRunContextConfig = {
            runId: view.runId || view.id,
            routineId: view.metadata.tags.find(tag => tag.startsWith("routine:"))?.split(":")[1] || "unknown",
            routineName: "Generated from unified context",
            currentStepId: view.stepId,
            userData: {
                id: view.metadata.userId,
                name: "System User",
                languages: ["en"],
                preferences: {},
            },
            environment: process.env as Record<string, string>,
            metadata: view.inputs,
            logger: this.logger,
        };

        return new ExecutionRunContext(config);
    }

    /**
     * Create a migration helper that wraps existing context managers
     */
    createMigrationWrapper<T extends BaseContext>(
        legacyContext: T,
        tier: 1 | 2 | 3,
    ): {
        unified: UnifiedContextManager;
        getLegacy: () => T;
        sync: () => void;
    } {
        let unifiedContext: UnifiedContextManager;

        // Convert legacy to unified
        switch (tier) {
            case 1:
                unifiedContext = this.coordinationToUnified(legacyContext as CoordinationContext);
                break;
            case 2:
                unifiedContext = this.processToUnified(legacyContext as ProcessContext);
                break;
            case 3:
                unifiedContext = this.executionToUnified(legacyContext as ExecutionContext);
                break;
            default:
                throw new Error(`Invalid tier: ${tier}`);
        }

        return {
            unified: unifiedContext,
            getLegacy: () => {
                // Convert unified back to legacy when requested
                switch (tier) {
                    case 1:
                        return this.unifiedToCoordination(unifiedContext) as T;
                    case 2:
                        return this.unifiedToProcess(unifiedContext) as T;
                    case 3:
                        return this.unifiedToExecution(unifiedContext) as T;
                    default:
                        throw new Error(`Invalid tier: ${tier}`);
                }
            },
            sync: () => {
                // Update the unified context if the legacy context was modified externally
                // This is a simplified sync - in practice, you'd want more sophisticated change detection
                this.logger.debug("Context sync called - implement change detection as needed");
            },
        };
    }
}

/**
 * Global context adapter instance
 */
let globalAdapter: ContextAdapter | null = null;

export function getContextAdapter(logger: Logger): ContextAdapter {
    if (!globalAdapter) {
        globalAdapter = new ContextAdapter(logger);
    }
    return globalAdapter;
}

/**
 * Helper functions for common context operations
 */

/**
 * Create a unified context from any legacy context
 */
export function createUnifiedFromLegacy(
    logger: Logger,
    legacyContext: CoordinationContext | ProcessContext | ExecutionContext,
): UnifiedContextManager {
    const adapter = getContextAdapter(logger);
    
    if ("swarmId" in legacyContext) {
        return adapter.coordinationToUnified(legacyContext as CoordinationContext);
    } else if ("runId" in legacyContext) {
        return adapter.processToUnified(legacyContext as ProcessContext);
    } else if ("stepId" in legacyContext) {
        return adapter.executionToUnified(legacyContext as ExecutionContext);
    } else {
        throw new Error("Unknown context type");
    }
}

/**
 * Convert unified context back to legacy format
 */
export function createLegacyFromUnified<T extends BaseContext>(
    logger: Logger,
    unifiedContext: UnifiedContextManager,
    targetTier: 1 | 2 | 3,
): T {
    const adapter = getContextAdapter(logger);
    
    switch (targetTier) {
        case 1:
            return adapter.unifiedToCoordination(unifiedContext) as T;
        case 2:
            return adapter.unifiedToProcess(unifiedContext) as T;
        case 3:
            return adapter.unifiedToExecution(unifiedContext) as T;
        default:
            throw new Error(`Invalid target tier: ${targetTier}`);
    }
}