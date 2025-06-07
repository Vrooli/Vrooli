import { type Logger } from "winston";
import {
    type Run,
    type Location,
    type Navigator,
    type BranchExecution,
    type RunContext,
    type RunEventType,
    RunEventType as RunEventTypeEnum,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/eventBus.js";
import { type StepExecutor } from "./stepExecutor.js";

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
 * - Context isolation and merging
 * - Failure handling and recovery
 * - Progress tracking across branches
 * 
 * The coordinator ensures that parallel branches execute efficiently while
 * maintaining proper isolation and resource constraints.
 */
export class BranchCoordinator {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly activeBranches: Map<string, BranchExecution> = new Map();

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
    }

    /**
     * Creates branches for execution
     */
    async createBranches(
        runId: string,
        locations: Location[],
        parallel: boolean,
    ): Promise<BranchExecution[]> {
        const branches: BranchExecution[] = [];
        const parentStepId = locations[0]?.nodeId || "unknown";

        for (let i = 0; i < locations.length; i++) {
            const branch: BranchExecution = {
                id: generatePk(),
                parentStepId,
                steps: [],
                state: "pending",
                parallel,
            };

            branches.push(branch);
            this.activeBranches.set(branch.id, branch);

            // Emit branch creation event
            await this.eventBus.publish("run.events", {
                type: RunEventTypeEnum.BRANCH_CREATED,
                timestamp: new Date(),
                runId,
                metadata: {
                    branchId: branch.id,
                    location: locations[i],
                    parallel,
                },
            });
        }

        this.logger.info("[BranchCoordinator] Created branches", {
            runId,
            count: branches.length,
            parallel,
        });

        return branches;
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
                await this.eventBus.publish("run.events", {
                    type: RunEventTypeEnum.BRANCH_FAILED,
                    timestamp: new Date(),
                    runId: run.id,
                    metadata: {
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

        const result: BranchResult = {
            branchId: branch.id,
            success: true,
            completedSteps: 0,
            failedSteps: 0,
            skippedSteps: 0,
            outputs: {},
        };

        try {
            // TODO: Implement actual branch execution logic
            // This would involve:
            // 1. Creating a branch-specific context
            // 2. Navigating through the branch path
            // 3. Executing each step
            // 4. Collecting outputs
            // 5. Handling failures

            // For now, simulate execution
            await new Promise(resolve => setTimeout(resolve, 100));
            result.completedSteps = 1;

            // Update branch state
            branch.state = "completed";
            this.activeBranches.set(branch.id, branch);

            // Emit completion event
            await this.eventBus.publish("run.events", {
                type: RunEventTypeEnum.BRANCH_COMPLETED,
                timestamp: new Date(),
                runId: run.id,
                metadata: {
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
            blackboard: parentContext.blackboard,
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

            await this.eventBus.publish("run.events", {
                type: RunEventTypeEnum.BRANCH_FAILED,
                timestamp: new Date(),
                runId,
                metadata: {
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
}
