/**
 * BPMN Advanced Activity Handler - Manages complex activity characteristics
 * 
 * Handles advanced BPMN activity features:
 * - Multi-instance activities (parallel and sequential execution over collections)
 * - Loop characteristics (repeat activities based on conditions)
 * - Compensation handlers (undo/rollback logic for activities)
 * - Standard loop characteristics (while/until loops)
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    LocationType,
} from "../../types.js";
import type { BpmnModel } from "./BpmnModel.js";
import { ContextUtils } from "../contextTransformer.js";

/**
 * Multi-instance configuration
 */
export interface MultiInstanceConfig {
    isSequential: boolean; // true for sequential, false for parallel
    loopCardinality?: number; // Fixed number of instances
    collection?: string; // Variable name for collection
    elementVariable?: string; // Variable name for current element
    completionCondition?: string; // Early completion condition
}

/**
 * Loop characteristics configuration
 */
export interface LoopConfig {
    testBefore: boolean; // true for while, false for do-while
    loopCondition: string; // Loop continuation condition
    loopMaximum?: number; // Maximum iterations to prevent infinite loops
}

/**
 * Multi-instance execution state
 */
export interface MultiInstanceState {
    id: string;
    activityId: string;
    isSequential: boolean;
    totalInstances: number;
    currentInstance: number;
    completedInstances: number;
    instanceResults: Array<{ instanceId: string; result: unknown; status: "running" | "completed" | "failed" }>;
    collection?: unknown[];
    elementVariable?: string;
    startedAt: Date;
    status: "running" | "completed" | "failed";
}

/**
 * Loop execution state
 */
export interface LoopState {
    id: string;
    activityId: string;
    currentIteration: number;
    maxIterations: number;
    testBefore: boolean;
    loopCondition: string;
    startedAt: Date;
    status: "running" | "completed" | "failed";
}

/**
 * Advanced activity processing result
 */
export interface AdvancedActivityResult {
    // New locations to navigate to
    nextLocations: AbstractLocation[];
    // Updated context with activity state
    updatedContext: EnhancedExecutionContext;
    // Whether activity is complete or needs more processing
    activityComplete: boolean;
    // Instance information for tracking
    instanceInfo?: {
        type: "multi_instance" | "loop" | "compensation";
        instanceId?: string;
        currentIteration?: number;
        totalIterations?: number;
    };
}

/**
 * BPMN Advanced Activity Handler
 */
export class BpmnAdvancedActivityHandler {

    /**
     * Process activity with advanced characteristics
     */
    processAdvancedActivity(
        model: BpmnModel,
        activityId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const activity = model.getElementById(activityId);
        if (!activity) {
            throw new Error(`Activity not found: ${activityId}`);
        }

        // Check for multi-instance characteristics
        const multiInstanceConfig = this.extractMultiInstanceConfig(activity);
        if (multiInstanceConfig) {
            return this.processMultiInstanceActivity(model, activityId, multiInstanceConfig, currentLocation, context);
        }

        // Check for loop characteristics
        const loopConfig = this.extractLoopConfig(activity);
        if (loopConfig) {
            return this.processLoopActivity(model, activityId, loopConfig, currentLocation, context);
        }

        // Check for compensation handlers
        if (this.hasCompensationHandler(activity)) {
            return this.processCompensationActivity(model, activityId, currentLocation, context);
        }

        // No advanced characteristics - return normal processing indicator
        return {
            nextLocations: [],
            updatedContext: context,
            activityComplete: false,
        };
    }

    /**
     * Process multi-instance activity
     */
    private processMultiInstanceActivity(
        model: BpmnModel,
        activityId: string,
        config: MultiInstanceConfig,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const result: AdvancedActivityResult = {
            nextLocations: [],
            updatedContext: { ...context },
            activityComplete: false,
        };

        // Check if multi-instance is already running
        const existingState = this.getMultiInstanceState(activityId, context);
        
        if (existingState) {
            return this.continueMultiInstanceExecution(model, activityId, existingState, currentLocation, context);
        } else {
            return this.startMultiInstanceExecution(model, activityId, config, currentLocation, context);
        }
    }

