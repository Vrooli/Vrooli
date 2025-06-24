import { type Logger } from "winston";
import { randomUUID } from "crypto";
import { deepClone } from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type StepExecutor, type StepExecutionResult } from "./stepExecutor.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { BaseComponent } from "../../shared/BaseComponent.js";

// Helper function to generate IDs
function generatePK(): string {
    return randomUUID();
}

// Import types from local execution architecture
// Note: These event types are used for the event bus communication

const RunEventTypeEnum = {
    RUN_STARTED: "RUN_STARTED" as const,
    RUN_PAUSED: "RUN_PAUSED" as const,
    RUN_RESUMED: "RUN_RESUMED" as const,
    RUN_COMPLETED: "RUN_COMPLETED" as const,
    RUN_FAILED: "RUN_FAILED" as const,
    RUN_CANCELLED: "RUN_CANCELLED" as const,
    STEP_STARTED: "STEP_STARTED" as const,
    STEP_COMPLETED: "STEP_COMPLETED" as const,
    STEP_FAILED: "STEP_FAILED" as const,
    STEP_SKIPPED: "STEP_SKIPPED" as const,
    BRANCH_CREATED: "BRANCH_CREATED" as const,
    BRANCH_COMPLETED: "BRANCH_COMPLETED" as const,
    BRANCH_FAILED: "BRANCH_FAILED" as const,
    CONTEXT_UPDATED: "CONTEXT_UPDATED" as const,
    VARIABLE_SET: "VARIABLE_SET" as const,
    CHECKPOINT_CREATED: "CHECKPOINT_CREATED" as const,
    BOTTLENECK_DETECTED: "BOTTLENECK_DETECTED" as const,
    OPTIMIZATION_APPLIED: "OPTIMIZATION_APPLIED" as const,
};

// Local type definitions based on existing interfaces
interface Location {
    id: string;
    routineId: string;
    nodeId: string;
    branchId?: string;
    index?: number;
}

interface StepInfo {
    id: string;
    name: string;
    type: string;
    description?: string;
    inputs?: Record<string, unknown>;
    outputs?: Record<string, unknown>;
    config?: Record<string, unknown>;
}

interface ContextScope {
    id: string;
    name: string;
    parentId?: string;
    variables: Record<string, unknown>;
}

interface RunContext {
    variables: Record<string, unknown>;
    blackboard: Record<string, unknown>;
    scopes: ContextScope[];
}

interface StepStatus {
    id: string;
    state: "pending" | "ready" | "running" | "completed" | "failed" | "skipped";
    startedAt?: Date;
    completedAt?: Date;
    error?: string;
    result?: unknown;
}

interface BranchExecution {
    id: string;
    parentStepId: string;
    steps?: StepStatus[];
    state: "pending" | "running" | "completed" | "failed";
    parallel: boolean;
    branchIndex?: number; // Index for selecting which parallel path to execute
}

interface BranchConfig {
    parentStepId: string;
    parallel: boolean;
    branchCount?: number; // For parallel: how many branches to create
    predefinedPaths?: Location[][]; // Optional: pre-calculated paths
}

interface RunConfig {
    maxSteps?: number;
    maxDepth?: number;
    maxTime?: number;
    maxCost?: number;
    parallelization?: boolean;
    checkpointInterval?: number;
    recoveryStrategy?: "retry" | "skip" | "fail";
}

interface Run {
    id: string;
    routineId: string;
    context: RunContext;
    config: RunConfig;
}

interface Navigator {
    type: string;
    version: string;
    canNavigate?(routine: unknown): boolean;
    getStartLocation?(routine: unknown): Location;
    getNextLocations?(current: Location, context: Record<string, unknown>): Location[];
    isEndLocation?(location: Location): boolean;
    getStepInfo(location: Location): StepInfo; // Required method
    getDependencies?(location: Location): string[];
    getParallelBranches?(location: Location): Location[][];
}

/**
 * Branch creation parameters
 */
export interface BranchCreateParams {
    id: string;
    parentStepId: string;
    locations: Location[];
    parallel: boolean;
}

/**
 * Branch execution result
 */
