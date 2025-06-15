import { type Logger } from "winston";
import {
    type ContextScope,
    deepClone,
} from "@vrooli/shared";
import { type IRunStateStore } from "../state/runStateStore.js";
import { ContextValidator, type ValidationResult } from "../../shared/contextValidator.js";
import { GenericStore } from "../../shared/GenericStore.js";
import { CacheService } from "../../../../redisConn.js";

/**
 * Process-specific RunContext interface for Tier 2
 * Extends the shared RunContext with process-specific features
 */
export interface ProcessRunContext {
    variables: Record<string, unknown>;
    blackboard: Record<string, unknown>;
    scopes: ContextScope[];
}

/**
 * Context initialization parameters
 */
export interface ContextInitParams {
    variables?: Record<string, unknown>;
    blackboard?: Record<string, unknown>;
    scopes?: ContextScope[];
}

/**
 * Variable access result
 */
export interface VariableAccess {
    value: unknown;
    scope: string;
    found: boolean;
}

/**
 * ContextManager - Manages variable scoping and context isolation
 * 
 * This component manages the hierarchical context system that enables:
 * - Variable scoping (global, routine, branch, step)
 * - Context isolation between parallel branches
 * - Variable inheritance and shadowing
 * - Isolated blackboard instances per branch
 * - Context serialization for checkpoints
 * 
 * The context system is critical for maintaining proper state isolation
 * in complex workflows with parallel execution and nested subroutines.
 * Each branch gets its own deep-cloned blackboard to prevent race conditions
 * and ensure data sharing only happens through explicit outputs.
 */
export class ContextManager {
    private readonly stateStore: IRunStateStore;
    private readonly logger: Logger;
    private readonly validator: ContextValidator;
    private readonly contextCache: GenericStore<ProcessRunContext>;

    constructor(stateStore: IRunStateStore, logger: Logger) {
        this.stateStore = stateStore;
        this.logger = logger;
        this.validator = new ContextValidator(logger);
        // Initialize contextCache as null, will be set up asynchronously
        this.contextCache = null as any; // Will be initialized in initialize()
    }
    
    async initialize(): Promise<void> {
        const redis = await CacheService.get().raw();
        this.contextCache = new GenericStore<ProcessRunContext>(
            this.logger,
            redis as any,
            {
                keyPrefix: "tier2.context",
                defaultTTL: 3600, // 1 hour
                publishEvents: true,
                eventChannelPrefix: "context"
            }
        );
    }

    /**
     * Creates a new context
     */
    async createContext(params: ContextInitParams = {}): Promise<ProcessRunContext> {
        const context: ProcessRunContext = {
            variables: params.variables || {},
            blackboard: params.blackboard || {},
            scopes: params.scopes || [{
                id: "global",
                name: "Global Scope",
                variables: {},
            }],
        };

        this.logger.debug("[ContextManager] Created new context", {
            scopeCount: context.scopes.length,
        });

        return context;
    }

    /**
     * Clones a context for branching with proper isolation
     */
    async cloneContext(context: ProcessRunContext, branchId: string): Promise<ProcessRunContext> {
        // Deep clone the context including blackboard for complete isolation
        const cloned: ProcessRunContext = {
            variables: { ...context.variables },
            blackboard: deepClone(context.blackboard), // Deep clone for isolated branch context
            scopes: context.scopes.map(scope => ({
                ...scope,
                variables: { ...scope.variables },
            })),
        };

        // Add branch scope
        cloned.scopes.push({
            id: `branch-${branchId}`,
            name: `Branch ${branchId}`,
            parentId: "global",
            variables: {},
        });

        this.logger.debug("[ContextManager] Cloned context for branch", {
            branchId,
            scopeCount: cloned.scopes.length,
        });

        return cloned;
    }

    /**
     * Merges contexts from parallel branches
     */
    async mergeContexts(
        parent: ProcessRunContext,
        branches: ProcessRunContext[],
        strategy: "first" | "last" | "merge" = "merge",
    ): Promise<ProcessRunContext> {
        const merged = await this.cloneContext(parent, "merged");

        if (strategy === "first" && branches.length > 0) {
            // Use first branch context
            return branches[0];
        }

        if (strategy === "last" && branches.length > 0) {
            // Use last branch context
            return branches[branches.length - 1];
        }

        // Merge strategy - combine all branch outputs
        const branchOutputs: Record<string, unknown[]> = {};

        for (const branch of branches) {
            // Find branch scope
            const branchScope = branch.scopes.find(s => s.id.startsWith("branch-"));
            if (!branchScope) continue;

            // Collect variables from branch scope
            for (const [key, value] of Object.entries(branchScope.variables)) {
                if (!branchOutputs[key]) {
                    branchOutputs[key] = [];
                }
                branchOutputs[key].push(value);
            }
        }

        // Add merged variables to parent context
        for (const [key, values] of Object.entries(branchOutputs)) {
            if (values.length === 1) {
                merged.variables[key] = values[0];
            } else {
                // Multiple values - store as array
                merged.variables[`${key}_branches`] = values;
            }
        }

        this.logger.debug("[ContextManager] Merged contexts", {
            branchCount: branches.length,
            strategy,
            mergedVars: Object.keys(branchOutputs).length,
        });

        return merged;
    }

