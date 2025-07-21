/**
 * BPMN Inclusive Gateway Handler - Manages OR-semantics and multi-path activation
 * 
 * Handles inclusive gateways which represent OR-semantics in BPMN:
 * - Split inclusive gateways: Activate multiple outgoing paths based on conditions
 * - Join inclusive gateways: Wait for all activated paths to complete
 * - Complex condition evaluation for path activation
 * - State tracking for activated and completed paths
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    InclusiveGatewayState,
    LocationType,
} from "../../types.js";
import type { BpmnModel } from "./BpmnModel.js";
import { ContextUtils } from "../contextTransformer.js";

/**
 * Inclusive gateway processing result
 */
export interface InclusiveGatewayResult {
    // New locations to navigate to (multiple for splits based on conditions)
    nextLocations: AbstractLocation[];
    // Updated context with inclusive gateway state
    updatedContext: EnhancedExecutionContext;
    // Whether processing is complete (for joins)
    processingComplete: boolean;
    // Information about path activation/completion
    pathUpdates: PathUpdate[];
}

/**
 * Path activation/completion information
 */
export interface PathUpdate {
    pathId: string;
    gatewayId: string;
    action: "activated" | "completed" | "synchronized";
    condition?: string;
    conditionResult?: boolean;
    timestamp: Date;
}

/**
 * BPMN Inclusive Gateway Handler
 */
export class BpmnInclusiveHandler {

    /**
     * Process inclusive gateway (split or join)
     */
    processInclusiveGateway(
        model: BpmnModel,
        gatewayId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): InclusiveGatewayResult {
        const gateway = model.getElementById(gatewayId);
        if (!gateway) {
            throw new Error(`Inclusive gateway not found: ${gatewayId}`);
        }

        const incomingFlows = model.getIncomingFlows(gatewayId);
        const outgoingFlows = model.getOutgoingFlows(gatewayId);

        // Determine gateway type based on flows
        if (incomingFlows.length === 1 && outgoingFlows.length > 1) {
            // Split gateway (diverging OR)
            return this.processInclusiveSplit(model, gatewayId, outgoingFlows, currentLocation, context);
        } else if (incomingFlows.length > 1 && outgoingFlows.length >= 1) {
            // Join gateway (converging OR)
            return this.processInclusiveJoin(model, gatewayId, incomingFlows, outgoingFlows, currentLocation, context);
        } else {
            // Single flow - treat as pass-through
            return this.processInclusivePassThrough(model, gatewayId, outgoingFlows, currentLocation, context);
        }
    }

    /**
     * Process inclusive split gateway (diverging OR)
     */
    private processInclusiveSplit(
        model: BpmnModel,
        gatewayId: string,
        outgoingFlows: any[],
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): InclusiveGatewayResult {
        const result: InclusiveGatewayResult = {
            nextLocations: [],
            updatedContext: { ...context },
            processingComplete: true,
            pathUpdates: [],
        };

        const splitId = `inclusive_split_${gatewayId}_${Date.now()}`;
        const activatedPaths: string[] = [];
        const evaluatedConditions: Array<{
            conditionId: string;
            expression: string;
            result: boolean;
            evaluatedAt: Date;
        }> = [];

        // Evaluate all outgoing flow conditions
        for (let i = 0; i < outgoingFlows.length; i++) {
            const flow = outgoingFlows[i];
            if (!flow.targetRef) continue;

            const conditionId = `${splitId}_condition_${i}`;
            const flowCondition = this.extractFlowCondition(flow);
            const conditionResult = this.evaluateCondition(flowCondition, context.variables);

            evaluatedConditions.push({
                conditionId,
                expression: flowCondition || "true",
                result: conditionResult,
                evaluatedAt: new Date(),
            });

            if (conditionResult) {
                // Activate this path
                const pathId = `${splitId}_path_${i}`;
                activatedPaths.push(pathId);

                const pathLocation = model.createAbstractLocation(
                    (flow.targetRef as any).id,
                    currentLocation.routineId,
                    "node",
                    {
                        parentNodeId: (flow.targetRef as any).id,
                        metadata: {
                            inclusiveGateway: gatewayId,
                            splitId,
                            pathId,
                            pathIndex: i,
                            activatedPaths: activatedPaths.length,
                            condition: flowCondition,
                        },
                    },
                );

                result.nextLocations.push(pathLocation);

                // Record path activation
                result.pathUpdates.push({
                    pathId,
                    gatewayId,
                    action: "activated",
                    condition: flowCondition,
                    conditionResult: true,
                    timestamp: new Date(),
                });
            }
        }

        // Store inclusive gateway state for potential join processing
        const inclusiveState: InclusiveGatewayState = {
            id: splitId,
            gatewayId,
            evaluatedConditions,
            activatedPaths,
        };

        result.updatedContext.gateways.inclusiveStates.push(inclusiveState);

        // If no paths were activated, that's an error condition
        if (activatedPaths.length === 0) {
            throw new Error(`No paths activated in inclusive gateway ${gatewayId} - all conditions evaluated to false`);
        }

        return result;
    }

