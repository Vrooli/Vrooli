/**
 * BPMN Intermediate Event Handler - Manages intermediate throw/catch events
 * 
 * Handles standalone intermediate events in BPMN flows:
 * - Intermediate catch events (waiting for signals, messages, timers)
 * - Intermediate throw events (generating signals, messages, errors)
 * - Event correlation and scope management
 * - Link events for process flow control
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    IntermediateEvent,
    EventInstance,
    MessageEvent,
    SignalEvent,
    TimerEvent,
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
 * Intermediate event processing result
 */
export interface IntermediateEventResult {
    // New locations to navigate to
    nextLocations: AbstractLocation[];
    // Updated context with event state changes
    updatedContext: EnhancedExecutionContext;
    // Whether processing should wait (for catch events)
    shouldWait: boolean;
    // Events thrown to external systems
    thrownEvents: EventInstance[];
    // Events caught and consumed
    caughtEvents: EventInstance[];
    // Convenience properties for backward compatibility
    eventThrown?: EventInstance;
    waitingForEvent?: {
        type: string;
        [key: string]: any;
    };
    eventCaught?: EventInstance;
}

/**
 * BPMN Intermediate Event Handler
 */
export class BpmnIntermediateHandler {

    /**
     * Process intermediate event (catch or throw)
     */
    processIntermediateEvent(
        model: BpmnModel,
        eventId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const event = model.getElementById(eventId);
        if (!event) {
            throw new Error(`Intermediate event not found: ${eventId}`);
        }

        const eventType = this.getIntermediateEventType(event);
        const isThrowEvent = this.isThrowEvent(event);

        if (isThrowEvent) {
            return this.processThrowEvent(model, event, eventType, currentLocation, context);
        } else {
            return this.processCatchEvent(model, event, eventType, currentLocation, context);
        }
    }

    /**
     * Process intermediate throw event
     */
    private processThrowEvent(
        model: BpmnModel,
        event: any,
        eventType: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: false,
            thrownEvents: [],
            caughtEvents: [],
        };