export interface BranchResult {
    branchId: string;
    success: boolean;
    completedSteps: number;
    failedSteps: number;
    skippedSteps: number;
    outputs: Record<string, unknown>;
    error?: string;
}

/**
 * BranchCoordinator - Manages parallel and sequential branch execution
 * 
 * This component orchestrates the execution of multiple branches in a workflow,
 * handling both parallel and sequential execution patterns. It manages:
 * 
 * - Parallel branch creation and coordination
 * - Resource allocation across branches
 * - Context isolation with deep-cloned blackboards per branch
 * - Output collection and merging
 * - Failure handling and recovery
 * - Progress tracking across branches
 * 
 * The coordinator ensures that parallel branches execute in complete isolation,
 * with each branch receiving its own deep-cloned context and blackboard.
 * Data sharing between branches only occurs through explicit outputs when
 * the subroutine completes.
 */
export class BranchCoordinator extends BaseComponent {
    private readonly activeBranches: Map<string, BranchExecution> = new Map();
    private readonly instanceId: string;
    private readonly stateStore: IRunStateStore;

    constructor(logger: Logger, eventBus: EventBus, stateStore: IRunStateStore) {
        super(logger, eventBus, "BranchCoordinator");
        this.stateStore = stateStore;
        this.instanceId = `branch-coordinator-${randomUUID()}`;
        
        this.logger.info(`[BranchCoordinator] Initialized with instanceId: ${this.instanceId}`);
    }

    /**
     * Restore branches for a run from persistent storage
     * Call this when resuming a run after system restart
     */
    async restoreBranches(runId: string): Promise<void> {
        try {
            const persistedBranches = await this.stateStore.listBranches(runId);
            
            for (const branch of persistedBranches) {
                this.activeBranches.set(branch.id, branch);
                
                this.logger.debug("[BranchCoordinator] Restored branch from storage", {
                    runId,
                    branchId: branch.id,
                    state: branch.state,
                    instanceId: this.instanceId,
                });
            }
            
            this.logger.info(`[BranchCoordinator] Restored ${persistedBranches.length} branches for run ${runId}`, {
                instanceId: this.instanceId,
            });
            
        } catch (error) {
            this.logger.error(`[BranchCoordinator] Failed to restore branches for run ${runId}`, {
                error: error instanceof Error ? error.message : String(error),
                instanceId: this.instanceId,
            });
            throw error;
        }
    }


    /**
     * Creates branches using the new configuration-based API
     * 
     * This method provides a cleaner interface for branch creation that:
     * - Separates branch creation logic from location arrays
     * - Validates branch count against available parallel paths
     * - Provides better error handling and logging
     * - Supports multiple creation patterns (navigator-driven, explicit count, predefined paths)
     */
    async createBranchesFromConfig(
        runId: string,
        config: BranchConfig,
        navigator?: Navigator,
    ): Promise<BranchExecution[]> {
        const branches: BranchExecution[] = [];
        
        // Determine how many branches to create
        let branchCount: number;
        
        if (config.parallel) {
            if (config.predefinedPaths) {
                branchCount = config.predefinedPaths.length;
                this.logger.debug("[BranchCoordinator] Using predefined paths", {
                    pathCount: branchCount,
                    parentStepId: config.parentStepId,
                });
            } else if (config.branchCount) {
                branchCount = config.branchCount;
                this.logger.debug("[BranchCoordinator] Using explicit branch count", {
                    branchCount,
                    parentStepId: config.parentStepId,
                });
            } else if (navigator) {
                // Query navigator for actual parallel paths
                const parentLocation: Location = {
                    id: config.parentStepId,
                    routineId: runId,
                    nodeId: config.parentStepId,
                };
                const parallelPaths = navigator.getParallelBranches?.(parentLocation) || [];
                branchCount = parallelPaths.length;
                
                if (branchCount === 0) {
                    this.logger.warn("[BranchCoordinator] No parallel paths found", {
                        parentStepId: config.parentStepId,
                        runId,
                    });
                    branchCount = 1; // Fallback to single branch
                }
                
                this.logger.debug("[BranchCoordinator] Derived branch count from navigator", {
                    branchCount,
                    parentStepId: config.parentStepId,
                });
            } else {
                throw new Error("Parallel branches require branchCount, predefinedPaths, or navigator");
            }
        } else {
            branchCount = 1; // Sequential execution uses single branch
        }
        
        // Create branches with proper indices
        for (let i = 0; i < branchCount; i++) {
            const branch: BranchExecution = {
                id: generatePK(),
                parentStepId: config.parentStepId,
                steps: [],
                state: "pending",
                parallel: config.parallel,
                branchIndex: config.parallel ? i : undefined,
            };
            
            branches.push(branch);
            this.activeBranches.set(branch.id, branch);
            
            // Persist branch to state store
            try {
                await this.stateStore.createBranch(runId, branch);
                this.logger.debug("[BranchCoordinator] Branch persisted to state store", {
                    runId,
                    branchId: branch.id,
                    instanceId: this.instanceId,
                });
            } catch (error) {
                this.logger.error("[BranchCoordinator] Failed to persist branch to state store", {
                    runId,
                    branchId: branch.id,
                    error: error instanceof Error ? error.message : String(error),
                    instanceId: this.instanceId,
                });
                // Don't throw - continue with in-memory operation for resilience
            }
            
            // Emit creation event with enhanced data
            await this.publishEvent(RunEventTypeEnum.BRANCH_CREATED, {
                runId,
                branchId: branch.id,
                parentStepId: config.parentStepId,
                branchIndex: branch.branchIndex,
                parallel: config.parallel,
                totalBranches: branchCount,
                instanceId: this.instanceId,
            });
        }
        
        this.logger.info("[BranchCoordinator] Created branches from config", {
            runId,
            parentStepId: config.parentStepId,
            branchCount,
            parallel: config.parallel,
            branchIds: branches.map(b => b.id),
        });
        
        return branches;
    }