    /**
     * Process inclusive join gateway (converging OR)
     */
    private processInclusiveJoin(
        model: BpmnModel,
        gatewayId: string,
        incomingFlows: any[],
        outgoingFlows: any[],
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): InclusiveGatewayResult {
        const result: InclusiveGatewayResult = {
            nextLocations: [],
            updatedContext: { ...context },
            processingComplete: false,
            pathUpdates: [],
        };

        // Find the corresponding split gateway state
        const correspondingSplit = this.findCorrespondingSplit(model, gatewayId, context);
        
        if (!correspondingSplit) {
            // No corresponding split found - this might be a merge from multiple splits
            // For now, treat as immediate continuation
            console.warn(`No corresponding inclusive split found for join gateway ${gatewayId}`);
            return this.processInclusivePassThrough(model, gatewayId, outgoingFlows, currentLocation, context);
        }

        // Track which paths have completed
        const currentPathId = this.extractPathIdFromLocation(currentLocation);
        const completedPaths = this.getCompletedPaths(gatewayId, context);
        
        if (currentPathId && !completedPaths.includes(currentPathId)) {
            completedPaths.push(currentPathId);
            
            result.pathUpdates.push({
                pathId: currentPathId,
                gatewayId,
                action: "completed",
                timestamp: new Date(),
            });
        }

        // Check if all activated paths from the split have completed
        const allActivatedPathsCompleted = correspondingSplit.activatedPaths.every(pathId =>
            completedPaths.includes(pathId),
        );

        if (allActivatedPathsCompleted) {
            // All paths completed - proceed with join
            result.processingComplete = true;

            // Continue with outgoing flows
            for (const flow of outgoingFlows) {
                if (flow.targetRef) {
                    const nextLocation = model.createAbstractLocation(
                        (flow.targetRef as any).id,
                        currentLocation.routineId,
                        "node",
                        { parentNodeId: (flow.targetRef as any).id },
                    );
                    result.nextLocations.push(nextLocation);
                }
            }

            // Clean up completed inclusive state
            result.updatedContext.gateways.inclusiveStates = 
                result.updatedContext.gateways.inclusiveStates.filter(state => state.id !== correspondingSplit.id);

            result.pathUpdates.push({
                pathId: `all_paths_${gatewayId}`,
                gatewayId,
                action: "synchronized",
                timestamp: new Date(),
            });
        } else {
            // Still waiting for other paths - create waiting location
            const waitingLocation = model.createAbstractLocation(
                gatewayId,
                currentLocation.routineId,
                "gateway_evaluation",
                {
                    parentNodeId: gatewayId,
                    metadata: {
                        gatewayType: "inclusive_join",
                        waitingForPaths: correspondingSplit.activatedPaths.filter(pathId => 
                            !completedPaths.includes(pathId),
                        ),
                        completedPaths,
                    },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    /**
     * Process inclusive pass-through (single flow)
     */
    private processInclusivePassThrough(
        model: BpmnModel,
        gatewayId: string,
        outgoingFlows: any[],
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): InclusiveGatewayResult {
        const result: InclusiveGatewayResult = {
            nextLocations: [],
            updatedContext: context,
            processingComplete: true,
            pathUpdates: [],
        };

        // Continue with first available outgoing flow
        if (outgoingFlows.length > 0 && outgoingFlows[0].targetRef) {
            const nextLocation = model.createAbstractLocation(
                (outgoingFlows[0].targetRef as any).id,
                currentLocation.routineId,
                "node",
                { parentNodeId: (outgoingFlows[0].targetRef as any).id },
            );
            result.nextLocations.push(nextLocation);
        }

        return result;
    }

    /**
     * Check for inclusive gateway deadlocks and resolve them
     */
    resolveInclusiveDeadlock(
        context: EnhancedExecutionContext,
        gatewayId: string,
        timeoutMs = 30000,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        const now = new Date();

        // Find stuck inclusive states
        const stuckStates = updatedContext.gateways.inclusiveStates.filter(state => {
            const isOld = (now.getTime() - state.evaluatedConditions[0]?.evaluatedAt.getTime()) > timeoutMs;
            return isOld && state.gatewayId === gatewayId;
        });

        for (const stuckState of stuckStates) {
            console.warn(`Resolving inclusive gateway deadlock for ${gatewayId}, state: ${stuckState.id}`);
            
            // Mark all activated paths as completed to unblock the join
            // This is a fallback mechanism - in production, better error handling would be needed
            updatedContext.gateways.inclusiveStates = 
                updatedContext.gateways.inclusiveStates.filter(state => state.id !== stuckState.id);
        }

        return updatedContext;
    }

    /**
     * Get inclusive gateway execution summary
     */
    getInclusiveGatewaySummary(context: EnhancedExecutionContext): {
        activeGateways: number;
        activePaths: number;
        pendingSynchronizations: number;
    } {
        const activeGateways = new Set(context.gateways.inclusiveStates.map(state => state.gatewayId)).size;
        
        const activePaths = context.gateways.inclusiveStates.reduce(
            (total, state) => total + state.activatedPaths.length, 0,
        );

        const pendingSynchronizations = context.gateways.inclusiveStates.length;

        return {
            activeGateways,
            activePaths,
            pendingSynchronizations,
        };
    }

    // Helper methods

    private extractFlowCondition(flow: any): string | undefined {
        // Extract condition from sequence flow
        if (flow.conditionExpression && flow.conditionExpression.body) {
            return flow.conditionExpression.body.trim();
        }
        
        // For flows without explicit conditions, return undefined (will evaluate to true)
        return undefined;
    }

    private evaluateCondition(condition: string | undefined, variables: Record<string, unknown>): boolean {
        if (!condition) {
            return true; // No condition means always true
        }

        try {
            // Enhanced condition evaluation supporting OR semantics
            
            // Handle multiple conditions with OR operator
            const orConditions = condition.split("||").map(c => c.trim());
            if (orConditions.length > 1) {
                return orConditions.some(cond => this.evaluateSingleCondition(cond, variables));
            }

            // Handle multiple conditions with AND operator
            const andConditions = condition.split("&&").map(c => c.trim());
            if (andConditions.length > 1) {
                return andConditions.every(cond => this.evaluateSingleCondition(cond, variables));
            }

            // Single condition
            return this.evaluateSingleCondition(condition, variables);
        } catch (error) {
            console.warn(`Error evaluating inclusive gateway condition "${condition}":`, error);
            return false; // Fail safe - don't activate path if condition can't be evaluated
        }
    }

    private evaluateSingleCondition(condition: string, variables: Record<string, unknown>): boolean {
        // Simple variable check
        const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
        if (varMatch) {
            return !!variables[varMatch[1]];
        }

        // Equality comparison
        const eqMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*==\s*(.+)$/);
        if (eqMatch) {
            const varName = eqMatch[1];
            const expectedValue = eqMatch[2].replace(/['"]/g, "");
            return variables[varName] == expectedValue;
        }

        // Inequality comparison
        const neqMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*!=\s*(.+)$/);
        if (neqMatch) {
            const varName = neqMatch[1];
            const expectedValue = neqMatch[2].replace(/['"]/g, "");
            return variables[varName] != expectedValue;
        }

        // Numeric comparisons
        const numMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*(>|<|>=|<=)\s*(\d+(?:\.\d+)?)$/);
        if (numMatch) {
            const varName = numMatch[1];
            const operator = numMatch[2];
            const value = parseFloat(numMatch[3]);
            const varValue = Number(variables[varName]);

            if (isNaN(varValue)) return false;

            switch (operator) {
                case ">": return varValue > value;
                case "<": return varValue < value;
                case ">=": return varValue >= value;
                case "<=": return varValue <= value;
            }
        }

        // Array/object checks
        const inMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s+in\s+(.+)$/);
        if (inMatch) {
            const varName = inMatch[1];
            const arrayName = inMatch[2];
            const array = variables[arrayName] as any[];
            return Array.isArray(array) && array.includes(variables[varName]);
        }

        return false;
    }

    private findCorrespondingSplit(
        model: BpmnModel,
        joinGatewayId: string,
        context: EnhancedExecutionContext,
    ): InclusiveGatewayState | null {
        // Look for inclusive states that might correspond to this join
        // This is a simplified approach - in complex workflows, more sophisticated matching would be needed
        
        // Find the most recent inclusive state (LIFO approach)
        const sortedStates = context.gateways.inclusiveStates
            .slice()
            .sort((a, b) => b.evaluatedConditions[0]?.evaluatedAt.getTime() - a.evaluatedConditions[0]?.evaluatedAt.getTime());

        return sortedStates[0] || null;
    }

    private extractPathIdFromLocation(location: AbstractLocation): string | null {
        return location.metadata?.pathId as string || null;
    }

    private getCompletedPaths(gatewayId: string, context: EnhancedExecutionContext): string[] {
        // This would typically be tracked in a more sophisticated way
        // For now, we'll use a simple approach
        const completedPathsKey = `completed_paths_${gatewayId}`;
        return (context.variables[completedPathsKey] as string[]) || [];
    }

    private setCompletedPaths(gatewayId: string, paths: string[], context: EnhancedExecutionContext): void {
        const completedPathsKey = `completed_paths_${gatewayId}`;
        context.variables[completedPathsKey] = paths;
    }

    /**
     * Update path completion status
     */
    updatePathCompletion(
        context: EnhancedExecutionContext,
        gatewayId: string,
        pathId: string,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        const completedPaths = this.getCompletedPaths(gatewayId, updatedContext);
        
        if (!completedPaths.includes(pathId)) {
            completedPaths.push(pathId);
            this.setCompletedPaths(gatewayId, completedPaths, updatedContext);
        }

        return updatedContext;
    }

    /**
     * Check if inclusive gateway is ready to proceed
     */
    isInclusiveGatewayReady(
        gatewayId: string,
        context: EnhancedExecutionContext,
    ): boolean {
        const inclusiveState = context.gateways.inclusiveStates.find(state => state.gatewayId === gatewayId);
        if (!inclusiveState) return true; // No state means ready to proceed

        const completedPaths = this.getCompletedPaths(gatewayId, context);
        return inclusiveState.activatedPaths.every(pathId => completedPaths.includes(pathId));
    }
}
