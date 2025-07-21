/**
 * BPMN Event Subprocess Handler - Manages event-driven subprocess execution
 * 
 * Handles event subprocesses which are triggered by events and run in parallel
 * to the main process flow:
 * - Event subprocess triggering (interrupting and non-interrupting)
 * - Event correlation and subprocess activation
 * - Parallel execution alongside main process
 * - Event subprocess lifecycle management
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    EventSubprocess,
    EventInstance,
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
 * Event subprocess processing result
 */
export interface EventSubprocessResult {
    // New locations to navigate to (event subprocess start events)
    nextLocations: AbstractLocation[];
    // Updated context with event subprocess state
    updatedContext: EnhancedExecutionContext;
    // Whether main process should be interrupted
    shouldInterruptMainProcess: boolean;
    // Information about subprocess activation
    subprocessActivations: SubprocessActivation[];
}

/**
 * Event subprocess activation information
 */
export interface SubprocessActivation {
    subprocessId: string;
    eventType: string;
    triggerEvent: EventInstance;
    interrupting: boolean;
    activatedAt: Date;
}

/**
 * BPMN Event Subprocess Handler
 */
export class BpmnEventSubprocessHandler {

    /**
     * Monitor for event subprocess triggers in current context
     */
    monitorEventSubprocesses(
        model: BpmnModel,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventSubprocessResult {
        const result: EventSubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldInterruptMainProcess: false,
            subprocessActivations: [],
        };

        // Get all event subprocesses in the current process
        const eventSubprocesses = this.findEventSubprocesses(model);

        for (const eventSubprocess of eventSubprocesses) {
            const subprocessResult = this.checkEventSubprocessTrigger(
                model, 
                eventSubprocess, 
                currentLocation, 
                result.updatedContext,
            );

            // Merge results
            result.nextLocations.push(...subprocessResult.nextLocations);
            result.updatedContext = subprocessResult.updatedContext;
            result.subprocessActivations.push(...subprocessResult.subprocessActivations);

            // If any interrupting event subprocess is triggered, interrupt main process
            if (subprocessResult.shouldInterruptMainProcess) {
                result.shouldInterruptMainProcess = true;
            }
        }

        return result;
    }

    /**
     * Process active event subprocess execution
     */
    processEventSubprocess(
        model: BpmnModel,
        subprocessId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventSubprocessResult {
        const result: EventSubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldInterruptMainProcess: false,
            subprocessActivations: [],
        };

        const eventSubprocess = context.subprocesses.eventSubprocesses.find(
            esp => esp.subprocessId === subprocessId,
        );

        if (!eventSubprocess) {
            throw new Error(`Event subprocess not found: ${subprocessId}`);
        }

        switch (eventSubprocess.status) {
            case "monitoring":
                // Subprocess is monitoring for events - check for triggers
                return this.checkEventSubprocessTrigger(model, { id: subprocessId }, currentLocation, context);
                
            case "active":
                // Subprocess is actively executing - continue navigation within it
                return this.navigateWithinEventSubprocess(model, subprocessId, currentLocation, context);
                
            case "completed":
                // Subprocess completed - clean up and potentially resume main process
                return this.completeEventSubprocess(model, subprocessId, currentLocation, context);
                
            default:
                return result;
        }
    }