    /**
     * Start multi-instance execution
     */
    private startMultiInstanceExecution(
        model: BpmnModel,
        activityId: string,
        config: MultiInstanceConfig,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const result: AdvancedActivityResult = {
            nextLocations: [],
            updatedContext: { ...context },
            activityComplete: false,
        };

        // Determine number of instances
        let totalInstances = config.loopCardinality || 1;
        let collection: unknown[] = [];

        if (config.collection) {
            collection = (context.variables[config.collection] as unknown[]) || [];
            totalInstances = collection.length;
        }

        if (totalInstances === 0) {
            // No instances to execute - complete immediately
            result.activityComplete = true;
            result.nextLocations = this.getNextLocationsAfterActivity(model, activityId, currentLocation);
            return result;
        }

        // Create multi-instance state
        const multiInstanceState: MultiInstanceState = {
            id: `multi_instance_${activityId}_${Date.now()}`,
            activityId,
            isSequential: config.isSequential,
            totalInstances,
            currentInstance: 0,
            completedInstances: 0,
            instanceResults: [],
            collection,
            elementVariable: config.elementVariable,
            startedAt: new Date(),
            status: "running",
        };

        // Store state in context
        if (!result.updatedContext.variables.multiInstanceStates) {
            result.updatedContext.variables.multiInstanceStates = [];
        }
        (result.updatedContext.variables.multiInstanceStates as MultiInstanceState[]).push(multiInstanceState);

        if (config.isSequential) {
            // Sequential: Start first instance
            return this.startSequentialInstance(model, activityId, multiInstanceState, 0, currentLocation, result.updatedContext);
        } else {
            // Parallel: Start all instances
            return this.startParallelInstances(model, activityId, multiInstanceState, currentLocation, result.updatedContext);
        }
    }

    /**
     * Start sequential instance
     */
    private startSequentialInstance(
        model: BpmnModel,
        activityId: string,
        state: MultiInstanceState,
        instanceIndex: number,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const instanceId = `${state.id}_instance_${instanceIndex}`;
        
        // Set up instance variables
        const instanceContext = { ...context };
        if (state.elementVariable && state.collection && state.collection[instanceIndex]) {
            instanceContext.variables[state.elementVariable] = state.collection[instanceIndex];
        }
        instanceContext.variables.loopCounter = instanceIndex;

        // Create instance tracking
        state.instanceResults.push({
            instanceId,
            result: undefined,
            status: "running",
        });
        state.currentInstance = instanceIndex;

        // Create location for instance execution
        const instanceLocation = model.createAbstractLocation(
            activityId,
            currentLocation.routineId,
            "multi_instance_execution",
            {
                parentNodeId: activityId,
                metadata: {
                    multiInstanceId: state.id,
                    instanceId,
                    instanceIndex,
                    executionType: "sequential",
                    instanceContext,
                },
            },
        );

        return {
            nextLocations: [instanceLocation],
            updatedContext: instanceContext,
            activityComplete: false,
            instanceInfo: {
                type: "multi_instance",
                instanceId,
                currentIteration: instanceIndex + 1,
                totalIterations: state.totalInstances,
            },
        };
    }

    /**
     * Start parallel instances
     */
    private startParallelInstances(
        model: BpmnModel,
        activityId: string,
        state: MultiInstanceState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const result: AdvancedActivityResult = {
            nextLocations: [],
            updatedContext: context,
            activityComplete: false,
        };

        // Create locations for all instances
        for (let i = 0; i < state.totalInstances; i++) {
            const instanceId = `${state.id}_instance_${i}`;
            
            // Set up instance variables
            const instanceContext = { ...context };
            if (state.elementVariable && state.collection && state.collection[i]) {
                instanceContext.variables[state.elementVariable] = state.collection[i];
            }
            instanceContext.variables.loopCounter = i;

            // Create instance tracking
            state.instanceResults.push({
                instanceId,
                result: undefined,
                status: "running",
            });

            // Create location for instance execution
            const instanceLocation = model.createAbstractLocation(
                activityId,
                currentLocation.routineId,
                "multi_instance_execution",
                {
                    parentNodeId: activityId,
                    metadata: {
                        multiInstanceId: state.id,
                        instanceId,
                        instanceIndex: i,
                        executionType: "parallel",
                        instanceContext,
                    },
                },
            );

            result.nextLocations.push(instanceLocation);
        }

        return result;
    }