    /**
     * Convenience method: Creates parallel branches using navigator to determine count
     */
    async createParallelBranches(
        runId: string,
        parentStepId: string,
        navigator: Navigator,
    ): Promise<BranchExecution[]> {
        const config: BranchConfig = {
            parentStepId,
            parallel: true,
        };
        
        return this.createBranchesFromConfig(runId, config, navigator);
    }

    /**
     * Convenience method: Creates a single sequential branch
     */
    async createSequentialBranch(
        runId: string,
        parentStepId: string,
    ): Promise<BranchExecution[]> {
        const config: BranchConfig = {
            parentStepId,
            parallel: false,
        };
        
        return this.createBranchesFromConfig(runId, config);
    }

    /**
     * Convenience method: Creates branches with predefined execution paths
     */
    async createBranchesWithPredefinedPaths(
        runId: string,
        parentStepId: string,
        paths: Location[][],
    ): Promise<BranchExecution[]> {
        const config: BranchConfig = {
            parentStepId,
            parallel: true,
            predefinedPaths: paths,
        };
        
        return this.createBranchesFromConfig(runId, config);
    }

    /**
     * Convenience method: Creates a specific number of parallel branches
     */
    async createParallelBranchesWithCount(
        runId: string,
        parentStepId: string,
        branchCount: number,
    ): Promise<BranchExecution[]> {
        const config: BranchConfig = {
            parentStepId,
            parallel: true,
            branchCount,
        };
        
        return this.createBranchesFromConfig(runId, config);
    }

    /**
     * Executes branches (parallel or sequential)
     */
    async executeBranches(
        run: Run,
        branches: BranchExecution[],
        navigator: Navigator,
        stepExecutor: StepExecutor,
    ): Promise<BranchResult[]> {
        if (branches.length === 0) {
            return [];
        }

        if (branches[0].parallel) {
            return this.executeParallelBranches(run, branches, navigator, stepExecutor);
        } else {
            return this.executeSequentialBranches(run, branches, navigator, stepExecutor);
        }
    }

