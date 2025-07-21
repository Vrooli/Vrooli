/**
 * BPMN Parallel Gateway Handler - Manages parallel execution flows
 * 
 * Handles the complex logic of parallel gateway processing:
 * - Split gateways: Creating multiple parallel execution branches
 * - Join gateways: Synchronizing parallel branches before proceeding
 * - Branch lifecycle management and state tracking
 * - Parallel branch coordination and synchronization
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    ParallelBranch,
    JoinPoint,
    LocationType,
} from "../../types.js";
import type { BpmnModel } from "./BpmnModel.js";
import { ContextUtils } from "../contextTransformer.js";

/**
 * Parallel processing result
 */
export interface ParallelProcessingResult {
    // New locations to navigate to (multiple for splits, single/none for joins)
    nextLocations: AbstractLocation[];
    // Updated context with parallel execution state
    updatedContext: EnhancedExecutionContext;
    // Whether parallel execution has completed
    parallelExecutionComplete: boolean;
    // Information about branch state changes
    branchUpdates: BranchUpdate[];
}

/**
 * Branch update information
 */
export interface BranchUpdate {
    branchId: string;
    action: "created" | "completed" | "failed" | "synchronized";
    location?: AbstractLocation;
    result?: unknown;
    timestamp: Date;
}

/**
 * BPMN Parallel Gateway Handler
 */
export class BpmnParallelHandler {

    /**
     * Process parallel gateway (split or join)
     */
    processParallelGateway(
        model: BpmnModel,
        gatewayId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): ParallelProcessingResult {
        const gateway = model.getElementById(gatewayId);
        if (!gateway) {
            throw new Error(`Gateway not found: ${gatewayId}`);
        }

        const incomingFlows = model.getIncomingFlows(gatewayId);
        const outgoingFlows = model.getOutgoingFlows(gatewayId);

        // Determine gateway type based on flows
        if (incomingFlows.length === 1 && outgoingFlows.length > 1) {
            // Split gateway (fork)
            return this.processParallelSplit(model, gatewayId, outgoingFlows, currentLocation, context);
        } else if (incomingFlows.length > 1 && outgoingFlows.length === 1) {
            // Join gateway (synchronization)
            return this.processParallelJoin(model, gatewayId, incomingFlows, outgoingFlows[0], currentLocation, context);
        } else if (incomingFlows.length === 1 && outgoingFlows.length === 1) {
            // Pass-through gateway (no parallel processing needed)
            return this.processPassThroughGateway(model, outgoingFlows[0], currentLocation, context);
        } else {
            // Complex gateway configuration - handle as inclusive for now
            return this.processComplexParallelGateway(model, gatewayId, incomingFlows, outgoingFlows, currentLocation, context);
        }
    }

    /**
     * Process parallel split gateway (fork)
     */
    private processParallelSplit(
        model: BpmnModel,
        gatewayId: string,
        outgoingFlows: any[],
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): ParallelProcessingResult {
        const result: ParallelProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            parallelExecutionComplete: false,
            branchUpdates: [],
        };

        const splitId = `split_${gatewayId}_${Date.now()}`;
        
        // Create parallel branches for each outgoing flow
        for (let i = 0; i < outgoingFlows.length; i++) {
            const flow = outgoingFlows[i];
            if (!flow.targetRef) continue;

            const branchId = `${splitId}_branch_${i}`;
            const branchLocation = model.createAbstractLocation(
                (flow.targetRef as any).id,
                currentLocation.routineId,
                "parallel_branch",
                {
                    parentNodeId: (flow.targetRef as any).id,
                    branchId,
                    metadata: {
                        splitGateway: gatewayId,
                        branchIndex: i,
                        totalBranches: outgoingFlows.length,
                        splitId,
                    },
                },
            );

            // Create parallel branch tracking
            const parallelBranch: ParallelBranch = {
                id: branchId,
                branchId,
                currentLocation: branchLocation,
                status: "running",
                startedAt: new Date(),
            };

            result.updatedContext = ContextUtils.addParallelBranch(result.updatedContext, parallelBranch);
            result.nextLocations.push(branchLocation);

            // Record branch creation
            result.branchUpdates.push({
                branchId,
                action: "created",
                location: branchLocation,
                timestamp: new Date(),
            });
        }