    /**
     * Continue multi-instance execution
     */
    private continueMultiInstanceExecution(
        model: BpmnModel,
        activityId: string,
        state: MultiInstanceState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        // Check if instance just completed
        const metadata = currentLocation.metadata;
        if (metadata?.instanceId) {
            this.markInstanceComplete(state, metadata.instanceId as string, metadata.instanceResult);
        }

        // Check completion condition
        if (this.checkMultiInstanceCompletion(state, context)) {
            return this.completeMultiInstanceActivity(model, activityId, state, currentLocation, context);
        }

        // For sequential: start next instance if current completed
        if (state.isSequential && state.currentInstance < state.totalInstances - 1) {
            return this.startSequentialInstance(model, activityId, state, state.currentInstance + 1, currentLocation, context);
        }

        // Still running - return waiting state
        const waitingLocation = model.createAbstractLocation(
            activityId,
            currentLocation.routineId,
            "multi_instance_waiting",
            {
                parentNodeId: activityId,
                metadata: {
                    multiInstanceId: state.id,
                    waitingFor: state.isSequential ? "next_instance" : "parallel_completion",
                },
            },
        );

        return {
            nextLocations: [waitingLocation],
            updatedContext: context,
            activityComplete: false,
        };
    }

    /**
     * Process loop activity
     */
    private processLoopActivity(
        model: BpmnModel,
        activityId: string,
        config: LoopConfig,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const existingState = this.getLoopState(activityId, context);
        
        if (existingState) {
            return this.continueLoopExecution(model, activityId, existingState, currentLocation, context);
        } else {
            return this.startLoopExecution(model, activityId, config, currentLocation, context);
        }
    }

    /**
     * Start loop execution
     */
    private startLoopExecution(
        model: BpmnModel,
        activityId: string,
        config: LoopConfig,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        const loopState: LoopState = {
            id: `loop_${activityId}_${Date.now()}`,
            activityId,
            currentIteration: 0,
            maxIterations: config.loopMaximum || 1000, // Safety limit
            testBefore: config.testBefore,
            loopCondition: config.loopCondition,
            startedAt: new Date(),
            status: "running",
        };

        // Store state in context
        if (!context.variables.loopStates) {
            context.variables.loopStates = [];
        }
        (context.variables.loopStates as LoopState[]).push(loopState);

        // Check condition before first iteration if testBefore
        if (config.testBefore && !this.evaluateLoopCondition(config.loopCondition, context.variables)) {
            // Condition false - don't execute loop
            return this.completeLoopActivity(model, activityId, loopState, currentLocation, context);
        }

        // Execute first iteration
        return this.executeLoopIteration(model, activityId, loopState, currentLocation, context);
    }

    /**
     * Continue loop execution
     */
    private continueLoopExecution(
        model: BpmnModel,
        activityId: string,
        state: LoopState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        state.currentIteration++;

        // Check maximum iterations
        if (state.currentIteration >= state.maxIterations) {
            console.warn(`Loop maximum iterations reached for activity ${activityId}`);
            return this.completeLoopActivity(model, activityId, state, currentLocation, context);
        }

        // Check loop condition
        const shouldContinue = this.evaluateLoopCondition(state.loopCondition, context.variables);
        
        if (!shouldContinue) {
            return this.completeLoopActivity(model, activityId, state, currentLocation, context);
        }

        // Continue with next iteration
        return this.executeLoopIteration(model, activityId, state, currentLocation, context);
    }

    /**
     * Process compensation activity
     */
    private processCompensationActivity(
        model: BpmnModel,
        activityId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        // Compensation activities are triggered by compensation events
        // This is a placeholder for compensation logic
        console.log(`Processing compensation for activity: ${activityId}`);
        
        return {
            nextLocations: this.getNextLocationsAfterActivity(model, activityId, currentLocation),
            updatedContext: context,
            activityComplete: true,
        };
    }

    // Helper methods

    private extractMultiInstanceConfig(activity: any): MultiInstanceConfig | null {
        const loopCharacteristics = activity.loopCharacteristics;
        if (!loopCharacteristics || loopCharacteristics.$type !== "bpmn:MultiInstanceLoopCharacteristics") {
            return null;
        }

        return {
            isSequential: loopCharacteristics.isSequential === true,
            loopCardinality: loopCharacteristics.loopCardinality?.body ? 
                parseInt(loopCharacteristics.loopCardinality.body) : undefined,
            collection: loopCharacteristics.loopDataInputRef?.name || 
                loopCharacteristics.inputDataItem?.name,
            elementVariable: loopCharacteristics.inputDataItem?.name ||
                loopCharacteristics.loopDataInputRef?.itemSubjectRef?.name,
            completionCondition: loopCharacteristics.completionCondition?.body,
        };
    }

    private extractLoopConfig(activity: any): LoopConfig | null {
        const loopCharacteristics = activity.loopCharacteristics;
        if (!loopCharacteristics || loopCharacteristics.$type !== "bpmn:StandardLoopCharacteristics") {
            return null;
        }

        return {
            testBefore: loopCharacteristics.testBefore !== false, // Default to true
            loopCondition: loopCharacteristics.loopCondition?.body || "true",
            loopMaximum: loopCharacteristics.loopMaximum ? 
                parseInt(loopCharacteristics.loopMaximum) : 1000,
        };
    }