    /**
     * Executes branches in parallel
     */
    private async executeParallelBranches(
        run: Run,
        branches: BranchExecution[],
        navigator: Navigator,
        stepExecutor: StepExecutor,
    ): Promise<BranchResult[]> {
        this.logger.info("[BranchCoordinator] Executing parallel branches", {
            runId: run.id,
            branchCount: branches.length,
        });

        // Start all branches
        const branchPromises = branches.map(branch => 
            this.executeSingleBranch(run, branch, navigator, stepExecutor),
        );

        // Wait for all to complete
        const results = await Promise.allSettled(branchPromises);

        // Process results
        const branchResults: BranchResult[] = [];
        for (let i = 0; i < results.length; i++) {
            const result = results[i];
            const branch = branches[i];

            if (result.status === "fulfilled") {
                branchResults.push(result.value);
            } else {
                // Branch failed
                branchResults.push({
                    branchId: branch.id,
                    success: false,
                    completedSteps: 0,
                    failedSteps: 1,
                    skippedSteps: 0,
                    outputs: {},
                    error: result.reason?.message || "Branch execution failed",
                });

                // Emit failure event
                await this.eventBus.publish({
                    id: generatePK(),
                    type: RunEventTypeEnum.BRANCH_FAILED,
                    timestamp: new Date(),
                    source: {
                        tier: 2,
                        component: "BranchCoordinator",
                        instanceId: this.instanceId,
                    },
                    data: {
                        runId: run.id,
                        branchId: branch.id,
                        error: result.reason?.message,
                    },
                });
            }
        }

        return branchResults;
    }

    /**
     * Executes branches sequentially
     */
    private async executeSequentialBranches(
        run: Run,
        branches: BranchExecution[],
        navigator: Navigator,
        stepExecutor: StepExecutor,
    ): Promise<BranchResult[]> {
        this.logger.info("[BranchCoordinator] Executing sequential branches", {
            runId: run.id,
            branchCount: branches.length,
        });

        const results: BranchResult[] = [];

        for (const branch of branches) {
            try {
                const result = await this.executeSingleBranch(run, branch, navigator, stepExecutor);
                results.push(result);

                // Stop on failure if configured
                if (!result.success && run.config.recoveryStrategy === "fail") {
                    break;
                }
            } catch (error) {
                const errorResult: BranchResult = {
                    branchId: branch.id,
                    success: false,
                    completedSteps: 0,
                    failedSteps: 1,
                    skippedSteps: 0,
                    outputs: {},
                    error: error instanceof Error ? error.message : String(error),
                };
                results.push(errorResult);

                // Stop on failure if configured
                if (run.config.recoveryStrategy === "fail") {
                    break;
                }
            }
        }

        return results;
    }

    /**
     * Executes a single branch
     */
    private async executeSingleBranch(
        run: Run,
        branch: BranchExecution,
        navigator: Navigator,
        stepExecutor: StepExecutor,
    ): Promise<BranchResult> {
        this.logger.debug("[BranchCoordinator] Executing branch", {
            runId: run.id,
            branchId: branch.id,
        });

        // Update branch state
        branch.state = "running";
        this.activeBranches.set(branch.id, branch);
        
        // Persist state change
        try {
            await this.stateStore.updateBranch(run.id, branch.id, { state: "running" });
        } catch (error) {
            this.logger.error("[BranchCoordinator] Failed to persist branch state update to running", {
                runId: run.id,
                branchId: branch.id,
                error: error instanceof Error ? error.message : String(error),
                instanceId: this.instanceId,
            });
        }

        const result: BranchResult = {
            branchId: branch.id,
            success: true,
            completedSteps: 0,
            failedSteps: 0,
            skippedSteps: 0,
            outputs: {},
        };

        try {
            // Phase 1: Create branch-specific context
            const branchContext = await this.createBranchContext(run, branch);
            
            // Phase 2: Navigate through the branch path
            const branchPath = await this.determineBranchPath(run, branch, navigator, branchContext);
            
            // Phase 3: Execute each step in the branch
            const executionResults = await this.executeStepsInBranch(
                run, 
                branch, 
                branchPath, 
                branchContext, 
                stepExecutor,
            );
            
            // Phase 4: Collect outputs
            result.outputs = await this.collectBranchOutputs(executionResults);
            result.completedSteps = executionResults.filter(r => r.success).length;
            result.failedSteps = executionResults.filter(r => !r.success).length;
            
            // Phase 5: Handle any failures according to recovery strategy
            if (result.failedSteps > 0) {
                const shouldContinue = await this.handleBranchFailures(
                    run, 
                    branch, 
                    executionResults,
                );
                
                if (!shouldContinue) {
                    result.success = false;
                    result.error = `Branch failed with ${result.failedSteps} failed steps`;
                }
            }

            // Update branch state
            branch.state = "completed";
            this.activeBranches.set(branch.id, branch);
            
            // Persist state change
            try {
                await this.stateStore.updateBranch(run.id, branch.id, { state: "completed" });
            } catch (error) {
                this.logger.error("[BranchCoordinator] Failed to persist branch state update to completed", {
                    runId: run.id,
                    branchId: branch.id,
                    error: error instanceof Error ? error.message : String(error),
                    instanceId: this.instanceId,
                });
            }

            // Emit completion event
            await this.eventBus.publish({
                id: generatePK(),
                type: RunEventTypeEnum.BRANCH_COMPLETED,
                timestamp: new Date(),
                source: {
                    tier: 2,
                    component: "BranchCoordinator",
                    instanceId: this.instanceId,
                },
                data: {
                    runId: run.id,
                    branchId: branch.id,
                    result,
                },
            });

        } catch (error) {
            result.success = false;
            result.failedSteps = 1;
            result.error = error instanceof Error ? error.message : String(error);

            // Update branch state
            branch.state = "failed";
            this.activeBranches.set(branch.id, branch);
            
            // Persist state change
            try {
                await this.stateStore.updateBranch(run.id, branch.id, { state: "failed" });
            } catch (error) {
                this.logger.error("[BranchCoordinator] Failed to persist branch state update to failed", {
                    runId: run.id,
                    branchId: branch.id,
                    error: error instanceof Error ? error.message : String(error),
                    instanceId: this.instanceId,
                });
            }

            this.logger.error("[BranchCoordinator] Branch execution failed", {
                runId: run.id,
                branchId: branch.id,
                error: result.error,
            });
        }

        return result;
    }

