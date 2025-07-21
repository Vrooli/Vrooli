/**
 * BPMN Event Handler - Core event processing for boundary and intermediate events
 * 
 * Handles the complex logic of BPMN event processing including:
 * - Boundary event monitoring and triggering
 * - Intermediate event handling (throw/catch)
 * - Timer event management
 * - Event correlation and scoping
 */

import type { 
    EnhancedExecutionContext, 
    AbstractLocation,
    BoundaryEvent,
    IntermediateEvent,
    TimerEvent,
    EventInstance,
    MessageEvent,
    SignalEvent,
    LocationType,
} from "../../types.js";
import type { BpmnModel } from "./BpmnModel.js";
import { ContextUtils } from "../contextTransformer.js";

/**
 * Event processing result
 */
export interface EventProcessingResult {
    // New locations to navigate to
    nextLocations: AbstractLocation[];
    // Updated context with event state changes
    updatedContext: EnhancedExecutionContext;
    // Whether the current location should be terminated
    shouldTerminate: boolean;
    // Fired events that need external notification
    firedEvents: EventInstance[];
}

/**
 * BPMN Event Handler for comprehensive event processing
 */
export class BpmnEventHandler {
    
    /**
     * Process boundary events attached to a given location
     */
    processBoundaryEvents(
        model: BpmnModel,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        // Get the base node ID (the activity this boundary event is attached to)
        const baseNodeId = currentLocation.parentNodeId || currentLocation.nodeId;
        const boundaryEvents = model.getBoundaryEvents(baseNodeId);

        for (const boundaryEvent of boundaryEvents) {
            const eventResult = this.processSingleBoundaryEvent(
                model, 
                boundaryEvent, 
                baseNodeId, 
                currentLocation, 
                result.updatedContext,
            );

            // Merge results
            result.nextLocations.push(...eventResult.nextLocations);
            result.updatedContext = eventResult.updatedContext;
            result.firedEvents.push(...eventResult.firedEvents);
            
            // If an interrupting boundary event fired, terminate current location
            if (eventResult.shouldTerminate) {
                result.shouldTerminate = true;
                break; // Interrupting events stop processing
            }
        }

        return result;
    }

    /**
     * Process a single boundary event
     */
    private processSingleBoundaryEvent(
        model: BpmnModel,
        boundaryEvent: any,
        attachedToNodeId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        const eventType = this.getBoundaryEventType(boundaryEvent);
        const isInterrupting = this.isBoundaryEventInterrupting(boundaryEvent);

        switch (eventType) {
            case "timer":
                return this.processTimerBoundaryEvent(
                    model, boundaryEvent, attachedToNodeId, currentLocation, context,
                );
                
            case "error":
                return this.processErrorBoundaryEvent(
                    model, boundaryEvent, attachedToNodeId, currentLocation, context,
                );
                
            case "message":
                return this.processMessageBoundaryEvent(
                    model, boundaryEvent, attachedToNodeId, currentLocation, context,
                );
                
            case "signal":
                return this.processSignalBoundaryEvent(
                    model, boundaryEvent, attachedToNodeId, currentLocation, context,
                );
                
            default:
                // Unknown boundary event type - create monitoring location
                const monitoringLocation = model.createAbstractLocation(
                    boundaryEvent.id,
                    currentLocation.routineId,
                    "boundary_event_monitor",
                    {
                        parentNodeId: attachedToNodeId,
                        eventId: boundaryEvent.id,
                        metadata: { eventType, isInterrupting },
                    },
                );
                result.nextLocations.push(monitoringLocation);
                break;
        }

        return result;
    }