    /**
     * Gets a variable value with scope resolution
     */
    async getVariable(
        context: ProcessRunContext,
        name: string,
        scopeId?: string,
    ): Promise<VariableAccess> {
        // Check specific scope if provided
        if (scopeId) {
            const scope = context.scopes.find(s => s.id === scopeId);
            if (scope && name in scope.variables) {
                return {
                    value: scope.variables[name],
                    scope: scopeId,
                    found: true,
                };
            }
        }

        // Check scopes in reverse order (most specific to least)
        for (let i = context.scopes.length - 1; i >= 0; i--) {
            const scope = context.scopes[i];
            if (name in scope.variables) {
                return {
                    value: scope.variables[name],
                    scope: scope.id,
                    found: true,
                };
            }
        }

        // Check global variables
        if (name in context.variables) {
            return {
                value: context.variables[name],
                scope: "global",
                found: true,
            };
        }

        return {
            value: undefined,
            scope: "",
            found: false,
        };
    }

    /**
     * Sets a variable in the appropriate scope
     */
    async setVariable(
        context: ProcessRunContext,
        name: string,
        value: unknown,
        scopeId?: string,
    ): Promise<void> {
        if (scopeId) {
            // Set in specific scope
            const scope = context.scopes.find(s => s.id === scopeId);
            if (scope) {
                scope.variables[name] = value;
                return;
            }
        }

        // Set in most specific scope (last in array)
        if (context.scopes.length > 0) {
            const currentScope = context.scopes[context.scopes.length - 1];
            currentScope.variables[name] = value;
        } else {
            // Fallback to global
            context.variables[name] = value;
        }
    }

    /**
     * Updates multiple variables
     */
    async updateVariables(
        runId: string,
        updates: Record<string, unknown>,
        scopeId?: string,
    ): Promise<void> {
        const context = await this.getContext(runId);

        for (const [name, value] of Object.entries(updates)) {
            await this.setVariable(context, name, value, scopeId);
        }

        await this.updateContext(runId, context);

        this.logger.debug("[ContextManager] Updated variables", {
            runId,
            count: Object.keys(updates).length,
            scopeId,
        });
    }

    /**
     * Pushes a new scope onto the stack
     */
    async pushScope(
        context: ProcessRunContext,
        scope: ContextScope,
    ): Promise<void> {
        context.scopes.push(scope);

        this.logger.debug("[ContextManager] Pushed scope", {
            scopeId: scope.id,
            scopeName: scope.name,
            depth: context.scopes.length,
        });
    }

    /**
     * Pops a scope from the stack
     */
    async popScope(context: ProcessRunContext): Promise<ContextScope | undefined> {
        if (context.scopes.length <= 1) {
            // Don't pop the global scope
            return undefined;
        }

        const popped = context.scopes.pop();

        this.logger.debug("[ContextManager] Popped scope", {
            scopeId: popped?.id,
            remainingDepth: context.scopes.length,
        });

        return popped;
    }

    /**
     * Gets blackboard value
     */
    async getBlackboardValue(
        context: ProcessRunContext,
        key: string,
    ): Promise<unknown> {
        return context.blackboard[key];
    }

    /**
     * Sets blackboard value
     */
    async setBlackboardValue(
        context: ProcessRunContext,
        key: string,
        value: unknown,
    ): Promise<void> {
        context.blackboard[key] = value;
    }

    /**
     * Gets context from cache or state store
     */
    async getContext(runId: string): Promise<ProcessRunContext> {
        // Check cache first
        const cachedResult = await this.contextCache.get(runId);
        if (cachedResult.success && cachedResult.data) {
            return cachedResult.data;
        }

        // Load from state store
        const context = await this.stateStore.getContext(runId);
        await this.contextCache.set(runId, context);

        return context;
    }

    /**
     * Updates context in cache and state store
     */
    async updateContext(runId: string, context: ProcessRunContext): Promise<void> {
        // Validate context before storing
        const validation = this.validator.validateProcessRunContext(context);
        if (!validation.valid) {
            const criticalErrors = validation.errors.filter(e => e.severity === "critical" || e.severity === "high");
            if (criticalErrors.length > 0) {
                this.logger.error("[ContextManager] Context validation failed", {
                    runId,
                    errors: criticalErrors,
                });
                throw new Error(`Context validation failed: ${criticalErrors[0].message}`);
            }
        }

        if (validation.warnings.length > 0) {
            this.logger.warn("[ContextManager] Context validation warnings", {
                runId,
                warnings: validation.warnings,
            });
        }

        // Update cache
        await this.contextCache.set(runId, context);

        // Persist to state store
        await this.stateStore.updateContext(runId, context);
    }

    /**
     * Creates a step execution scope
     */
    async createStepScope(
        context: ProcessRunContext,
        stepId: string,
        inputs: Record<string, unknown>,
    ): Promise<ContextScope> {
        const scope: ContextScope = {
            id: `step-${stepId}`,
            name: `Step ${stepId}`,
            parentId: context.scopes[context.scopes.length - 1]?.id || "global",
            variables: inputs,
        };

        await this.pushScope(context, scope);

        return scope;
    }

    /**
     * Serializes context for checkpointing
     */
    async serializeContext(context: ProcessRunContext): Promise<string> {
        return JSON.stringify(context, null, 2);
    }

    /**
     * Deserializes context from checkpoint
     */
    async deserializeContext(data: string): Promise<ProcessRunContext> {
        return JSON.parse(data);
    }

    /**
     * Clears context cache for a run
     */
    async clearCache(runId: string): Promise<void> {
        await this.contextCache.delete(runId);
    }

    /**
     * Gets context size for monitoring
     */
    async getContextSize(context: ProcessRunContext): Promise<number> {
        const serialized = await this.serializeContext(context);
        return new TextEncoder().encode(serialized).length;
    }

    /**
     * Validates context integrity
     */
    async validateContext(context: ProcessRunContext): Promise<ValidationResult> {
        return this.validator.validateProcessRunContext(context);
    }
}