    private hasCompensationHandler(activity: any): boolean {
        return activity.isForCompensation === true;
    }

    private getMultiInstanceState(activityId: string, context: EnhancedExecutionContext): MultiInstanceState | null {
        const states = (context.variables.multiInstanceStates as MultiInstanceState[]) || [];
        return states.find(state => state.activityId === activityId && state.status === "running") || null;
    }

    private getLoopState(activityId: string, context: EnhancedExecutionContext): LoopState | null {
        const states = (context.variables.loopStates as LoopState[]) || [];
        return states.find(state => state.activityId === activityId && state.status === "running") || null;
    }

    private markInstanceComplete(state: MultiInstanceState, instanceId: string, result?: unknown): void {
        const instance = state.instanceResults.find(inst => inst.instanceId === instanceId);
        if (instance) {
            instance.status = "completed";
            instance.result = result;
            state.completedInstances++;
        }
    }

    private checkMultiInstanceCompletion(state: MultiInstanceState, context: EnhancedExecutionContext): boolean {
        // Check if all instances completed
        if (state.completedInstances >= state.totalInstances) {
            return true;
        }

        // Check completion condition if specified
        // This would require evaluating the completion condition expression
        return false;
    }

    private completeMultiInstanceActivity(
        model: BpmnModel,
        activityId: string,
        state: MultiInstanceState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        state.status = "completed";

        // Aggregate results if needed
        const results = state.instanceResults.map(inst => inst.result);
        context.variables[`${activityId}_results`] = results;

        return {
            nextLocations: this.getNextLocationsAfterActivity(model, activityId, currentLocation),
            updatedContext: context,
            activityComplete: true,
        };
    }

    private executeLoopIteration(
        model: BpmnModel,
        activityId: string,
        state: LoopState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        // Set loop counter variable
        context.variables.loopCounter = state.currentIteration;

        // Create location for loop iteration
        const iterationLocation = model.createAbstractLocation(
            activityId,
            currentLocation.routineId,
            "loop_execution",
            {
                parentNodeId: activityId,
                metadata: {
                    loopId: state.id,
                    iteration: state.currentIteration,
                },
            },
        );

        return {
            nextLocations: [iterationLocation],
            updatedContext: context,
            activityComplete: false,
            instanceInfo: {
                type: "loop",
                currentIteration: state.currentIteration + 1,
            },
        };
    }

    private completeLoopActivity(
        model: BpmnModel,
        activityId: string,
        state: LoopState,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): AdvancedActivityResult {
        state.status = "completed";

        return {
            nextLocations: this.getNextLocationsAfterActivity(model, activityId, currentLocation),
            updatedContext: context,
            activityComplete: true,
        };
    }

    private evaluateLoopCondition(condition: string, variables: Record<string, unknown>): boolean {
        try {
            // Simple condition evaluation - can be extended for more complex expressions
            const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
            if (varMatch) {
                return !!variables[varMatch[1]];
            }

            const comparisonMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\\s*(==|!=|>|<|>=|<=)\\s*(.+)$/);
            if (comparisonMatch) {
                const varName = comparisonMatch[1];
                const operator = comparisonMatch[2];
                const value = comparisonMatch[3].replace(/['\"]/g, "");
                const varValue = variables[varName];

                switch (operator) {
                    case "==": return varValue == value;
                    case "!=": return varValue != value;
                    case ">": return Number(varValue) > Number(value);
                    case "<": return Number(varValue) < Number(value);
                    case ">=": return Number(varValue) >= Number(value);
                    case "<=": return Number(varValue) <= Number(value);
                }
            }

            return true; // Default to continue
        } catch {
            return false;
        }
    }

    private getNextLocationsAfterActivity(
        model: BpmnModel,
        activityId: string,
        currentLocation: AbstractLocation,
    ): AbstractLocation[] {
        const outgoingFlows = model.getOutgoingFlows(activityId);
        const nextLocations: AbstractLocation[] = [];

        for (const flow of outgoingFlows) {
            if (flow.targetRef) {
                const nextLocation = model.createAbstractLocation(
                    (flow.targetRef as any).id,
                    currentLocation.routineId,
                    "node",
                    { parentNodeId: (flow.targetRef as any).id },
                );
                nextLocations.push(nextLocation);
            }
        }

        return nextLocations;
    }
}