        switch (eventType) {
            case "signal":
                return this.processSignalThrowEvent(model, event, currentLocation, context);
            case "message":
                return this.processMessageThrowEvent(model, event, currentLocation, context);
            case "error":
                return this.processErrorThrowEvent(model, event, currentLocation, context);
            case "escalation":
                return this.processEscalationThrowEvent(model, event, currentLocation, context);
            case "compensation":
                return this.processCompensationThrowEvent(model, event, currentLocation, context);
            case "link":
                return this.processLinkThrowEvent(model, event, currentLocation, context);
            default:
                // Unknown throw event - treat as no-op and continue
                return this.continueAfterEvent(model, event, currentLocation, context);
        }
    }

    /**
     * Process intermediate catch event
     */
    private processCatchEvent(
        model: BpmnModel,
        event: any,
        eventType: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        switch (eventType) {
            case "timer":
                return this.processTimerCatchEvent(model, event, currentLocation, context);
            case "signal":
                return this.processSignalCatchEvent(model, event, currentLocation, context);
            case "message":
                return this.processMessageCatchEvent(model, event, currentLocation, context);
            case "conditional":
                return this.processConditionalCatchEvent(model, event, currentLocation, context);
            case "link":
                return this.processLinkCatchEvent(model, event, currentLocation, context);
            default:
                // Unknown catch event - continue immediately
                return this.continueAfterEvent(model, event, currentLocation, context);
        }
    }

    // Throw Event Implementations

    /**
     * Process signal throw event
     */
    private processSignalThrowEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: false,
            thrownEvents: [],
            caughtEvents: [],
        };

        const signalDefinition = this.extractSignalDefinition(event);

        // Create signal event instance
        const signalEvent: EventInstance = {
            id: `signal_${event.id}_${Date.now()}`,
            eventId: event.id,
            type: "signal",
            payload: {
                signalRef: signalDefinition.signalRef,
                scope: signalDefinition.scope || "global",
                data: context.variables, // Pass current variables as signal data
            },
            firedAt: new Date(),
            source: "intermediate_throw",
        };

        result.thrownEvents.push(signalEvent);
        result.eventThrown = signalEvent; // Add convenience property
        result.updatedContext = ContextUtils.fireEvent(result.updatedContext, signalEvent);

        // Add signal to external events for other processes to catch
        const externalSignal: SignalEvent = {
            id: signalEvent.id,
            signalRef: signalDefinition.signalRef,
            scope: signalDefinition.scope || "global",
            payload: signalEvent.payload,
            propagatedAt: new Date(),
        };

        result.updatedContext.external.signalEvents.push(externalSignal);

        // Continue to next element
        const continueResult = this.continueAfterEvent(model, event, currentLocation, result.updatedContext);
        // Merge results, preserving thrown events
        return {
            ...continueResult,
            thrownEvents: [...result.thrownEvents, ...continueResult.thrownEvents],
            eventThrown: result.eventThrown,
        };
    }

    /**
     * Process message throw event
     */
    private processMessageThrowEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: false,
            thrownEvents: [],
            caughtEvents: [],
        };

        const messageDefinition = this.extractMessageDefinition(event);

        // Create message event instance
        const messageEvent: EventInstance = {
            id: `message_${event.id}_${Date.now()}`,
            eventId: event.id,
            type: "message",
            payload: {
                messageRef: messageDefinition.messageRef,
                correlationKey: messageDefinition.correlationKey,
                data: context.variables,
            },
            firedAt: new Date(),
            source: "intermediate_throw",
        };

        result.thrownEvents.push(messageEvent);
        result.eventThrown = messageEvent; // Add convenience property
        result.updatedContext = ContextUtils.fireEvent(result.updatedContext, messageEvent);

        // Continue to next element
        const continueResult = this.continueAfterEvent(model, event, currentLocation, result.updatedContext);
        // Merge results, preserving thrown events
        return {
            ...continueResult,
            thrownEvents: [...result.thrownEvents, ...continueResult.thrownEvents],
            eventThrown: result.eventThrown,
        };
    }

    /**
     * Process error throw event
     */
    private processErrorThrowEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: false,
            thrownEvents: [],
            caughtEvents: [],
        };

        const errorDefinition = this.extractErrorDefinition(event);

        // Create error event instance
        const errorEvent: EventInstance = {
            id: `error_${event.id}_${Date.now()}`,
            eventId: event.id,
            type: "error",
            payload: {
                errorCode: errorDefinition.errorCode,
                errorMessage: errorDefinition.errorMessage,
                data: context.variables,
            },
            firedAt: new Date(),
            source: "intermediate_throw",
        };

        result.thrownEvents.push(errorEvent);
        result.updatedContext = ContextUtils.fireEvent(result.updatedContext, errorEvent);

        // Add error to context variables for potential boundary event handling
        if (!result.updatedContext.variables.errors) {
            result.updatedContext.variables.errors = [];
        }
        (result.updatedContext.variables.errors as any[]).push({
            code: errorDefinition.errorCode,
            message: errorDefinition.errorMessage,
            source: event.id,
            timestamp: new Date(),
        });

        // Error throw typically doesn't continue - it propagates up
        // Don't add next locations - let error boundary events handle it
        return result;
    }

    /**
     * Process link throw event
     */
    private processLinkThrowEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const linkDefinition = this.extractLinkDefinition(event);

        // Find corresponding link catch event
        const linkCatchEvents = model.getElementsByType("bpmn:IntermediateCatchEvent")
            .filter((catchEvent: any) => {
                const catchLinkDef = this.extractLinkDefinition(catchEvent);
                return catchLinkDef.linkName === linkDefinition.linkName;
            });

        if (linkCatchEvents.length > 0) {
            const targetEvent = linkCatchEvents[0];
            const linkLocation = model.createAbstractLocation(
                (targetEvent as any).id,
                currentLocation.routineId,
                "node",
                { 
                    parentNodeId: (targetEvent as any).id,
                    metadata: { linkSource: event.id, linkName: linkDefinition.linkName },
                },
            );

            // Create a link event for tracking
            const linkEvent: EventInstance = {
                id: `link_${event.id}_${Date.now()}`,
                eventId: event.id,
                type: "link",
                payload: { linkName: linkDefinition.linkName },
                firedAt: new Date(),
                source: "intermediate_throw",
            };

            return {
                nextLocations: [linkLocation],
                updatedContext: context,
                shouldWait: false,
                thrownEvents: [linkEvent],
                caughtEvents: [],
                eventThrown: linkEvent,
            };
        }

        // No matching catch event - this is an error condition
        throw new Error(`No matching link catch event found for link: ${linkDefinition.linkName}`);
    }

    // Catch Event Implementations

    /**
     * Process timer catch event
     */
    private processTimerCatchEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: true,
            thrownEvents: [],
            caughtEvents: [],
        };

        const timerDefinition = this.extractTimerDefinition(event);

        // Check if timer already exists for this event
        const existingTimer = context.events.timers.find(t => t.eventId === event.id);

        if (!existingTimer) {
            // Create new timer
            const timerEvent: TimerEvent = {
                id: `timer_${event.id}_${Date.now()}`,
                eventId: event.id,
                duration: timerDefinition.duration,
                dueDate: timerDefinition.dueDate,
                cycle: timerDefinition.cycle,
                expiresAt: this.calculateTimerExpiration(timerDefinition),
            };

            result.updatedContext = ContextUtils.addTimerEvent(result.updatedContext, timerEvent);
            
            // Add convenience property for tests
            result.waitingForEvent = {
                type: "timer",
                duration: timerDefinition.duration,
                dueDate: timerDefinition.dueDate,
                cycle: timerDefinition.cycle,
            };

            // Create waiting location
            const waitingLocation = model.createAbstractLocation(
                event.id,
                currentLocation.routineId,
                "timer_waiting",
                {
                    parentNodeId: event.id,
                    eventId: event.id,
                    metadata: { 
                        timerExpiration: timerEvent.expiresAt.toISOString(),
                        timerDefinition,
                    },
                },
            );
            result.nextLocations.push(waitingLocation);
        } else {
            // Check if timer has expired
            const now = new Date();
            if (existingTimer.expiresAt <= now) {
                // Timer expired - continue
                const caughtEvent: EventInstance = {
                    id: `caught_${event.id}_${Date.now()}`,
                    eventId: event.id,
                    type: "timer",
                    payload: { expiredAt: now.toISOString() },
                    firedAt: now,
                    source: "intermediate_catch",
                };

                result.caughtEvents.push(caughtEvent);
                result.eventCaught = caughtEvent; // Add convenience property
                result.updatedContext = ContextUtils.fireEvent(result.updatedContext, caughtEvent);

                // Remove expired timer
                result.updatedContext.events.timers = result.updatedContext.events.timers
                    .filter(t => t.id !== existingTimer.id);

                result.shouldWait = false;
                const continueResult = this.continueAfterEvent(model, event, currentLocation, result.updatedContext);
                // Merge results, preserving caught events
                return {
                    ...continueResult,
                    caughtEvents: [...result.caughtEvents, ...continueResult.caughtEvents],
                    eventCaught: result.eventCaught,
                };
            } else {
                // Still waiting
                const waitingLocation = model.createAbstractLocation(
                    event.id,
                    currentLocation.routineId,
                    "timer_waiting",
                    {
                        parentNodeId: event.id,
                        eventId: event.id,
                        metadata: { 
                            timerExpiration: existingTimer.expiresAt.toISOString(),
                        },
                    },
                );
                result.nextLocations.push(waitingLocation);
            }
        }

        return result;
    }

    /**
     * Process signal catch event
     */
    private processSignalCatchEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: true,
            thrownEvents: [],
            caughtEvents: [],
        };

        const signalDefinition = this.extractSignalDefinition(event);

        // Check for matching signal in context
        const matchingSignal = context.external.signalEvents.find(signal => 
            signal.signalRef === signalDefinition.signalRef &&
            this.isSignalInScope(signal, signalDefinition.scope || "global"),
        );

        if (matchingSignal) {
            // Signal received - continue
            const caughtEvent: EventInstance = {
                id: `caught_${event.id}_${Date.now()}`,
                eventId: event.id,
                type: "signal",
                payload: matchingSignal.payload || {},
                firedAt: new Date(),
                source: "intermediate_catch",
            };

            result.caughtEvents.push(caughtEvent);
            result.eventCaught = caughtEvent; // Add convenience property
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, caughtEvent);

            // Remove consumed signal
            result.updatedContext.external.signalEvents = result.updatedContext.external.signalEvents
                .filter(signal => signal.id !== matchingSignal.id);

            result.shouldWait = false;
            const continueResult = this.continueAfterEvent(model, event, currentLocation, result.updatedContext);
            // Merge results, preserving caught events
            return {
                ...continueResult,
                caughtEvents: [...result.caughtEvents, ...continueResult.caughtEvents],
                eventCaught: result.eventCaught,
            };
        } else {
            // Wait for signal
            result.waitingForEvent = {
                type: "signal",
                signalRef: signalDefinition.signalRef,
                scope: signalDefinition.scope,
            };
            const waitingLocation = model.createAbstractLocation(
                event.id,
                currentLocation.routineId,
                "signal_waiting",
                {
                    parentNodeId: event.id,
                    eventId: event.id,
                    metadata: { signalDefinition },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    /**
     * Process message catch event
     */
    private processMessageCatchEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldWait: true,
            thrownEvents: [],
            caughtEvents: [],
        };

        const messageDefinition = this.extractMessageDefinition(event);

        // Check for matching message in context
        const matchingMessage = context.external.messageEvents.find(msg => 
            this.messageMatches(msg, messageDefinition) && msg.receivedAt,
        );

        if (matchingMessage) {
            // Message received - continue
            const caughtEvent: EventInstance = {
                id: `caught_${event.id}_${Date.now()}`,
                eventId: event.id,
                type: "message",
                payload: matchingMessage.payload || {},
                firedAt: new Date(),
                source: "intermediate_catch",
            };

            result.caughtEvents.push(caughtEvent);
            result.eventCaught = caughtEvent; // Add convenience property
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, caughtEvent);

            // Remove consumed message
            result.updatedContext.external.messageEvents = result.updatedContext.external.messageEvents
                .filter(msg => msg.id !== matchingMessage.id);

            result.shouldWait = false;
            const continueResult = this.continueAfterEvent(model, event, currentLocation, result.updatedContext);
            // Merge results, preserving caught events
            return {
                ...continueResult,
                caughtEvents: [...result.caughtEvents, ...continueResult.caughtEvents],
                eventCaught: result.eventCaught,
            };
        } else {
            // Wait for message
            result.waitingForEvent = {
                type: "message",
                messageRef: messageDefinition.messageRef,
                correlationKey: messageDefinition.correlationKey,
            };
            const waitingLocation = model.createAbstractLocation(
                event.id,
                currentLocation.routineId,
                "intermediate_waiting",
                {
                    parentNodeId: event.id,
                    eventId: event.id,
                    metadata: { messageDefinition },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    /**
     * Process conditional catch event
     */
    private processConditionalCatchEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const conditionDefinition = this.extractConditionDefinition(event);
        const conditionMet = this.evaluateCondition(conditionDefinition.condition, context.variables);

        if (conditionMet) {
            // Condition met - continue
            const caughtEvent: EventInstance = {
                id: `caught_${event.id}_${Date.now()}`,
                eventId: event.id,
                type: "conditional",
                payload: { condition: conditionDefinition.condition },
                firedAt: new Date(),
                source: "intermediate_catch",
            };

            const result = this.continueAfterEvent(model, event, currentLocation, context);
            result.caughtEvents.push(caughtEvent);
            result.eventCaught = caughtEvent; // Add convenience property
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, caughtEvent);
            result.shouldWait = false;
            return result;
        } else {
            // Wait for condition
            const waitingLocation = model.createAbstractLocation(
                event.id,
                currentLocation.routineId,
                "event_waiting",
                {
                    parentNodeId: event.id,
                    eventId: event.id,
                    metadata: { conditionDefinition },
                },
            );

            return {
                nextLocations: [waitingLocation],
                updatedContext: context,
                shouldWait: true,
                thrownEvents: [],
                caughtEvents: [],
                waitingForEvent: {
                    type: "conditional",
                    condition: conditionDefinition.condition,
                },
            };
        }
    }

    /**
     * Process link catch event
     */
    private processLinkCatchEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        // Link catch events are targets of link throw events
        // If we reach here, the link has been activated
        return this.continueAfterEvent(model, event, currentLocation, context);
    }

    // Helper methods

    /**
     * Continue navigation after event processing
     */
    private continueAfterEvent(
        model: BpmnModel,
        event: any,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): IntermediateEventResult {
        const result: IntermediateEventResult = {
            nextLocations: [],
            updatedContext: context,
            shouldWait: false,
            thrownEvents: [],
            caughtEvents: [],
        };

        const outgoingFlows = model.getOutgoingFlows(event.id);
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

        return result;
    }

    private getIntermediateEventType(event: any): string {
        if (event.eventDefinitions && event.eventDefinitions.length > 0) {
            const eventDef = event.eventDefinitions[0];
            if (eventDef.$type.includes("Timer")) return "timer";
            if (eventDef.$type.includes("Signal")) return "signal";
            if (eventDef.$type.includes("Message")) return "message";
            if (eventDef.$type.includes("Error")) return "error";
            if (eventDef.$type.includes("Escalation")) return "escalation";
            if (eventDef.$type.includes("Compensation")) return "compensation";
            if (eventDef.$type.includes("Conditional")) return "conditional";
            if (eventDef.$type.includes("Link")) return "link";
        }
        return "none";
    }

    private isThrowEvent(event: any): boolean {
        return isElementType(event, "bpmn:IntermediateThrowEvent");
    }

    private extractSignalDefinition(event: any): any {
        const signalDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Signal"));
        return {
            signalRef: signalDef?.signalRef?.name || signalDef?.signalRef?.id || "default",
            scope: signalDef?.scope || "global",
        };
    }

    private extractMessageDefinition(event: any): any {
        const messageDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Message"));
        return {
            messageRef: messageDef?.messageRef?.name || messageDef?.messageRef?.id || "default",
            correlationKey: messageDef?.correlationKey || null,
        };
    }

    private extractErrorDefinition(event: any): any {
        const errorDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Error"));
        return {
            errorCode: errorDef?.errorRef?.errorCode || "default",
            errorMessage: errorDef?.errorRef?.name || "Error thrown",
        };
    }

    private extractLinkDefinition(event: any): any {
        const linkDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Link"));
        return {
            linkName: linkDef?.name || linkDef?.target || event.id,
        };
    }

    private extractTimerDefinition(event: any): any {
        const timerDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Timer"));
        return {
            duration: timerDef?.timeDuration?.body || null,
            dueDate: timerDef?.timeDate?.body || null,
            cycle: timerDef?.timeCycle?.body || null,
        };
    }

    private extractConditionDefinition(event: any): any {
        const conditionDef = event.eventDefinitions?.find((ed: any) => ed.$type.includes("Conditional"));
        return {
            condition: conditionDef?.condition?.body || "true",
        };
    }

    private calculateTimerExpiration(timerDefinition: any): Date {
        const now = new Date();
        
        if (timerDefinition.dueDate) {
            return new Date(timerDefinition.dueDate);
        }
        
        if (timerDefinition.duration) {
            const durationMs = this.parseISO8601Duration(timerDefinition.duration);
            return new Date(now.getTime() + durationMs);
        }
        
        return new Date(now.getTime() + 5 * 60 * 1000); // Default 5 minutes
    }

    private parseISO8601Duration(duration: string): number {
        const match = duration.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
        if (!match) return 5 * 60 * 1000;
        
        const hours = parseInt(match[1] || "0");
        const minutes = parseInt(match[2] || "0");
        const seconds = parseInt(match[3] || "0");
        
        return (hours * 3600 + minutes * 60 + seconds) * 1000;
    }

    private isSignalInScope(signal: SignalEvent, scope: string): boolean {
        return signal.scope === scope || signal.scope === "global" || scope === "global";
    }

    private messageMatches(messageEvent: MessageEvent, messageDefinition: any): boolean {
        return messageEvent.messageRef === messageDefinition.messageRef ||
               (messageDefinition.correlationKey && 
                messageEvent.correlationKey === messageDefinition.correlationKey);
    }

    private evaluateCondition(condition: string, variables: Record<string, unknown>): boolean {
        try {
            // Simple condition evaluation
            const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
            if (varMatch) {
                return !!variables[varMatch[1]];
            }

            const comparisonMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*(==|!=|>|<)\s*(.+)$/);
            if (comparisonMatch) {
                const varName = comparisonMatch[1];
                const operator = comparisonMatch[2];
                const value = comparisonMatch[3].replace(/['"]/g, "");
                const varValue = variables[varName];

                switch (operator) {
                    case "==": return varValue == value;
                    case "!=": return varValue != value;
                    case ">": return Number(varValue) > Number(value);
                    case "<": return Number(varValue) < Number(value);
                }
            }

            return true; // Default to true if can't evaluate
        } catch {
            return true;
        }
    }

    // Escalation and compensation handlers (placeholders for advanced features)
    private processEscalationThrowEvent(model: BpmnModel, event: any, currentLocation: AbstractLocation, context: EnhancedExecutionContext): IntermediateEventResult {
        // Create escalation event for tracking
        const escalationEvent: EventInstance = {
            id: `escalation_${event.id}_${Date.now()}`,
            eventId: event.id,
            type: "escalation",
            payload: { escalationCode: "default" },
            firedAt: new Date(),
            source: "intermediate_throw",
        };

        const result = this.continueAfterEvent(model, event, currentLocation, context);
        result.thrownEvents.push(escalationEvent);
        result.eventThrown = escalationEvent;
        return result;
    }

    private processCompensationThrowEvent(model: BpmnModel, event: any, currentLocation: AbstractLocation, context: EnhancedExecutionContext): IntermediateEventResult {
        // Create compensation event for tracking
        const compensationEvent: EventInstance = {
            id: `compensation_${event.id}_${Date.now()}`,
            eventId: event.id,
            type: "compensation",
            payload: { compensationActivity: event.id },
            firedAt: new Date(),
            source: "intermediate_throw",
        };

        const result = this.continueAfterEvent(model, event, currentLocation, context);
        result.thrownEvents.push(compensationEvent);
        result.eventThrown = compensationEvent;
        return result;
    }

    /**
     * Get pending intermediate events, optionally filtered by type
     */
    getPendingIntermediateEvents(
        context: EnhancedExecutionContext,
        eventType?: string,
    ): IntermediateEvent[] {
        let pendingEvents = context.events.pending;

        if (eventType) {
            pendingEvents = pendingEvents.filter(event => event.type === eventType);
        }

        return pendingEvents.filter(event => event.waiting);
    }

    /**
     * Complete an intermediate event
     */
    completeIntermediateEvent(
        context: EnhancedExecutionContext,
        eventId: string,
    ): { updatedContext: EnhancedExecutionContext } {
        const result = {
            updatedContext: { ...context },
        };

        // Find the intermediate event and mark it as completed
        result.updatedContext.events.pending = result.updatedContext.events.pending.map(event => {
            if (event.id === eventId) {
                return {
                    ...event,
                    waiting: false,
                };
            }
            return event;
        });

        // Move to fired events
        const completedEvent = result.updatedContext.events.pending.find(
            event => event.id === eventId && !event.waiting,
        );

        if (completedEvent) {
            const eventInstance: EventInstance = {
                id: `intermediate_${eventId}_${Date.now()}`,
                eventId,
                type: completedEvent.type,
                payload: completedEvent.eventDefinition,
                firedAt: new Date(),
                source: "intermediate_event",
            };

            result.updatedContext.events.fired.push(eventInstance);
        }

        return result;
    }
}
