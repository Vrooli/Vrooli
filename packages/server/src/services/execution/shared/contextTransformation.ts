import { type Logger } from "winston";
import { 
    type RunContext,
    type ExecutionContext,
    type ProcessContext,
    type CoordinationContext,
    type BaseContext,
    type ContextScope,
} from "@vrooli/shared";
import { type ProcessRunContext } from "../tier2/context/contextManager.js";
import { type ExecutionRunContext } from "../tier3/context/runContext.js";

/**
 * Context transformation utilities for converting between tier-specific contexts
 * 
 * These utilities enable proper context flow between tiers while maintaining
 * data integrity and respecting the three-layer architecture defined in docs.
 */

export interface ContextTransformationOptions {
    preserveVariables?: boolean;
    includeMetadata?: boolean;
    filterSensitiveData?: boolean;
    maxVariableSize?: number;
}

/**
 * Context Transformer - Converts between different context types
 */
export class ContextTransformer {
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Transforms Tier 2 ProcessRunContext to shared RunContext
     */
    processToShared(
        processContext: ProcessRunContext,
        runId: string,
        routineManifest: any,
        options: ContextTransformationOptions = {},
    ): RunContext {
        const variables = new Map<string, any>();
        
        // Transform scoped variables to flat map
        for (const scope of processContext.scopes) {
            for (const [key, value] of Object.entries(scope.variables)) {
                // Add scope prefix to avoid conflicts
                const scopedKey = scope.id === "global" ? key : `${scope.id}.${key}`;
                variables.set(scopedKey, {
                    key: scopedKey,
                    value,
                    source: this.mapScopeToSource(scope.id),
                    timestamp: new Date(),
                    sensitivity: "internal" as const,
                    mutable: true,
                });
            }
        }

        // Add blackboard variables
        for (const [key, value] of Object.entries(processContext.blackboard)) {
            variables.set(`blackboard.${key}`, {
                key: `blackboard.${key}`,
                value,
                source: "system" as const,
                timestamp: new Date(),
                sensitivity: "internal" as const,
                mutable: true,
            });
        }

        const sharedContext: RunContext = {
            runId,
            routineManifest,
            permissions: [], // TODO: Map from process context if available
            resourceLimits: {
                maxCredits: 1000, // Default values - should come from config
                maxDurationMs: 300000,
                maxConcurrentBranches: 5,
                maxMemoryMB: 256,
                maxToolCalls: 50,
            },
            variables,
            intermediate: new Map(Object.entries(processContext.variables)),
            exports: [], // TODO: Extract from process context
            sensitivityMap: new Map(), // TODO: Build from variable sensitivity
            
            // Implement required methods
            createChild: (overrides) => {
                // This would need to be implemented based on requirements
                throw new Error("createChild not implemented in transformed context");
            },
            inheritFromParent: () => {
                throw new Error("inheritFromParent not implemented in transformed context");
            },
            resolveVariableConflicts: () => {
                throw new Error("resolveVariableConflicts not implemented in transformed context");
            },
            markForExport: () => {
                throw new Error("markForExport not implemented in transformed context");
            },
        };

        this.logger.debug("[ContextTransformer] Transformed process to shared context", {
            runId,
            variableCount: variables.size,
            scopeCount: processContext.scopes.length,
        });

        return sharedContext;
    }

    /**
     * Transforms shared RunContext back to Tier 2 ProcessRunContext
     */
    sharedToProcess(
        sharedContext: RunContext,
        options: ContextTransformationOptions = {},
    ): ProcessRunContext {
        const scopes: ContextScope[] = [];
        const variables: Record<string, unknown> = {};
        const blackboard: Record<string, unknown> = {};

        // Extract global scope
        const globalScope: ContextScope = {
            id: "global",
            name: "Global Scope", 
            variables: {},
        };

        // Process shared variables back to scoped structure
        for (const [key, contextVar] of sharedContext.variables) {
            if (key.startsWith("blackboard.")) {
                const blackboardKey = key.replace("blackboard.", "");
                blackboard[blackboardKey] = contextVar.value;
            } else if (key.includes(".")) {
                const [scopeId, ...keyParts] = key.split(".");
                const actualKey = keyParts.join(".");
                
                // Find or create scope
                let scope = scopes.find(s => s.id === scopeId);
                if (!scope) {
                    scope = {
                        id: scopeId,
                        name: `Scope ${scopeId}`,
                        variables: {},
                    };
                    scopes.push(scope);
                }
                scope.variables[actualKey] = contextVar.value;
            } else {
                globalScope.variables[key] = contextVar.value;
                variables[key] = contextVar.value;
            }
        }

        // Add intermediate variables to variables
        for (const [key, value] of sharedContext.intermediate) {
            variables[key] = value;
            globalScope.variables[key] = value;
        }

        // Ensure global scope is first
        scopes.unshift(globalScope);

        const processContext: ProcessRunContext = {
            variables,
            blackboard,
            scopes,
        };

        this.logger.debug("[ContextTransformer] Transformed shared to process context", {
            runId: sharedContext.runId,
            variableCount: Object.keys(variables).length,
            blackboardSize: Object.keys(blackboard).length,
            scopeCount: scopes.length,
        });

        return processContext;
    }