    /**
     * Check if an event subprocess should be triggered
     */
    private checkEventSubprocessTrigger(
        model: BpmnModel,
        eventSubprocess: { id: string },
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventSubprocessResult {
        const result: EventSubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldInterruptMainProcess: false,
            subprocessActivations: [],
        };

        const subprocessElement = model.getElementById(eventSubprocess.id);
        if (!subprocessElement || !subprocessElement.flowElements) {
            return result;
        }

        // Find start events in the event subprocess
        const startEvents = subprocessElement.flowElements.filter(
            (element: any) => isElementType(element, "bpmn:StartEvent"),
        );

        for (const startEvent of startEvents) {
            const triggerResult = this.checkStartEventTrigger(
                model, 
                startEvent, 
                eventSubprocess.id, 
                currentLocation, 
                result.updatedContext,
            );

            if (triggerResult.isTriggered) {
                // Event subprocess should be activated
                const isInterrupting = this.isEventSubprocessInterrupting(subprocessElement);
                
                // Create event subprocess tracking
                const eventSubprocessTracker: EventSubprocess = {
                    id: `event_subprocess_${eventSubprocess.id}_${Date.now()}`,
                    subprocessId: eventSubprocess.id,
                    triggerEvent: triggerResult.triggerEvent!.id,
                    interrupting: isInterrupting,
                    status: "active",
                    startedAt: new Date(),
                };

                result.updatedContext.subprocesses.eventSubprocesses.push(eventSubprocessTracker);

                // Create location for event subprocess start event
                const subprocessStartLocation = model.createAbstractLocation(
                    startEvent.id,
                    currentLocation.routineId,
                    "subprocess_context",
                    {
                        parentNodeId: startEvent.id,
                        subprocessId: eventSubprocess.id,
                        eventId: startEvent.id,
                        metadata: {
                            subprocessType: "event_subprocess",
                            interrupting: isInterrupting,
                            triggerEvent: triggerResult.triggerEvent,
                            eventSubprocessTracker,
                        },
                    },
                );

                result.nextLocations.push(subprocessStartLocation);

                // Record activation
                result.subprocessActivations.push({
                    subprocessId: eventSubprocess.id,
                    eventType: triggerResult.triggerEvent!.type,
                    triggerEvent: triggerResult.triggerEvent!,
                    interrupting: isInterrupting,
                    activatedAt: new Date(),
                });

                // If interrupting, signal to interrupt main process
                if (isInterrupting) {
                    result.shouldInterruptMainProcess = true;
                }

                // Consume the trigger event
                result.updatedContext = this.consumeTriggerEvent(
                    result.updatedContext, 
                    triggerResult.triggerEvent!,
                );
            }
        }

        return result;
    }

    /**
     * Check if a start event in event subprocess should be triggered
     */
    private checkStartEventTrigger(
        model: BpmnModel,
        startEvent: any,
        subprocessId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        if (!startEvent.eventDefinitions || startEvent.eventDefinitions.length === 0) {
            return { isTriggered: false };
        }

        const eventDefinition = startEvent.eventDefinitions[0];
        const eventType = this.getEventType(eventDefinition);

        switch (eventType) {
            case "timer":
                return this.checkTimerTrigger(eventDefinition, context);
            case "message":
                return this.checkMessageTrigger(eventDefinition, context);
            case "signal":
                return this.checkSignalTrigger(eventDefinition, context);
            case "error":
                return this.checkErrorTrigger(eventDefinition, context);
            case "escalation":
                return this.checkEscalationTrigger(eventDefinition, context);
            case "conditional":
                return this.checkConditionalTrigger(eventDefinition, context);
            default:
                return { isTriggered: false };
        }
    }

    /**
     * Navigate within an active event subprocess
     */
    private navigateWithinEventSubprocess(
        model: BpmnModel,
        subprocessId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventSubprocessResult {
        const result: EventSubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldInterruptMainProcess: false,
            subprocessActivations: [],
        };

        // Check if we've reached an end event within the event subprocess
        const currentElement = model.getElementById(currentLocation.nodeId);
        
        if (currentElement && isElementType(currentElement, "bpmn:EndEvent")) {
            // Event subprocess completed
            return this.completeEventSubprocess(model, subprocessId, currentLocation, context);
        }

        // Continue normal navigation within event subprocess
        const outgoingFlows = model.getOutgoingFlows(currentLocation.nodeId);
        
        for (const flow of outgoingFlows) {
            if (flow.targetRef) {
                const nextLocation = model.createAbstractLocation(
                    (flow.targetRef as any).id,
                    currentLocation.routineId,
                    "subprocess_context",
                    {
                        parentNodeId: (flow.targetRef as any).id,
                        subprocessId,
                        metadata: {
                            subprocessType: "event_subprocess",
                        },
                    },
                );
                result.nextLocations.push(nextLocation);
            }
        }

        return result;
    }

    /**
     * Complete event subprocess execution
     */
    private completeEventSubprocess(
        model: BpmnModel,
        subprocessId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventSubprocessResult {
        const result: EventSubprocessResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldInterruptMainProcess: false,
            subprocessActivations: [],
        };

        // Find and update event subprocess status
        const eventSubprocessIndex = result.updatedContext.subprocesses.eventSubprocesses.findIndex(
            esp => esp.subprocessId === subprocessId && esp.status === "active",
        );

        if (eventSubprocessIndex >= 0) {
            result.updatedContext.subprocesses.eventSubprocesses[eventSubprocessIndex].status = "completed";
        }

        // For non-interrupting event subprocesses, main process continues normally
        // For interrupting event subprocesses, this completion might allow main process to resume
        // (depending on the specific BPMN semantics needed)

        return result;
    }