    /**
     * Merges branch results into parent context
     */
    async mergeBranchResults(
        parentContext: RunContext,
        results: BranchResult[],
    ): Promise<RunContext> {
        // Clone parent context
        const merged: RunContext = {
            variables: { ...parentContext.variables },
            blackboard: deepClone(parentContext.blackboard), // Deep clone for isolation
            scopes: [...parentContext.scopes],
        };

        // Merge outputs from all branches
        const mergedOutputs: Record<string, unknown[]> = {};

        for (const result of results) {
            if (!result.success) continue;

            for (const [key, value] of Object.entries(result.outputs)) {
                if (!mergedOutputs[key]) {
                    mergedOutputs[key] = [];
                }
                mergedOutputs[key].push(value);
            }
        }

        // Add merged outputs to context
        for (const [key, values] of Object.entries(mergedOutputs)) {
            if (values.length === 1) {
                merged.variables[key] = values[0];
            } else {
                // Multiple values - store as array
                merged.variables[`${key}_merged`] = values;
            }
        }

        this.logger.debug("[BranchCoordinator] Merged branch results", {
            branchCount: results.length,
            successfulBranches: results.filter(r => r.success).length,
            mergedVars: Object.keys(mergedOutputs).length,
        });

        return merged;
    }

    /**
     * Cancels active branches
     */
    async cancelBranches(runId: string): Promise<void> {
        const branches = Array.from(this.activeBranches.values())
            .filter(b => b.state === "running");

        for (const branch of branches) {
            branch.state = "failed";
            this.activeBranches.set(branch.id, branch);
            
            // Persist state change
            try {
                await this.stateStore.updateBranch(runId, branch.id, { state: "failed" });
            } catch (error) {
                this.logger.error("[BranchCoordinator] Failed to persist branch cancellation state", {
                    runId,
                    branchId: branch.id,
                    error: error instanceof Error ? error.message : String(error),
                    instanceId: this.instanceId,
                });
            }

            await this.eventBus.publish({
                id: generatePK(),
                type: RunEventTypeEnum.BRANCH_FAILED,
                timestamp: new Date(),
                source: {
                    tier: 2,
                    component: "BranchCoordinator",
                    instanceId: this.instanceId,
                },
                data: {
                    runId,
                    branchId: branch.id,
                    reason: "Cancelled",
                },
            });
        }

        this.logger.info("[BranchCoordinator] Cancelled branches", {
            runId,
            cancelledCount: branches.length,
        });
    }

