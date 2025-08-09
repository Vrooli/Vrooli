/**
 * BPMN Subprocess Handler - Manages subprocess navigation and lifecycle
 * 
 * Handles subprocess execution in BPMN workflows:
 * - Call Activities (calling external routines/processes)
 * - Embedded Subprocesses (nested workflows within the same process)
 * - Subprocess variable scoping and data flow
 * - Subprocess lifecycle management (start, execute, complete, error)
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    SubprocessContext,
    EventSubprocess,
    LocationType,
} from "../../types.js";
import type { BpmnModel } from "./BpmnModel.js";
import { ContextUtils } from "../contextTransformer.js";

/**
 * Utility function for case-insensitive BPMN type comparison
 */
function isElementType(element: any, expectedType: string): boolean {
    if (!element?.$type) return false;
    return element.$type.toLowerCase() === expectedType.toLowerCase();
}

/**
 * Subprocess processing result
 */
export interface SubprocessResult {
    // New locations to navigate to
    nextLocations: AbstractLocation[];
    // Updated context with subprocess state
    updatedContext: EnhancedExecutionContext;
    // Whether subprocess is starting, running, or completing
    subprocessState: "starting" | "running" | "completing" | "completed" | "failed";
    // Subprocess call information (for external integration)
    subprocessCall?: SubprocessCall;
    // Results from completed subprocess
    subprocessResults?: Record<string, unknown>;
}

/**
 * Information about a subprocess call
 */
export interface SubprocessCall {
    subprocessId: string;
    type: "call_activity" | "embedded_subprocess";
    calledRoutineId?: string; // For call activities
    inputVariables: Record<string, unknown>;
    outputMapping?: Record<string, string>;
    inputMapping?: Record<string, string>;
}

/**
 * BPMN Subprocess Handler
 */
export class BpmnSubprocessHandler {

    /**
     * Process subprocess element (call activity or embedded subprocess)
     */
    processSubprocess(
        model: BpmnModel,
        subprocessId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        const subprocess = model.getElementById(subprocessId);
        if (!subprocess) {
            throw new Error(`Subprocess not found: ${subprocessId}`);
        }

        const subprocessType = this.getSubprocessType(subprocess);

        switch (subprocessType) {
            case "call_activity":
                return this.processCallActivity(model, subprocess, currentLocation, context);
            case "embedded_subprocess":
                return this.processEmbeddedSubprocess(model, subprocess, currentLocation, context);
            case "event_subprocess":
                return this.processEventSubprocess(model, subprocess, currentLocation, context);
            case "transaction":
                return this.processTransactionSubprocess(model, subprocess, currentLocation, context);
            case "adhoc":
                return this.processAdhocSubprocess(model, subprocess, currentLocation, context);
            default:
                throw new Error(`Unknown subprocess type: ${subprocessType}`);
        }
    }

    /**
     * Process call activity (calling external routine)
     */
    private processCallActivity(
        model: BpmnModel,
        callActivity: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        const result: SubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            subprocessState: "starting",
        };

        // Check if this call activity is already running
        const existingSubprocess = context.subprocesses.stack.find(
            sp => sp.subprocessId === callActivity.id && sp.status === "running",
        );