    /**
     * Get event subprocess execution summary
     */
    getEventSubprocessSummary(context: EnhancedExecutionContext): {
        monitoring: number;
        active: number;
        completed: number;
        interrupting: number;
        nonInterrupting: number;
    } {
        const eventSubprocesses = context.subprocesses.eventSubprocesses;
        
        return {
            monitoring: eventSubprocesses.filter(esp => esp.status === "monitoring").length,
            active: eventSubprocesses.filter(esp => esp.status === "active").length,
            completed: eventSubprocesses.filter(esp => esp.status === "completed").length,
            interrupting: eventSubprocesses.filter(esp => esp.interrupting).length,
            nonInterrupting: eventSubprocesses.filter(esp => !esp.interrupting).length,
        };
    }

    // Helper methods

    private findEventSubprocesses(model: BpmnModel): Array<{ id: string }> {
        // Find all event subprocesses in the current process
        const processes = model.getAllProcesses();
        const eventSubprocesses: Array<{ id: string }> = [];

        for (const process of processes) {
            if (process.flowElements) {
                for (const element of process.flowElements) {
                    if (isElementType(element, "bpmn:SubProcess") && element.triggeredByEvent) {
                        eventSubprocesses.push({ id: element.id });
                    }
                }
            }
        }

        return eventSubprocesses;
    }

    private isEventSubprocessInterrupting(subprocessElement: any): boolean {
        // Check if event subprocess is interrupting (default is true)
        return subprocessElement.cancelActivity !== false;
    }

    private getEventType(eventDefinition: any): string {
        if (eventDefinition.$type.includes("Timer")) return "timer";
        if (eventDefinition.$type.includes("Message")) return "message";
        if (eventDefinition.$type.includes("Signal")) return "signal";
        if (eventDefinition.$type.includes("Error")) return "error";
        if (eventDefinition.$type.includes("Escalation")) return "escalation";
        if (eventDefinition.$type.includes("Conditional")) return "conditional";
        return "unknown";
    }

    private checkTimerTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        const timerDef = eventDefinition.timeDuration || eventDefinition.timeDate || eventDefinition.timeCycle;
        if (!timerDef) return { isTriggered: false };

        // Check if timer has expired
        const now = new Date();
        const timerEvents = context.events.timers.filter(timer => timer.expiresAt <= now);
        
        if (timerEvents.length > 0) {
            const triggerEvent: EventInstance = {
                id: `event_subprocess_timer_${Date.now()}`,
                eventId: timerEvents[0].eventId,
                type: "timer",
                payload: { expiredAt: now.toISOString() },
                firedAt: now,
                source: "event_subprocess",
            };
            return { isTriggered: true, triggerEvent };
        }