    /**
     * Gets branch status
     */
    async getBranchStatus(branchId: string): Promise<BranchExecution | null> {
        return this.activeBranches.get(branchId) || null;
    }

    /**
     * Cleans up completed branches
     */
    async cleanup(runId: string): Promise<void> {
        const branches = Array.from(this.activeBranches.entries());
        let removed = 0;

        for (const [id, branch] of branches) {
            if (["completed", "failed"].includes(branch.state)) {
                this.activeBranches.delete(id);
                removed++;
            }
        }

        if (removed > 0) {
            this.logger.debug("[BranchCoordinator] Cleaned up branches", {
                runId,
                removedCount: removed,
            });
        }
    }

    /**
     * Phase 1: Creates a branch-specific context with proper isolation
     */
    private async createBranchContext(run: Run, branch: BranchExecution): Promise<RunContext> {
        this.logger.debug("[BranchCoordinator] Creating branch context", {
            runId: run.id,
            branchId: branch.id,
        });

        // Clone the parent context for isolation
        const branchContext: RunContext = {
            variables: { ...run.context.variables },
            blackboard: deepClone(run.context.blackboard), // Deep clone to isolate branch context
            scopes: [
                ...run.context.scopes.map(scope => ({ ...scope })), // Clone scopes
                {
                    id: `branch-${branch.id}`,
                    name: `Branch ${branch.id}`,
                    parentId: run.context.scopes[run.context.scopes.length - 1]?.id,
                    variables: {}, // Branch-specific variables
                },
            ],
        };

        // Emit context creation event
        await this.eventBus.publish({
            id: generatePK(),
            type: RunEventTypeEnum.CONTEXT_UPDATED,
            timestamp: new Date(),
            source: {
                tier: 2,
                component: "BranchCoordinator",
                instanceId: this.instanceId,
            },
            data: {
                runId: run.id,
                branchId: branch.id,
                scopesCount: branchContext.scopes.length,
                operation: "branch_context_created",
            },
        });

        return branchContext;
    }

    /**
     * Phase 2: Determines the execution path for this branch
     */
    private async determineBranchPath(
        run: Run,
        branch: BranchExecution,
        navigator: Navigator,
        _context: RunContext,
    ): Promise<Location[]> {
        this.logger.debug("[BranchCoordinator] Determining branch path", {
            runId: run.id,
            branchId: branch.id,
        });

        const path: Location[] = [];
        
        // For now, use a simple approach where each branch corresponds to a specific location
        // In the future, this could be enhanced to support complex branch definitions
        
        // If branch has specific steps defined, use those
        if (branch.steps && branch.steps.length > 0) {
            // Convert step statuses to locations 
            for (const stepStatus of branch.steps) {
                const location: Location = {
                    id: stepStatus.id,
                    routineId: run.routineId,
                    nodeId: stepStatus.id,
                    branchId: branch.id,
                };
                path.push(location);
            }
        } else {
            // Determine path from navigator based on parent step
            const parentLocation: Location = {
                id: branch.parentStepId,
                routineId: run.routineId,
                nodeId: branch.parentStepId,
            };
            
            if (branch.parallel) {
                // Handle parallel branch path selection
                const parallelBranches = navigator.getParallelBranches?.(parentLocation) || [];
                
                if (parallelBranches.length === 0) {
                    this.logger.warn("[BranchCoordinator] No parallel paths available for parallel branch", {
                        branchId: branch.id,
                        parentStepId: branch.parentStepId,
                        runId: run.id,
                    });
                    path.push(parentLocation); // Fallback to parent step
                } else {
                    const branchIndex = branch.branchIndex ?? 0;
                    
                    if (branchIndex < parallelBranches.length) {
                        path.push(...parallelBranches[branchIndex]);
                        
                        this.logger.debug("[BranchCoordinator] Selected parallel path", {
                            branchId: branch.id,
                            branchIndex,
                            pathLength: parallelBranches[branchIndex].length,
                            totalPaths: parallelBranches.length,
                        });
                    } else {
                        // Better error handling for out-of-bounds
                        this.logger.error("[BranchCoordinator] Branch index exceeds available paths", {
                            branchId: branch.id,
                            branchIndex,
                            availablePaths: parallelBranches.length,
                            parentStepId: branch.parentStepId,
                        });
                        
                        // Use last available path instead of first for better distribution
                        const fallbackIndex = Math.min(branchIndex, parallelBranches.length - 1);
                        path.push(...parallelBranches[fallbackIndex]);
                        
                        this.logger.warn("[BranchCoordinator] Using fallback parallel path", {
                            branchId: branch.id,
                            requestedIndex: branchIndex,
                            fallbackIndex,
                        });
                    }
                }
            } else {
                // Sequential branch - use parent location directly
                path.push(parentLocation);
                
                this.logger.debug("[BranchCoordinator] Using parent location for sequential branch", {
                    branchId: branch.id,
                    parentStepId: branch.parentStepId,
                });
            }
        }

        this.logger.debug("[BranchCoordinator] Branch path determined", {
            runId: run.id,
            branchId: branch.id,
            pathLength: path.length,
            locationIds: path.map(l => l.nodeId),
        });

        return path;
    }