    /**
     * Process timer boundary event
     */
    private processTimerBoundaryEvent(
        model: BpmnModel,
        boundaryEvent: any,
        attachedToNodeId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        const isInterrupting = this.isBoundaryEventInterrupting(boundaryEvent);
        const timerDefinition = this.extractTimerDefinition(boundaryEvent);

        // Check if timer already exists for this boundary event
        const existingTimer = context.events.timers.find(t => t.eventId === boundaryEvent.id);

        if (!existingTimer) {
            // Create new timer event
            const timerEvent: TimerEvent = {
                id: `timer_${boundaryEvent.id}_${Date.now()}`,
                eventId: boundaryEvent.id,
                duration: timerDefinition.duration,
                dueDate: timerDefinition.dueDate,
                cycle: timerDefinition.cycle,
                expiresAt: this.calculateTimerExpiration(timerDefinition),
                attachedToRef: attachedToNodeId,
            };

            result.updatedContext = ContextUtils.addTimerEvent(result.updatedContext, timerEvent);

            // Create timer monitoring location
            const timerLocation = model.createAbstractLocation(
                boundaryEvent.id,
                currentLocation.routineId,
                "timer_waiting",
                {
                    parentNodeId: attachedToNodeId,
                    eventId: boundaryEvent.id,
                    metadata: { 
                        isInterrupting,
                        timerExpiration: timerEvent.expiresAt.toISOString(),
                        timerDefinition,
                    },
                },
            );
            result.nextLocations.push(timerLocation);
        } else {
            // Check if timer has expired
            const now = new Date();
            if (existingTimer.expiresAt <= now) {
                // Timer fired! Create event instance
                const eventInstance: EventInstance = {
                    id: `event_${boundaryEvent.id}_${Date.now()}`,
                    eventId: boundaryEvent.id,
                    type: "timer",
                    payload: { 
                        attachedToRef: attachedToNodeId,
                        firedAt: now.toISOString(),
                        timerDefinition,
                    },
                    firedAt: now,
                    source: "boundary_timer",
                };

                result.firedEvents.push(eventInstance);
                result.updatedContext = ContextUtils.fireEvent(result.updatedContext, eventInstance);

                // Remove expired timer
                result.updatedContext.events.timers = result.updatedContext.events.timers
                    .filter(t => t.id !== existingTimer.id);

                // Navigate to timer's outgoing flows
                const outgoingFlows = model.getOutgoingFlows(boundaryEvent.id);
                for (const flow of outgoingFlows) {
                    if (flow.targetRef) {
                        const targetLocation = model.createAbstractLocation(
                            flow.targetRef.id,
                            currentLocation.routineId,
                            "node",
                            { parentNodeId: flow.targetRef.id },
                        );
                        result.nextLocations.push(targetLocation);
                    }
                }

                // If interrupting, terminate the current activity
                if (isInterrupting) {
                    result.shouldTerminate = true;
                }
            } else {
                // Timer still active - continue monitoring
                const timerLocation = model.createAbstractLocation(
                    boundaryEvent.id,
                    currentLocation.routineId,
                    "timer_waiting",
                    {
                        parentNodeId: attachedToNodeId,
                        eventId: boundaryEvent.id,
                        metadata: { 
                            isInterrupting,
                            timerExpiration: existingTimer.expiresAt.toISOString(),
                        },
                    },
                );
                result.nextLocations.push(timerLocation);
            }
        }

        return result;
    }

    /**
     * Process error boundary event
     */
    private processErrorBoundaryEvent(
        model: BpmnModel,
        boundaryEvent: any,
        attachedToNodeId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        // Check if there's an error in the context that matches this boundary event
        const errorDefinition = this.extractErrorDefinition(boundaryEvent);
        const matchingError = this.findMatchingError(context, errorDefinition);

        if (matchingError) {
            // Error boundary event triggered!
            const eventInstance: EventInstance = {
                id: `event_${boundaryEvent.id}_${Date.now()}`,
                eventId: boundaryEvent.id,
                type: "error",
                payload: { 
                    attachedToRef: attachedToNodeId,
                    errorCode: errorDefinition.errorCode,
                    errorData: matchingError,
                },
                firedAt: new Date(),
                source: "boundary_error",
            };

            result.firedEvents.push(eventInstance);
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, eventInstance);

            // Navigate to error handler flows
            const outgoingFlows = model.getOutgoingFlows(boundaryEvent.id);
            for (const flow of outgoingFlows) {
                if (flow.targetRef) {
                    const targetLocation = model.createAbstractLocation(
                        flow.targetRef.id,
                        currentLocation.routineId,
                        "node",
                        { parentNodeId: flow.targetRef.id },
                    );
                    result.nextLocations.push(targetLocation);
                }
            }

            // Error boundary events are always interrupting
            result.shouldTerminate = true;
        } else {
            // No matching error - continue monitoring
            const monitoringLocation = model.createAbstractLocation(
                boundaryEvent.id,
                currentLocation.routineId,
                "boundary_event_monitor",
                {
                    parentNodeId: attachedToNodeId,
                    eventId: boundaryEvent.id,
                    metadata: { 
                        eventType: "error",
                        errorDefinition,
                    },
                },
            );
            result.nextLocations.push(monitoringLocation);
        }