        return { isTriggered: false };
    }

    private checkMessageTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        const messageRef = eventDefinition.messageRef?.name || eventDefinition.messageRef?.id;
        if (!messageRef) return { isTriggered: false };

        // Check for matching message
        const matchingMessage = context.external.messageEvents.find(msg => 
            msg.messageRef === messageRef && msg.receivedAt,
        );

        if (matchingMessage) {
            const triggerEvent: EventInstance = {
                id: `event_subprocess_message_${Date.now()}`,
                eventId: matchingMessage.id,
                type: "message",
                payload: matchingMessage.payload || {},
                firedAt: new Date(),
                source: "event_subprocess",
            };
            return { isTriggered: true, triggerEvent };
        }

        return { isTriggered: false };
    }

    private checkSignalTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        const signalRef = eventDefinition.signalRef?.name || eventDefinition.signalRef?.id;
        if (!signalRef) return { isTriggered: false };

        // Check for matching signal
        const matchingSignal = context.external.signalEvents.find(signal => 
            signal.signalRef === signalRef,
        );

        if (matchingSignal) {
            const triggerEvent: EventInstance = {
                id: `event_subprocess_signal_${Date.now()}`,
                eventId: matchingSignal.id,
                type: "signal",
                payload: matchingSignal.payload || {},
                firedAt: new Date(),
                source: "event_subprocess",
            };
            return { isTriggered: true, triggerEvent };
        }

        return { isTriggered: false };
    }

    private checkErrorTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        const errorCode = eventDefinition.errorRef?.errorCode;
        
        // Check for matching error in context
        const errors = (context.variables.errors as any[]) || [];
        const matchingError = errorCode 
            ? errors.find(error => error.code === errorCode)
            : errors[0]; // Any error if no specific code

        if (matchingError) {
            const triggerEvent: EventInstance = {
                id: `event_subprocess_error_${Date.now()}`,
                eventId: matchingError.id || `error_${Date.now()}`,
                type: "error",
                payload: matchingError,
                firedAt: new Date(),
                source: "event_subprocess",
            };
            return { isTriggered: true, triggerEvent };
        }

        return { isTriggered: false };
    }

    private checkEscalationTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        // Escalation events are triggered by escalation throws
        const escalationCode = eventDefinition.escalationRef?.escalationCode;
        
        // Check for matching escalation in fired events
        const matchingEscalation = context.events.fired.find(event => 
            event.type === "escalation" && 
            (!escalationCode || event.payload.escalationCode === escalationCode),
        );

        if (matchingEscalation) {
            return { isTriggered: true, triggerEvent: matchingEscalation };
        }

        return { isTriggered: false };
    }

    private checkConditionalTrigger(
        eventDefinition: any, 
        context: EnhancedExecutionContext,
    ): { isTriggered: boolean; triggerEvent?: EventInstance } {
        const condition = eventDefinition.condition?.body;
        if (!condition) return { isTriggered: false };

        // Evaluate condition
        const conditionResult = this.evaluateCondition(condition, context.variables);
        
        if (conditionResult) {
            const triggerEvent: EventInstance = {
                id: `event_subprocess_conditional_${Date.now()}`,
                eventId: `conditional_${Date.now()}`,
                type: "conditional",
                payload: { condition },
                firedAt: new Date(),
                source: "event_subprocess",
            };
            return { isTriggered: true, triggerEvent };
        }

        return { isTriggered: false };
    }

    private evaluateCondition(condition: string, variables: Record<string, unknown>): boolean {
        try {
            // Simple condition evaluation
            const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
            if (varMatch) {
                return !!variables[varMatch[1]];
            }

            const comparisonMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*(==|!=)\s*(.+)$/);
            if (comparisonMatch) {
                const varName = comparisonMatch[1];
                const operator = comparisonMatch[2];
                const value = comparisonMatch[3].replace(/['"]/g, "");
                const varValue = variables[varName];

                return operator === "==" ? varValue == value : varValue != value;
            }

            return false;
        } catch {
            return false;
        }
    }

    private consumeTriggerEvent(
        context: EnhancedExecutionContext, 
        triggerEvent: EventInstance,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };

        // Remove consumed events from context
        switch (triggerEvent.type) {
            case "message":
                updatedContext.external.messageEvents = updatedContext.external.messageEvents
                    .filter(msg => msg.id !== triggerEvent.eventId);
                break;
            case "signal":
                updatedContext.external.signalEvents = updatedContext.external.signalEvents
                    .filter(signal => signal.id !== triggerEvent.eventId);
                break;
            case "timer":
                updatedContext.events.timers = updatedContext.events.timers
                    .filter(timer => timer.eventId !== triggerEvent.eventId);
                break;
        }

        return updatedContext;
    }

    /**
     * Activate event subprocess monitoring
     */
    activateEventSubprocessMonitoring(
        context: EnhancedExecutionContext,
        subprocessId: string,
        triggerEvent: string,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        
        const eventSubprocess: EventSubprocess = {
            id: `event_subprocess_monitor_${subprocessId}_${Date.now()}`,
            subprocessId,
            triggerEvent,
            interrupting: true, // Default to interrupting
            status: "monitoring",
        };

        updatedContext.subprocesses.eventSubprocesses.push(eventSubprocess);
        return updatedContext;
    }

    /**
     * Deactivate event subprocess monitoring
     */
    deactivateEventSubprocessMonitoring(
        context: EnhancedExecutionContext,
        subprocessId: string,
    ): EnhancedExecutionContext {
        const updatedContext = { ...context };
        
        updatedContext.subprocesses.eventSubprocesses = 
            updatedContext.subprocesses.eventSubprocesses.filter(
                esp => esp.subprocessId !== subprocessId || esp.status !== "monitoring",
            );

        return updatedContext;
    }
}