    /**
     * Phase 3: Executes each step in the branch path
     */
    private async executeStepsInBranch(
        run: Run,
        branch: BranchExecution,
        path: Location[],
        context: RunContext,
        stepExecutor: StepExecutor,
    ): Promise<StepExecutionResult[]> {
        this.logger.debug("[BranchCoordinator] Executing steps in branch", {
            runId: run.id,
            branchId: branch.id,
            stepCount: path.length,
        });

        const results: StepExecutionResult[] = [];
        let currentContext = context;

        for (let i = 0; i < path.length; i++) {
            const location = path[i];
            
            try {
                // Create basic step information 
                // Note: In a real implementation, this would be populated from the routine definition
                const stepInfo: StepInfo = {
                    id: location.nodeId,
                    name: `Step ${location.nodeId}`,
                    type: "action", // Default type
                    description: `Executing step ${location.nodeId}`,
                };
                
                // Create step status if not exists
                let stepStatus = branch.steps?.find(s => s.id === location.nodeId);
                if (!stepStatus) {
                    stepStatus = {
                        id: location.nodeId,
                        state: "running",
                        startedAt: new Date(),
                    };
                    
                    if (!branch.steps) {
                        branch.steps = [];
                    }
                    branch.steps.push(stepStatus);
                }

                // Update step status
                stepStatus.state = "running";
                stepStatus.startedAt = new Date();

                // Emit step started event
                await this.eventBus.publish({
                    id: generatePK(),
                    type: RunEventTypeEnum.STEP_STARTED,
                    timestamp: new Date(),
                    source: {
                        tier: 2,
                        component: "BranchCoordinator",
                        instanceId: this.instanceId,
                    },
                    data: {
                        runId: run.id,
                        stepId: location.nodeId,
                        branchId: branch.id,
                        stepIndex: i,
                        stepInfo,
                    },
                });

                // Execute the step through StepExecutor
                const executionParams = {
                    runId: run.id,
                    stepId: location.nodeId,
                    stepInfo,
                    context: currentContext,
                    location,
                };

                const result = await stepExecutor.executeStep(executionParams);
                results.push(result);

                // Update step status based on result
                stepStatus.state = result.success ? "completed" : "failed";
                stepStatus.completedAt = new Date();
                
                if (result.success) {
                    stepStatus.result = result.outputs;
                    
                    // Update context with step outputs
                    if (result.outputs) {
                        currentContext = this.updateContextWithOutputs(currentContext, result.outputs);
                    }

                    // Emit step completed event
                    await this.eventBus.publish({
                        id: generatePK(),
                        type: RunEventTypeEnum.STEP_COMPLETED,
                        timestamp: new Date(),
                        source: {
                            tier: 2,
                            component: "BranchCoordinator",
                            instanceId: this.instanceId,
                        },
                        data: {
                            runId: run.id,
                            stepId: location.nodeId,
                            branchId: branch.id,
                            stepIndex: i,
                            outputs: result.outputs,
                            duration: result.duration,
                        },
                    });
                } else {
                    stepStatus.error = result.error;

                    // Emit step failed event
                    await this.eventBus.publish({
                        id: generatePK(),
                        type: RunEventTypeEnum.STEP_FAILED,
                        timestamp: new Date(),
                        source: {
                            tier: 2,
                            component: "BranchCoordinator",
                            instanceId: this.instanceId,
                        },
                        data: {
                            runId: run.id,
                            stepId: location.nodeId,
                            branchId: branch.id,
                            stepIndex: i,
                            error: result.error,
                        },
                    });

                    // Check recovery strategy
                    if (run.config.recoveryStrategy === "fail") {
                        this.logger.info("[BranchCoordinator] Stopping branch execution due to failure", {
                            runId: run.id,
                            branchId: branch.id,
                            failedStepId: location.nodeId,
                        });
                        break; // Stop execution on failure
                    }
                }

            } catch (error) {
                // Handle execution errors
                const errorMessage = error instanceof Error ? error.message : String(error);
                
                this.logger.error("[BranchCoordinator] Step execution error", {
                    runId: run.id,
                    branchId: branch.id,
                    stepId: location.nodeId,
                    error: errorMessage,
                });

                // Add error result
                results.push({
                    success: false,
                    error: errorMessage,
                    duration: 0,
                });

                // Update step status
                const stepStatus = branch.steps?.find(s => s.id === location.nodeId);
                if (stepStatus) {
                    stepStatus.state = "failed";
                    stepStatus.error = errorMessage;
                    stepStatus.completedAt = new Date();
                }

                // Check recovery strategy
                if (run.config.recoveryStrategy === "fail") {
                    break;
                }
            }
        }

        return results;
    }

