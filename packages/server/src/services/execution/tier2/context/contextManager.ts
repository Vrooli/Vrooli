import { type Logger } from "winston";
import {
    type RunContext,
    type ContextScope,
} from "@vrooli/shared";
import { type IRunStateStore } from "../state/runStateStore.js";

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
 * - Blackboard pattern for shared state
 * - Context serialization for checkpoints
 * 
 * The context system is critical for maintaining proper state isolation
 * in complex workflows with parallel execution and nested subroutines.
 */
export class ContextManager {
    private readonly stateStore: IRunStateStore;
    private readonly logger: Logger;
    private readonly contextCache: Map<string, RunContext> = new Map();

    constructor(stateStore: IRunStateStore, logger: Logger) {
        this.stateStore = stateStore;
        this.logger = logger;
    }

    /**
     * Creates a new context
     */
    async createContext(params: ContextInitParams = {}): Promise<RunContext> {
        const context: RunContext = {
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
     * Clones a context for branching
     */
    async cloneContext(context: RunContext, branchId: string): Promise<RunContext> {
        // Deep clone the context
        const cloned: RunContext = {
            variables: { ...context.variables },
            blackboard: context.blackboard, // Blackboard is shared
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
        parent: RunContext,
        branches: RunContext[],
        strategy: "first" | "last" | "merge" = "merge",
    ): Promise<RunContext> {
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
        context: RunContext,
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
        context: RunContext,
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
        context: RunContext,
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
    async popScope(context: RunContext): Promise<ContextScope | undefined> {
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
        context: RunContext,
        key: string,
    ): Promise<unknown> {
        return context.blackboard[key];
    }

    /**
     * Sets blackboard value
     */
    async setBlackboardValue(
        context: RunContext,
        key: string,
        value: unknown,
    ): Promise<void> {
        context.blackboard[key] = value;
    }

    /**
     * Gets context from cache or state store
     */
    async getContext(runId: string): Promise<RunContext> {
        // Check cache first
        const cached = this.contextCache.get(runId);
        if (cached) {
            return cached;
        }

        // Load from state store
        const context = await this.stateStore.getContext(runId);
        this.contextCache.set(runId, context);

        return context;
    }

    /**
     * Updates context in cache and state store
     */
    async updateContext(runId: string, context: RunContext): Promise<void> {
        // Update cache
        this.contextCache.set(runId, context);

        // Persist to state store
        await this.stateStore.updateContext(runId, context);
    }

    /**
     * Creates a step execution scope
     */
    async createStepScope(
        context: RunContext,
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
    async serializeContext(context: RunContext): Promise<string> {
        return JSON.stringify(context, null, 2);
    }

    /**
     * Deserializes context from checkpoint
     */
    async deserializeContext(data: string): Promise<RunContext> {
        return JSON.parse(data);
    }

    /**
     * Clears context cache for a run
     */
    async clearCache(runId: string): Promise<void> {
        this.contextCache.delete(runId);
    }

    /**
     * Gets context size for monitoring
     */
    async getContextSize(context: RunContext): Promise<number> {
        const serialized = await this.serializeContext(context);
        return new TextEncoder().encode(serialized).length;
    }
}