        return result;
    }

    /**
     * Process message boundary event
     */
    private processMessageBoundaryEvent(
        model: BpmnModel,
        boundaryEvent: any,
        attachedToNodeId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        const isInterrupting = this.isBoundaryEventInterrupting(boundaryEvent);
        const messageDefinition = this.extractMessageDefinition(boundaryEvent);

        // Check for matching message in context
        const matchingMessage = context.external.messageEvents.find(msg => 
            this.messageMatches(msg, messageDefinition),
        );

        if (matchingMessage && matchingMessage.receivedAt) {
            // Message received! Fire boundary event
            const eventInstance: EventInstance = {
                id: `event_${boundaryEvent.id}_${Date.now()}`,
                eventId: boundaryEvent.id,
                type: "message",
                payload: { 
                    attachedToRef: attachedToNodeId,
                    messageRef: messageDefinition.messageRef,
                    messageData: matchingMessage.payload,
                },
                firedAt: new Date(),
                source: "boundary_message",
            };

            result.firedEvents.push(eventInstance);
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, eventInstance);

            // Remove consumed message
            result.updatedContext.external.messageEvents = result.updatedContext.external.messageEvents
                .filter(msg => msg.id !== matchingMessage.id);

            // Navigate to message handler flows
            const outgoingFlows = model.getOutgoingFlows(boundaryEvent.id);
            for (const flow of outgoingFlows) {
                if (flow.targetRef) {
                    const targetLocation = model.createAbstractLocation(
                        flow.targetRef.id,
                        currentLocation.routineId,
                        "node",
                        { parentNodeId: flow.targetRef.id },
                    );
                    result.nextLocations.push(targetLocation);
                }
            }

            if (isInterrupting) {
                result.shouldTerminate = true;
            }
        } else {
            // No matching message - continue waiting
            const waitingLocation = model.createAbstractLocation(
                boundaryEvent.id,
                currentLocation.routineId,
                "message_waiting",
                {
                    parentNodeId: attachedToNodeId,
                    eventId: boundaryEvent.id,
                    metadata: { 
                        isInterrupting,
                        messageDefinition,
                    },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    /**
     * Process signal boundary event
     */
    private processSignalBoundaryEvent(
        model: BpmnModel,
        boundaryEvent: any,
        attachedToNodeId: string,
        currentLocation: AbstractLocation,
        context: EnhancedExecutionContext,
    ): EventProcessingResult {
        const result: EventProcessingResult = {
            nextLocations: [],
            updatedContext: { ...context },
            shouldTerminate: false,
            firedEvents: [],
        };

        const isInterrupting = this.isBoundaryEventInterrupting(boundaryEvent);
        const signalDefinition = this.extractSignalDefinition(boundaryEvent);

        // Check for matching signal in context
        const matchingSignal = context.external.signalEvents.find(signal => 
            signal.signalRef === signalDefinition.signalRef,
        );

        if (matchingSignal) {
            // Signal received! Fire boundary event
            const eventInstance: EventInstance = {
                id: `event_${boundaryEvent.id}_${Date.now()}`,
                eventId: boundaryEvent.id,
                type: "signal",
                payload: { 
                    attachedToRef: attachedToNodeId,
                    signalRef: signalDefinition.signalRef,
                    signalData: matchingSignal.payload,
                },
                firedAt: new Date(),
                source: "boundary_signal",
            };

            result.firedEvents.push(eventInstance);
            result.updatedContext = ContextUtils.fireEvent(result.updatedContext, eventInstance);

            // Navigate to signal handler flows
            const outgoingFlows = model.getOutgoingFlows(boundaryEvent.id);
            for (const flow of outgoingFlows) {
                if (flow.targetRef) {
                    const targetLocation = model.createAbstractLocation(
                        flow.targetRef.id,
                        currentLocation.routineId,
                        "node",
                        { parentNodeId: flow.targetRef.id },
                    );
                    result.nextLocations.push(targetLocation);
                }
            }

            if (isInterrupting) {
                result.shouldTerminate = true;
            }
        } else {
            // No matching signal - continue waiting
            const waitingLocation = model.createAbstractLocation(
                boundaryEvent.id,
                currentLocation.routineId,
                "signal_waiting",
                {
                    parentNodeId: attachedToNodeId,
                    eventId: boundaryEvent.id,
                    metadata: { 
                        isInterrupting,
                        signalDefinition,
                    },
                },
            );
            result.nextLocations.push(waitingLocation);
        }

        return result;
    }

    // Helper methods for event processing

    private getBoundaryEventType(boundaryEvent: any): string {
        // Extract event type from BPMN element
        if (boundaryEvent.eventDefinitions && boundaryEvent.eventDefinitions.length > 0) {
            const eventDef = boundaryEvent.eventDefinitions[0];
            if (eventDef.$type.includes("Timer")) return "timer";
            if (eventDef.$type.includes("Error")) return "error";
            if (eventDef.$type.includes("Message")) return "message";
            if (eventDef.$type.includes("Signal")) return "signal";
            if (eventDef.$type.includes("Compensation")) return "compensation";
        }
        return "unknown";
    }

    private isBoundaryEventInterrupting(boundaryEvent: any): boolean {
        // Check cancelActivity attribute (default is true for interrupting)
        return boundaryEvent.cancelActivity !== false;
    }

    private extractTimerDefinition(boundaryEvent: any): any {
        const timerDef = boundaryEvent.eventDefinitions?.find((ed: any) => 
            ed.$type.includes("Timer"),
        );
        
        return {
            duration: timerDef?.timeDuration?.body || null,
            dueDate: timerDef?.timeDate?.body || null,
            cycle: timerDef?.timeCycle?.body || null,
        };
    }

    private calculateTimerExpiration(timerDefinition: any): Date {
        const now = new Date();
        
        if (timerDefinition.dueDate) {
            return new Date(timerDefinition.dueDate);
        }
        
        if (timerDefinition.duration) {
            // Parse ISO 8601 duration (simplified)
            const durationMs = this.parseISO8601Duration(timerDefinition.duration);
            return new Date(now.getTime() + durationMs);
        }
        
        // Default to 5 minutes if no valid timer definition
        return new Date(now.getTime() + 5 * 60 * 1000);
    }

    private parseISO8601Duration(duration: string): number {
        // Simplified ISO 8601 duration parsing
        // Format: PT30S (30 seconds), PT5M (5 minutes), PT1H (1 hour)
        const match = duration.match(/PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?/);
        if (!match) return 5 * 60 * 1000; // Default 5 minutes
        
        const hours = parseInt(match[1] || "0");
        const minutes = parseInt(match[2] || "0");
        const seconds = parseInt(match[3] || "0");
        
        return (hours * 3600 + minutes * 60 + seconds) * 1000;
    }

    private extractErrorDefinition(boundaryEvent: any): any {
        const errorDef = boundaryEvent.eventDefinitions?.find((ed: any) => 
            ed.$type.includes("Error"),
        );
        
        return {
            errorCode: errorDef?.errorRef?.errorCode || null,
            errorMessage: errorDef?.errorRef?.name || null,
        };
    }

    private findMatchingError(context: EnhancedExecutionContext, errorDefinition: any): any {
        // Look for errors in the context variables
        const errors = context.variables.errors as any[] || [];
        
        if (!errorDefinition.errorCode) {
            // No specific error code - match any error
            return errors.length > 0 ? errors[0] : null;
        }
        
        return errors.find(error => 
            error.code === errorDefinition.errorCode ||
            error.type === errorDefinition.errorCode,
        );
    }

    private extractMessageDefinition(boundaryEvent: any): any {
        const messageDef = boundaryEvent.eventDefinitions?.find((ed: any) => 
            ed.$type.includes("Message"),
        );
        
        return {
            messageRef: messageDef?.messageRef?.name || messageDef?.messageRef?.id || "default",
            correlationKey: messageDef?.correlationKey || null,
        };
    }

    private messageMatches(messageEvent: MessageEvent, messageDefinition: any): boolean {
        return messageEvent.messageRef === messageDefinition.messageRef ||
               (messageDefinition.correlationKey && 
                messageEvent.correlationKey === messageDefinition.correlationKey);
    }

    private extractSignalDefinition(boundaryEvent: any): any {
        const signalDef = boundaryEvent.eventDefinitions?.find((ed: any) => 
            ed.$type.includes("Signal"),
        );
        
        return {
            signalRef: signalDef?.signalRef?.name || signalDef?.signalRef?.id || "default",
        };
    }
}