    /**
     * Phase 4: Collects and aggregates outputs from branch execution
     */
    private async collectBranchOutputs(results: StepExecutionResult[]): Promise<Record<string, unknown>> {
        const outputs: Record<string, unknown> = {};

        for (const result of results) {
            if (result.success && result.outputs) {
                // Merge outputs, handling conflicts by creating arrays
                for (const [key, value] of Object.entries(result.outputs)) {
                    if (outputs[key] !== undefined) {
                        // Conflict - convert to array or append to existing array
                        if (Array.isArray(outputs[key])) {
                            (outputs[key] as unknown[]).push(value);
                        } else {
                            outputs[key] = [outputs[key], value];
                        }
                    } else {
                        outputs[key] = value;
                    }
                }
            }
        }

        return outputs;
    }

    /**
     * Phase 5: Handles failures according to recovery strategy
     */
    private async handleBranchFailures(
        run: Run,
        branch: BranchExecution,
        results: StepExecutionResult[],
    ): Promise<boolean> {
        const failures = results.filter(r => !r.success);
        
        if (failures.length === 0) {
            return true; // No failures to handle
        }

        this.logger.info("[BranchCoordinator] Handling branch failures", {
            runId: run.id,
            branchId: branch.id,
            failureCount: failures.length,
            recoveryStrategy: run.config.recoveryStrategy,
        });

        switch (run.config.recoveryStrategy) {
            case "retry":
                // TODO: Implement retry logic - for now, just continue
                this.logger.warn("[BranchCoordinator] Retry strategy not yet implemented, continuing");
                return true;

            case "skip":
                // Continue execution despite failures
                this.logger.info("[BranchCoordinator] Skipping failed steps and continuing");
                return true;

            case "fail":
            default:
                // Stop execution on failure
                this.logger.info("[BranchCoordinator] Failing branch due to step failures");
                return false;
        }
    }

    /**
     * Helper: Updates context with step outputs
     */
    private updateContextWithOutputs(context: RunContext, outputs: Record<string, unknown>): RunContext {
        const updatedContext: RunContext = {
            ...context,
            variables: { ...context.variables },
            scopes: context.scopes.map(scope => ({ ...scope })),
        };

        // Add outputs to the branch scope (last scope)
        const branchScope = updatedContext.scopes[updatedContext.scopes.length - 1];
        if (branchScope) {
            branchScope.variables = {
                ...branchScope.variables,
                ...outputs,
            };
        }

        // Also add to main variables for backward compatibility
        Object.assign(updatedContext.variables, outputs);

        return updatedContext;
    }
}