        return result;
    }

    /**
     * Process parallel join gateway (synchronization)
     */
    private processParallelJoin(
        model: BpmnModel,
        gatewayId: string,
        incomingFlows: any[],
        outgoingFlow: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): ParallelProcessingResult {
        const result: ParallelProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            parallelExecutionComplete: false,
            branchUpdates: [],
        };

        // Check if there's already a join point for this gateway
        let joinPoint = context.parallelExecution.joinPoints.find(jp => jp.gatewayId === gatewayId);
        
        if (!joinPoint) {
            // Create new join point
            const requiredBranches = this.identifyRequiredBranches(model, incomingFlows, context);
            joinPoint = {
                id: `join_${gatewayId}_${Date.now()}`,
                gatewayId,
                requiredBranches,
                completedBranches: [],
                isReady: false,
            };

            result.updatedContext.parallelExecution.joinPoints.push(joinPoint);
        }

        // Update join point with current branch completion
        const currentBranchId = currentLocation.branchId;
        if (currentBranchId && !joinPoint.completedBranches.includes(currentBranchId)) {
            joinPoint.completedBranches.push(currentBranchId);

            // Mark branch as completed in context
            result.updatedContext = ContextUtils.completeParallelBranch(
                result.updatedContext, 
                currentBranchId,
                currentLocation.metadata,
            );

            result.branchUpdates.push({
                branchId: currentBranchId,
                action: "completed",
                location: currentLocation,
                timestamp: new Date(),
            });
        }

        // Check if all required branches have completed
        const allBranchesCompleted = joinPoint.requiredBranches.every(branchId => 
            joinPoint!.completedBranches.includes(branchId),
        );

        if (allBranchesCompleted) {
            // All branches completed - proceed with join
            joinPoint.isReady = true;
            result.parallelExecutionComplete = true;

            // Create location for continuing after join
            if (outgoingFlow.targetRef) {
                const nextLocation = model.createAbstractLocation(
                    (outgoingFlow.targetRef as any).id,
                    currentLocation.routineId,
                    "node",
                    { parentNodeId: (outgoingFlow.targetRef as any).id },
                );
                result.nextLocations.push(nextLocation);
            }

            // Clean up completed join point
            result.updatedContext.parallelExecution.joinPoints = 
                result.updatedContext.parallelExecution.joinPoints.filter(jp => jp.id !== joinPoint!.id);

            // Record synchronization
            result.branchUpdates.push({
                branchId: `all_branches_${gatewayId}`,
                action: "synchronized",
                timestamp: new Date(),
            });
        } else {
            // Still waiting for other branches - create waiting location
            const waitingLocation = model.createAbstractLocation(
                gatewayId,
                currentLocation.routineId,
                "gateway_evaluation",
                {
                    parentNodeId: gatewayId,
                    metadata: {
                        joinPoint,
                        waitingForBranches: joinPoint.requiredBranches.filter(branchId => 
                            !joinPoint!.completedBranches.includes(branchId),
                        ),
                    },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    /**
     * Process pass-through gateway (no parallelism)
     */
    private processPassThroughGateway(
        model: BpmnModel,
        outgoingFlow: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): ParallelProcessingResult {
        const result: ParallelProcessingResult = {
            nextLocations: [],
            updatedContext: context,
            parallelExecutionComplete: false,
            branchUpdates: [],
        };

        if (outgoingFlow.targetRef) {
            const nextLocation = model.createAbstractLocation(
                (outgoingFlow.targetRef as any).id,
                currentLocation.routineId,
                "node",
                { parentNodeId: (outgoingFlow.targetRef as any).id },
            );
            result.nextLocations.push(nextLocation);
        }

        return result;
    }

    /**
     * Process complex parallel gateway configuration
     */
    private processComplexParallelGateway(
        model: BpmnModel,
        gatewayId: string,
        incomingFlows: any[],
        outgoingFlows: any[],
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): ParallelProcessingResult {
        // For complex configurations, treat as both split and join
        // This handles cases like multiple incoming and outgoing flows

        if (incomingFlows.length > 1) {
            // Handle join aspect first
            const joinResult = this.processParallelJoin(
                model, gatewayId, incomingFlows, outgoingFlows[0], currentLocation, context,
            );

            // If join is complete and there are multiple outgoing flows, split
            if (joinResult.parallelExecutionComplete && outgoingFlows.length > 1) {
                return this.processParallelSplit(
                    model, gatewayId, outgoingFlows, currentLocation, joinResult.updatedContext,
                );
            }

            return joinResult;
        } else {
            // Only split needed
            return this.processParallelSplit(
                model, gatewayId, outgoingFlows, currentLocation, context,
            );
        }
    }

    /**
     * Update parallel branch status
     */
    updateBranchStatus(
        context: EnhancedExecutionContext,
        branchId: string,
        status: "pending" | "running" | "completed" | "failed",
        result?: unknown,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        
        updatedContext.parallelExecution.activeBranches = 
            updatedContext.parallelExecution.activeBranches.map(branch => {
                if (branch.branchId === branchId) {
                    return {
                        ...branch,
                        status,
                        completedAt: status === "completed" || status === "failed" ? new Date() : branch.completedAt,
                        result: result !== undefined ? result : branch.result,
                    };
                }
                return branch;
            });

        return updatedContext;
    }

    /**
     * Check if parallel execution is blocked
     */
    isParallelExecutionBlocked(
        context: EnhancedExecutionContext,
        gatewayId: string,
    ): boolean {
        const joinPoint = context.parallelExecution.joinPoints.find(jp => jp.gatewayId === gatewayId);
        
        if (!joinPoint) return false;

        // Check if any required branch has failed
        const failedBranches = context.parallelExecution.activeBranches
            .filter(branch => 
                joinPoint.requiredBranches.includes(branch.branchId) && 
                branch.status === "failed",
            );

        return failedBranches.length > 0;
    }

    /**
     * Get parallel execution summary
     */
    getParallelExecutionSummary(context: EnhancedExecutionContext): {
        activeBranches: number;
        completedBranches: number;
        failedBranches: number;
        pendingJoins: number;
    } {
        const activeBranches = context.parallelExecution.activeBranches.filter(
            branch => branch.status === "running" || branch.status === "pending",
        ).length;

        const completedBranches = context.parallelExecution.completedBranches.length;

        const failedBranches = context.parallelExecution.activeBranches.filter(
            branch => branch.status === "failed",
        ).length;

        const pendingJoins = context.parallelExecution.joinPoints.filter(
            join => !join.isReady,
        ).length;

        return {
            activeBranches,
            completedBranches,
            failedBranches,
            pendingJoins,
        };
    }

    // Helper methods

    /**
     * Identify required branches for a join gateway
     */
    private identifyRequiredBranches(
        model: BpmnModel,
        incomingFlows: any[],
        context: EnhancedExecutionContext,
    ): string[] {
        const requiredBranches: string[] = [];

        // Look for active branches that should arrive at this join
        for (const flow of incomingFlows) {
            if (!flow.sourceRef) continue;

            // Find branches that have this flow's source as their current or target location
            const matchingBranches = context.parallelExecution.activeBranches.filter(branch => {
                const branchLocation = branch.currentLocation as AbstractLocation;
                return branchLocation.nodeId === (flow.sourceRef as any).id ||
                       branchLocation.parentNodeId === (flow.sourceRef as any).id;
            });

            for (const branch of matchingBranches) {
                if (!requiredBranches.includes(branch.branchId)) {
                    requiredBranches.push(branch.branchId);
                }
            }
        }

        // If no specific branches found, look for any branches with matching split gateway
        if (requiredBranches.length === 0) {
            for (const branch of context.parallelExecution.activeBranches) {
                if (branch.status === "running") {
                    requiredBranches.push(branch.branchId);
                }
            }
        }

        return requiredBranches;
    }

    /**
     * Check if branch can proceed to join
     */
    canBranchProceedToJoin(
        branchId: string,
        context: EnhancedExecutionContext,
        gatewayId: string,
    ): boolean {
        const branch = context.parallelExecution.activeBranches.find(b => b.branchId === branchId);
        if (!branch || branch.status !== "running") return false;

        const joinPoint = context.parallelExecution.joinPoints.find(jp => jp.gatewayId === gatewayId);
        if (!joinPoint) return true; // No join point yet, can proceed

        return joinPoint.requiredBranches.includes(branchId);
    }

    /**
     * Resolve parallel gateway deadlocks
     */
    resolveParallelDeadlock(
        context: EnhancedExecutionContext,
        gatewayId: string,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        
        // Find stuck join points
        const stuckJoinPoint = updatedContext.parallelExecution.joinPoints.find(
            jp => jp.gatewayId === gatewayId && !jp.isReady,
        );

        if (stuckJoinPoint) {
            // Mark missing branches as completed (with failure)
            for (const requiredBranch of stuckJoinPoint.requiredBranches) {
                if (!stuckJoinPoint.completedBranches.includes(requiredBranch)) {
                    stuckJoinPoint.completedBranches.push(requiredBranch);
                    
                    // Mark branch as failed
                    updatedContext.parallelExecution.activeBranches = 
                        updatedContext.parallelExecution.activeBranches.map(branch => {
                            if (branch.branchId === requiredBranch) {
                                return { ...branch, status: "failed" as const, completedAt: new Date() };
                            }
                            return branch;
                        });
                }
            }

            stuckJoinPoint.isReady = true;
        }

        return updatedContext;
    }
}