    /**
     * Creates ExecutionContext for Tier 3 from ExecutionRunContext
     */
    executionRunToExecutionContext(
        runContext: ExecutionRunContext,
        stepId: string,
        strategyType: string = "conversational",
    ): ExecutionContext {
        const executionContext: ExecutionContext = {
            tier: 3,
            stepId,
            strategyType,
            executionMemory: {
                inputs: {},
                outputs: {},
                toolCalls: [],
                strategyData: {},
                learningData: {
                    patterns: [],
                    feedback: [],
                    adaptations: [],
                },
            },
            adaptationState: {
                mode: "stable" as const,
                currentStrategy: strategyType,
                alternativeStrategies: ["conversational", "reasoning", "deterministic"],
                confidenceThreshold: 0.7,
                adaptationRate: 0.1,
            },
            id: runContext.runId,
            timestamp: new Date(),
            metadata: {
                userId: runContext.userData.id,
                sessionId: runContext.runId,
                requestId: runContext.runId,
                tags: [],
            },
        };

        this.logger.debug("[ContextTransformer] Created execution context", {
            runId: runContext.runId,
            stepId,
            strategyType,
        });

        return executionContext;
    }

    /**
     * Creates ProcessContext for Tier 2 from ProcessRunContext
     */
    processRunToProcessContext(
        processContext: ProcessRunContext,
        runId: string,
        routineId: string,
    ): ProcessContext {
        const context: ProcessContext = {
            tier: 2,
            runId,
            routineId,
            navigationState: {
                currentLocation: "start",
                locationStack: [],
                visitedLocations: new Set(),
                branchStates: {},
            },
            processMemory: {
                variables: processContext.variables,
                checkpoints: [],
                optimizations: [],
                performanceData: {
                    stepDurations: {},
                    resourceUsage: {},
                    bottlenecks: [],
                    efficiencyScore: 1.0,
                },
            },
            orchestrationState: {
                phase: "initializing" as const,
                activeSteps: [],
                pendingSteps: [],
                completedSteps: [],
                failedSteps: [],
            },
            id: runId,
            timestamp: new Date(),
            metadata: {
                userId: "", // TODO: Extract from context
                sessionId: runId,
                requestId: runId,
                tags: [],
            },
        };

        this.logger.debug("[ContextTransformer] Created process context", {
            runId,
            routineId,
            variableCount: Object.keys(processContext.variables).length,
        });

        return context;
    }

    /**
     * Validates context transformation
     */
    validateTransformation(
        original: any,
        transformed: any,
        transformationType: string,
    ): boolean {
        try {
            if (!original || !transformed) {
                this.logger.warn("[ContextTransformer] Null context in validation", {
                    transformationType,
                    hasOriginal: !!original,
                    hasTransformed: !!transformed,
                });
                return false;
            }

            // Basic structural validation
            if (typeof transformed !== "object") {
                this.logger.warn("[ContextTransformer] Transformed context is not an object", {
                    transformationType,
                    transformedType: typeof transformed,
                });
                return false;
            }

            this.logger.debug("[ContextTransformer] Context transformation validated", {
                transformationType,
                originalKeys: Object.keys(original).length,
                transformedKeys: Object.keys(transformed).length,
            });

            return true;

        } catch (error) {
            this.logger.error("[ContextTransformer] Context validation failed", {
                transformationType,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Helper methods
     */
    private mapScopeToSource(scopeId: string): "parent" | "step" | "user" | "system" {
        if (scopeId === "global") return "system";
        if (scopeId.startsWith("step-")) return "step";
        if (scopeId.startsWith("branch-")) return "parent";
        return "user";
    }

    /**
     * Filters sensitive data from context
     */
    private filterSensitiveData(
        data: Record<string, unknown>,
        options: ContextTransformationOptions,
    ): Record<string, unknown> {
        if (!options.filterSensitiveData) return data;

        const filtered: Record<string, unknown> = {};
        const sensitiveKeys = ["password", "token", "secret", "key", "credential"];

        for (const [key, value] of Object.entries(data)) {
            const isSensitive = sensitiveKeys.some(sensitiveKey => 
                key.toLowerCase().includes(sensitiveKey),
            );

            if (isSensitive) {
                filtered[key] = "[FILTERED]";
            } else {
                filtered[key] = value;
            }
        }

        return filtered;
    }

    /**
     * Validates variable size limits
     */
    private validateVariableSize(
        value: unknown,
        maxSize: number = 1024 * 1024, // 1MB default
    ): boolean {
        try {
            const serialized = JSON.stringify(value);
            return serialized.length <= maxSize;
        } catch {
            return false; // Non-serializable values are rejected
        }
    }
}

/**
 * Factory for creating context transformers
 */
export class ContextTransformerFactory {
    static create(logger: Logger): ContextTransformer {
        return new ContextTransformer(logger);
    }
}