        if (existingSubprocess) {
            // Subprocess is running - check for completion
            return this.checkCallActivityCompletion(model, callActivity, existingSubprocess, currentLocation, context);
        } else {
            // Start new call activity
            return this.startCallActivity(model, callActivity, currentLocation, context);
        }
    }

    /**
     * Start call activity execution
     */
    private startCallActivity(
        model: BpmnModel,
        callActivity: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        const callDefinition = this.extractCallActivityDefinition(callActivity);
        
        // Prepare input variables with mapping
        const inputVariables = this.mapInputVariables(
            context.variables, 
            callDefinition.inputMapping,
        );

        // Create subprocess context
        const subprocessContext: SubprocessContext = {
            id: `subprocess_${callActivity.id}_${Date.now()}`,
            subprocessId: callActivity.id,
            parentLocation: currentLocation,
            variables: inputVariables,
            startedAt: new Date(),
            status: "running",
        };

        const updatedContext = ContextUtils.enterSubprocess(context, subprocessContext);

        // Create subprocess call for external processing
        const subprocessCall: SubprocessCall = {
            subprocessId: callActivity.id,
            type: "call_activity",
            calledRoutineId: callDefinition.calledElement,
            inputVariables,
            outputMapping: callDefinition.outputMapping,
            inputMapping: callDefinition.inputMapping,
        };

        // Create waiting location for subprocess completion
        const waitingLocation = model.createAbstractLocation(
            callActivity.id,
            currentLocation.routineId,
            "subprocess_context",
            {
                parentNodeId: callActivity.id,
                subprocessId: callActivity.id,
                metadata: {
                    subprocessType: "call_activity",
                    subprocessContext,
                    callDefinition,
                },
            },
        );

        return {
            nextLocations: [waitingLocation],
            updatedContext,
            subprocessState: "starting",
            subprocessCall,
        };
    }

    /**
     * Check call activity completion
     */
    private checkCallActivityCompletion(
        model: BpmnModel,
        callActivity: any,
        subprocessContext: SubprocessContext,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Check if subprocess has completed (this would be set by external system)
        const isCompleted = subprocessContext.status === "completed";
        const hasFailed = subprocessContext.status === "failed";

        if (isCompleted || hasFailed) {
            // Subprocess completed - process results and continue
            return this.completeCallActivity(model, callActivity, subprocessContext, currentLocation, context);
        } else {
            // Still running - continue waiting
            const waitingLocation = model.createAbstractLocation(
                callActivity.id,
                currentLocation.routineId,
                "subprocess_context",
                {
                    parentNodeId: callActivity.id,
                    subprocessId: callActivity.id,
                    metadata: {
                        subprocessType: "call_activity",
                        subprocessContext,
                    },
                },
            );

            return {
                nextLocations: [waitingLocation],
                updatedContext: context,
                subprocessState: "running",
            };
        }
    }

    /**
     * Complete call activity and process results
     */
    private completeCallActivity(
        model: BpmnModel,
        callActivity: any,
        subprocessContext: SubprocessContext,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        const callDefinition = this.extractCallActivityDefinition(callActivity);
        const updatedContext = ContextUtils.exitSubprocess(context);

        // Map output variables back to parent context
        if (subprocessContext.status === "completed") {
            const outputVariables = this.mapOutputVariables(
                subprocessContext.variables,
                callDefinition.outputMapping,
            );

            // Merge output variables into parent context
            updatedContext.variables = { ...updatedContext.variables, ...outputVariables };
        }

        // Continue to next elements
        const outgoingFlows = model.getOutgoingFlows(callActivity.id);
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

        return {
            nextLocations,
            updatedContext,
            subprocessState: subprocessContext.status === "completed" ? "completed" : "failed",
            subprocessResults: subprocessContext.variables,
        };
    }

    /**
     * Process embedded subprocess (nested workflow)
     */
    private processEmbeddedSubprocess(
        model: BpmnModel,
        subprocess: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Check if subprocess is already running
        const existingSubprocess = context.subprocesses.stack.find(
            sp => sp.subprocessId === subprocess.id && sp.status === "running",
        );

        if (existingSubprocess) {
            // Subprocess is running - navigate within it
            return this.navigateWithinEmbeddedSubprocess(model, subprocess, existingSubprocess, currentLocation, context);
        } else {
            // Start embedded subprocess
            return this.startEmbeddedSubprocess(model, subprocess, currentLocation, context);
        }
    }

    /**
     * Start embedded subprocess execution
     */
    private startEmbeddedSubprocess(
        model: BpmnModel,
        subprocess: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Get subprocess flow elements
        const subprocessElements = model.getSubprocessElements(subprocess.id);
        
        // Find start events within subprocess
        let startEvents = subprocessElements.filter(el => isElementType(el, "bpmn:StartEvent"));
        
        // If no start events found, look for any elements without incoming flows
        if (startEvents.length === 0) {
            startEvents = subprocessElements.filter(element => 
                !model.hasIncomingFlows(element.id),
            );
        }
        
        // If still no elements found, create a default entry point
        if (startEvents.length === 0) {
            // For transaction and adhoc subprocesses, create a synthetic start location
            const startLocation = model.createAbstractLocation(
                subprocess.id,
                currentLocation.routineId,
                "subprocess_context",
                { 
                    parentNodeId: subprocess.id,
                    subprocessId: subprocess.id,
                    metadata: { synthetic: true },
                },
            );
            
            const updatedContext = ContextUtils.enterSubprocess(context, {
                id: `subprocess_${subprocess.id}_${Date.now()}`,
                subprocessId: subprocess.id,
                parentLocation: currentLocation,
                variables: { ...context.variables },
                startedAt: new Date(),
                status: "running",
            });
            
            return {
                nextLocations: [startLocation],
                updatedContext,
                subprocessState: "starting",
            };
        }

        // Create subprocess context with isolated variables
        const subprocessVariables = { ...context.variables }; // Copy parent variables
        const subprocessContext: SubprocessContext = {
            id: `subprocess_${subprocess.id}_${Date.now()}`,
            subprocessId: subprocess.id,
            parentLocation: currentLocation,
            variables: subprocessVariables,
            startedAt: new Date(),
            status: "running",
        };

        const updatedContext = ContextUtils.enterSubprocess(context, subprocessContext);

        // Create location for subprocess start event
        const subprocessStartLocation = model.createAbstractLocation(
            startEvents[0].id,
            currentLocation.routineId,
            "subprocess_context",
            {
                parentNodeId: startEvents[0].id,
                subprocessId: subprocess.id,
                metadata: {
                    subprocessType: "embedded_subprocess",
                    subprocessContext,
                },
            },
        );

        return {
            nextLocations: [subprocessStartLocation],
            updatedContext,
            subprocessState: "starting",
        };
    }

    /**
     * Navigate within embedded subprocess
     */
    private navigateWithinEmbeddedSubprocess(
        model: BpmnModel,
        subprocess: any,
        subprocessContext: SubprocessContext,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Check if we've reached an end event within the subprocess
        const currentElement = model.getElementById(currentLocation.nodeId);
        
        if (currentElement && isElementType(currentElement, "bpmn:EndEvent")) {
            // Subprocess completed
            return this.completeEmbeddedSubprocess(model, subprocess, subprocessContext, currentLocation, context);
        }

        // Continue normal navigation within subprocess
        const outgoingFlows = model.getOutgoingFlows(currentLocation.nodeId);
        const nextLocations: AbstractLocation[] = [];

        for (const flow of outgoingFlows) {
            if (flow.targetRef) {
                const nextLocation = model.createAbstractLocation(
                    (flow.targetRef as any).id,
                    currentLocation.routineId,
                    "subprocess_context",
                    {
                        parentNodeId: (flow.targetRef as any).id,
                        subprocessId: subprocess.id,
                        metadata: {
                            subprocessType: "embedded_subprocess",
                            subprocessContext,
                        },
                    },
                );
                nextLocations.push(nextLocation);
            }
        }

        return {
            nextLocations,
            updatedContext: context,
            subprocessState: "running",
        };
    }

    /**
     * Complete embedded subprocess
     */
    private completeEmbeddedSubprocess(
        model: BpmnModel,
        subprocess: any,
        subprocessContext: SubprocessContext,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Update subprocess status
        subprocessContext.status = "completed";
        
        // Exit subprocess context
        const updatedContext = ContextUtils.exitSubprocess(context);

        // Merge subprocess variables back to parent (embedded subprocesses share scope)
        updatedContext.variables = { ...updatedContext.variables, ...subprocessContext.variables };

        // Continue after subprocess
        const outgoingFlows = model.getOutgoingFlows(subprocess.id);
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

        return {
            nextLocations,
            updatedContext,
            subprocessState: "completed",
            subprocessResults: subprocessContext.variables,
        };
    }

    /**
     * Process event subprocess (placeholder for advanced features)
     */
    private processEventSubprocess(
        model: BpmnModel,
        subprocess: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // Event subprocesses are triggered by events - placeholder implementation
        return {
            nextLocations: [],
            updatedContext: context,
            subprocessState: "running",
        };
    }

    /**
     * Update subprocess status (called externally when subprocess completes)
     */
    updateSubprocessStatus(
        context: EnhancedExecutionContext,
        subprocessId: string,
        status: "completed" | "failed",
        results?: Record<string, unknown>,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        
        // Find and update subprocess in stack
        updatedContext.subprocesses.stack = updatedContext.subprocesses.stack.map(sp => {
            if (sp.subprocessId === subprocessId) {
                return {
                    ...sp,
                    status,
                    variables: results ? { ...sp.variables, ...results } : sp.variables,
                };
            }
            return sp;
        });

        return updatedContext;
    }

    /**
     * Get current subprocess context
     */
    getCurrentSubprocessContext(context: EnhancedExecutionContext): SubprocessContext | null {
        return context.subprocesses.stack.length > 0 
            ? context.subprocesses.stack[context.subprocesses.stack.length - 1] 
            : null;
    }

    /**
     * Check if location is within a subprocess
     */
    isWithinSubprocess(location: AbstractLocation): boolean {
        return location.locationType === "subprocess_context" && !!location.subprocessId;
    }

    /**
     * Process transaction subprocess
     */
    private processTransactionSubprocess(
        model: BpmnModel,
        subprocess: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // For now, treat transaction subprocesses like embedded subprocesses
        // with additional transaction properties
        const result = this.processEmbeddedSubprocess(model, subprocess, currentLocation, context);
        
        // Add transaction-specific metadata
        if (result.nextLocations.length > 0) {
            result.nextLocations[0].metadata = {
                ...result.nextLocations[0].metadata,
                isTransaction: true,
                subprocessType: "transaction",
            };
        }
        
        return {
            ...result,
            subprocessState: "starting",
        };
    }

    /**
     * Process adhoc subprocess
     */
    private processAdhocSubprocess(
        model: BpmnModel,
        subprocess: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): SubprocessResult {
        // For now, treat adhoc subprocesses like embedded subprocesses
        // with additional adhoc properties
        const result = this.processEmbeddedSubprocess(model, subprocess, currentLocation, context);
        
        // Add adhoc-specific metadata
        if (result.nextLocations.length > 0) {
            result.nextLocations[0].metadata = {
                ...result.nextLocations[0].metadata,
                isAdhoc: true,
                subprocessType: "adhoc",
                ordering: subprocess.ordering || "Sequential",
            };
        }
        
        return {
            ...result,
            subprocessState: "starting",
        };
    }

    /**
     * Get subprocess nesting level
     */
    getSubprocessNestingLevel(context: EnhancedExecutionContext): number {
        return context.subprocesses.stack.length;
    }

    // Helper methods

    private getSubprocessType(subprocess: any): string {
        if (isElementType(subprocess, "bpmn:CallActivity")) {
            return "call_activity";
        } else if (isElementType(subprocess, "bpmn:SubProcess")) {
            // Check if it's an event subprocess
            if (subprocess.triggeredByEvent) {
                return "event_subprocess";
            }
            return "embedded_subprocess";
        } else if (isElementType(subprocess, "bpmn:Transaction")) {
            return "transaction";
        } else if (isElementType(subprocess, "bpmn:AdHocSubProcess")) {
            return "adhoc";
        }
        return "unknown";
    }

    private extractCallActivityDefinition(callActivity: any): any {
        return {
            calledElement: callActivity.calledElement || null,
            inputMapping: this.extractIOMapping(callActivity, "input"),
            outputMapping: this.extractIOMapping(callActivity, "output"),
        };
    }

    private extractIOMapping(element: any, direction: "input" | "output"): Record<string, string> {
        const mapping: Record<string, string> = {};
        
        const ioSpecification = element.ioSpecification;
        if (!ioSpecification) return mapping;

        const properties = direction === "input" ? ioSpecification.inputSets : ioSpecification.outputSets;
        if (!properties || !Array.isArray(properties)) return mapping;

        for (const property of properties) {
            if (property.name && property.value) {
                mapping[property.name] = property.value;
            }
        }

        return mapping;
    }

    private mapInputVariables(
        parentVariables: Record<string, unknown>,
        inputMapping?: Record<string, string>,
    ): Record<string, unknown> {
        if (!inputMapping) {
            return { ...parentVariables }; // Copy all variables if no mapping
        }

        const mappedVariables: Record<string, unknown> = {};
        
        for (const [subprocessVar, parentVar] of Object.entries(inputMapping)) {
            if (parentVar in parentVariables) {
                mappedVariables[subprocessVar] = parentVariables[parentVar];
            }
        }

        return mappedVariables;
    }

    private mapOutputVariables(
        subprocessVariables: Record<string, unknown>,
        outputMapping?: Record<string, string>,
    ): Record<string, unknown> {
        if (!outputMapping) {
            return { ...subprocessVariables }; // Copy all variables if no mapping
        }

        const mappedVariables: Record<string, unknown> = {};
        
        for (const [parentVar, subprocessVar] of Object.entries(outputMapping)) {
            if (subprocessVar in subprocessVariables) {
                mappedVariables[parentVar] = subprocessVariables[subprocessVar];
            }
        }

        return mappedVariables;
    }

    /**
     * Complete a subprocess and return next locations
     */
    completeSubprocess(
        model: BpmnModel,
        subprocessId: string,
        location: AbstractLocation,
        context: EnhancedExecutionContext,
    ): { nextLocations: AbstractLocation[]; updatedContext: EnhancedExecutionContext } {
        const result = {
            nextLocations: [] as AbstractLocation[],
            updatedContext: { ...context },
        };

        // Find the subprocess in stack
        const subprocessIndex = result.updatedContext.subprocesses.stack.findIndex(
            sp => sp.subprocessId === subprocessId,
        );

        if (subprocessIndex === -1) {
            return result; // Subprocess not found
        }

        const subprocess = result.updatedContext.subprocesses.stack[subprocessIndex];

        // Mark subprocess as completed and remove from stack
        result.updatedContext.subprocesses.stack.splice(subprocessIndex, 1);

        // Get outgoing flows from the subprocess
        const outgoingFlows = model.getOutgoingFlows(subprocessId);
        for (const flow of outgoingFlows) {
            if (flow.targetRef) {
                const targetLocation = model.createAbstractLocation(
                    flow.targetRef.id,
                    location.routineId,
                    "node",
                    { parentNodeId: flow.targetRef.id },
                );
                result.nextLocations.push(targetLocation);
            }
        }

        return result;
    }

    /**
     * Navigate within a subprocess boundary
     */
    navigateWithinSubprocess(
        model: BpmnModel,
        nodeId: string,
        location: AbstractLocation,
        context: EnhancedExecutionContext,
    ): { nextLocations: AbstractLocation[] } {
        const result = {
            nextLocations: [] as AbstractLocation[],
        };

        // Check if current location is within a subprocess
        const subprocessId = location.metadata?.subprocessId;
        if (!subprocessId) {
            return result;
        }

        // Get current subprocess context
        const subprocessContext = context.subprocesses.stack.find(
            sp => sp.subprocessId === subprocessId,
        );

        if (!subprocessContext) {
            return result;
        }

        // Navigate within subprocess boundaries
        const currentNode = model.getElementById(nodeId);
        
        if (currentNode) {
            const outgoingFlows = model.getOutgoingFlows(nodeId);
            
            if (outgoingFlows.length > 0) {
                for (const flow of outgoingFlows) {
                    if (flow.targetRef) {
                        const targetLocation = model.createAbstractLocation(
                            flow.targetRef.id,
                            location.routineId,
                            "subprocess_context",
                            { 
                                parentNodeId: flow.targetRef.id,
                                subprocessId,
                            },
                        );
                        result.nextLocations.push(targetLocation);
                    }
                }
            } else {
                // If no outgoing flows, this might be an end event
                // Navigate to the subprocess's end location
                const subprocessEndLocation = model.createAbstractLocation(
                    subprocessId,
                    location.routineId,
                    "node",
                    { 
                        parentNodeId: subprocessId,
                        metadata: { completed: true },
                    },
                );
                result.nextLocations.push(subprocessEndLocation);
            }
        } else {
            // Node not found - return empty result (will be handled upstream)
        }

        return result;
    }

    /**
     * Get active subprocesses, optionally filtered by type
     */
    getActiveSubprocesses(
        context: EnhancedExecutionContext,
        type?: string,
    ): Array<{
        id: string;
        subprocessId: string;
        type: string;
        parentProcessId: string;
        status: string;
        startedAt: Date;
    }> {
        let activeSubprocesses = context.subprocesses.stack;

        if (type) {
            activeSubprocesses = activeSubprocesses.filter(sp => (sp as any).type === type);
        }

        return activeSubprocesses.map(sp => ({
            id: sp.id,
            subprocessId: sp.subprocessId,
            type: (sp as any).type || "embedded", // Default type
            parentProcessId: (sp as any).parentProcessId || "",
            status: sp.status,
            startedAt: sp.startedAt,
        }));
    }
}
