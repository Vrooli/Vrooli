import { type Logger } from "winston";
import {
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject, type GraphBpmnConfig } from "@vrooli/shared";
import { BaseNavigator } from "./baseNavigator.js";

/**
 * BpmnNavigator - Navigator for BPMN-2.0 workflow graphs
 * 
 * This navigator handles Business Process Model and Notation (BPMN) 2.0 format
 * workflows stored in the routine's graph configuration. It parses BPMN XML
 * to understand the workflow structure and provides navigation capabilities.
 * 
 * Key features:
 * - BPMN 2.0 XML parsing and navigation
 * - Support for start/end events, tasks, gateways
 * - Sequence flow evaluation with conditions
 * - Activity mapping for subroutine integration
 * - Parallel and exclusive gateway handling
 */
export class BpmnNavigator extends BaseNavigator {
    readonly type = "bpmn";
    readonly version = "1.0.0";

    constructor(logger: Logger) {
        super(logger);
    }

    /**
     * Checks if this navigator can handle the given routine config
     */
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            
            // Must have a version
            if (!config.__version) {
                return false;
            }

            // Must have a graph config with BPMN-2.0 type
            if (!config.graph || config.graph.__type !== "BPMN-2.0") {
                return false;
            }

            // Must have BPMN schema with data
            const bpmnConfig = config.graph as GraphBpmnConfig;
            if (!bpmnConfig.schema || !bpmnConfig.schema.data) {
                return false;
            }

            // Validate it's proper XML with BPMN namespace
            const bpmnData = bpmnConfig.schema.data;
            if (typeof bpmnData !== "string" || !bpmnData.includes("xmlns:bpmn")) {
                return false;
            }

            return true;

        } catch (error) {
            this.logger.debug("[BpmnNavigator] Cannot navigate routine", {
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets the starting location in the BPMN workflow
     */
    getStartLocation(routine: unknown): Location {
        const allStartLocations = this.getAllStartLocations(routine);
        if (allStartLocations.length > 0) {
            return allStartLocations[0]; // Return the first start location for backward compatibility
        }
        throw new Error("Could not find start node in BPMN diagram");
    }

    /**
     * Gets all possible starting locations in the BPMN workflow
     */
    getAllStartLocations(routine: unknown): Location[] {
        const routineConfig = this.validateAndCache(routine);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        const startLocations: Location[] = [];

        // Find all BPMN start events
        const startEventRegex = /<bpmn:startEvent[^>]*id="([^"]*)"[^>]*>/g;
        let match;

        while ((match = startEventRegex.exec(bpmnData)) !== null) {
            if (match[1]) {
                startLocations.push(this.createLocation(match[1], routineConfig));
            }
        }

        // If no start events found, look for tasks/activities without incoming flows (orphaned starts)
        if (startLocations.length === 0) {
            const activityRegex = /<bpmn:(task|callActivity|userTask|serviceTask)[^>]*id="([^"]*)"[^>]*>/g;
            
            while ((match = activityRegex.exec(bpmnData)) !== null) {
                const activityId = match[2];
                if (activityId) {
                    // Check if this activity has any incoming sequence flows
                    const hasIncoming = bpmnData.match(new RegExp(`<bpmn:sequenceFlow[^>]*targetRef="${activityId}"`));
                    if (!hasIncoming) {
                        startLocations.push(this.createLocation(activityId, routineConfig));
                    }
                }
            }
        }

        this.logger.debug("[BpmnNavigator] Found start locations", {
            count: startLocations.length,
            locations: startLocations.map(loc => ({ id: loc.id, nodeId: loc.nodeId })),
        });

        return startLocations;
    }

    /**
     * Gets the next possible locations from current location
     */
    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        const routineConfig = await this.getCachedConfig(current.routineId);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        // Find all outgoing sequence flows from current node
        const sequenceFlowRegex = new RegExp(`<bpmn:sequenceFlow[^>]*sourceRef="${current.nodeId}"[^>]*targetRef="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?(?:>.*?<bpmn:conditionExpression[^>]*>([^<]*)</bpmn:conditionExpression>.*?</bpmn:sequenceFlow>|/>)`, "gs");
        
        const nextNodes: Location[] = [];
        let match;

        while ((match = sequenceFlowRegex.exec(bpmnData)) !== null) {
            const targetRef = match[1];
            const _flowName = match[2]; // Flow name for future use
            const condition = match[3];

            if (targetRef) {
                // Evaluate condition if present
                if (condition && condition.trim()) {
                    try {
                        const conditionResult = this.evaluateCondition(condition.trim(), context);
                        if (!conditionResult) {
                            continue; // Skip this flow if condition is false
                        }
                    } catch (error) {
                        this.logger.warn("[BpmnNavigator] Failed to evaluate sequence flow condition", {
                            nodeId: current.nodeId,
                            targetRef,
                            condition,
                            error: error instanceof Error ? error.message : String(error),
                        });
                        continue; // Skip flows with invalid conditions
                    }
                }

                nextNodes.push(this.createLocation(targetRef, routineConfig));
            }
        }

        return nextNodes;
    }

    /**
     * Checks if location is an end location
     */
    async isEndLocation(location: Location): Promise<boolean> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        // Check if it's an end event
        const endEventMatch = bpmnData.match(new RegExp(`<bpmn:endEvent[^>]*id="${location.nodeId}"`));
        if (endEventMatch) {
            return true;
        }

        // Check if it has no outgoing sequence flows
        const hasOutgoing = bpmnData.match(new RegExp(`<bpmn:sequenceFlow[^>]*sourceRef="${location.nodeId}"`));
        return !hasOutgoing;
    }

    /**
     * Gets information about a step at a location
     */
    async getStepInfo(location: Location): Promise<StepInfo> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        // Find the BPMN element by ID
        const elementMatch = bpmnData.match(new RegExp(`<bpmn:(\\w+)[^>]*id="${location.nodeId}"[^>]*(?:name="([^"]*)"[^>]*)?>`));
        
        if (elementMatch) {
            const elementType = elementMatch[1] || "task";
            const elementName = elementMatch[2] || location.nodeId;

            // Get step configuration from activity map if available
            const stepConfig = this.getStepConfigFromActivityMap(bpmnConfig, location.nodeId);

            return {
                id: location.nodeId,
                name: elementName,
                type: this.mapBpmnTypeToStepType(elementType),
                description: `BPMN ${elementType}`,
                config: stepConfig,
            };
        }

        // Fallback
        return {
            id: location.nodeId,
            name: location.nodeId,
            type: "task",
            description: "BPMN element",
        };
    }

    /**
     * Gets dependencies for a step
     */
    async getDependencies(location: Location): Promise<string[]> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        // Find all incoming sequence flows
        const sequenceFlowRegex = new RegExp(`<bpmn:sequenceFlow[^>]*sourceRef="([^"]*)"[^>]*targetRef="${location.nodeId}"`, "g");
        const dependencies: string[] = [];
        let match;

        while ((match = sequenceFlowRegex.exec(bpmnData)) !== null) {
            if (match[1]) {
                dependencies.push(match[1]);
            }
        }

        return dependencies;
    }

    /**
     * Gets parallel branches from a location
     */
    async getParallelBranches(location: Location): Promise<Location[][]> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const bpmnConfig = this.getBpmnConfig(routineConfig);
        const bpmnData = bpmnConfig.schema.data;

        // Check if current node is a parallel gateway
        const parallelGatewayMatch = bpmnData.match(new RegExp(`<bpmn:parallelGateway[^>]*id="${location.nodeId}"`));
        if (!parallelGatewayMatch) {
            return []; // Not a parallel gateway
        }

        // Find all outgoing branches
        const nextLocations = await this.getNextLocations(location, {});

        // Each next location starts a parallel branch
        // This is simplified - in a full implementation you'd trace each branch to convergence
        return nextLocations.map(nextLocation => [nextLocation]);
    }

    /**
     * Private helper methods
     */
    private getBpmnConfig(routineConfig: RoutineVersionConfigObject): GraphBpmnConfig {
        if (!routineConfig.graph || routineConfig.graph.__type !== "BPMN-2.0") {
            throw new Error("Routine config does not contain BPMN-2.0 graph");
        }

        return routineConfig.graph as GraphBpmnConfig;
    }

    private mapBpmnTypeToStepType(bpmnType: string): string {
        const typeMap: Record<string, string> = {
            "startEvent": "start",
            "endEvent": "end",
            "task": "task",
            "userTask": "user",
            "serviceTask": "service",
            "callActivity": "subroutine",
            "exclusiveGateway": "decision",
            "parallelGateway": "parallel",
            "inclusiveGateway": "inclusive",
        };

        return typeMap[bpmnType] || "task";
    }

    private getStepConfigFromActivityMap(bpmnConfig: GraphBpmnConfig, nodeId: string): Record<string, unknown> | undefined {
        if (bpmnConfig.schema.activityMap && bpmnConfig.schema.activityMap[nodeId]) {
            return bpmnConfig.schema.activityMap[nodeId];
        }
        return undefined;
    }

    private evaluateCondition(condition: string, context: Record<string, unknown>): boolean {
        // Create a safe evaluation context
        const safeContext = {
            ...context,
            // Add safe utility functions
            isEmpty: (val: unknown) => !val || (Array.isArray(val) && val.length === 0),
            isNumber: (val: unknown) => typeof val === "number",
            isString: (val: unknown) => typeof val === "string",
            isBoolean: (val: unknown) => typeof val === "boolean",
            isArray: Array.isArray,
        };

        try {
            // Use Function constructor for safer evaluation
            const keys = Object.keys(safeContext);
            const values = Object.values(safeContext);

            const func = new Function(...keys, `return ${condition}`);
            return !!func(...values);
        } catch (error) {
            this.logger.warn("[BpmnNavigator] Condition evaluation failed", {
                condition,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets triggers for a BPMN location
     */
    async getLocationTriggers(location: Location): Promise<NavigationTrigger[]> {
        try {
            const routineConfig = await this.getCachedConfig(location.routineId);
            const bpmnConfig = this.getBpmnConfig(routineConfig);
            const bpmnData = bpmnConfig.schema.data;
            const triggers: NavigationTrigger[] = [];

            // Check for message start events
            const messageStartEventRegex = new RegExp(`<bpmn:startEvent[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:messageEventDefinition[^>]*messageRef="([^"]*)"[^>]*/>`, "s");
            const messageMatch = messageStartEventRegex.exec(bpmnData);
            if (messageMatch) {
                triggers.push({
                    id: `message_trigger_${location.nodeId}`,
                    type: "message",
                    name: messageMatch[1],
                    config: {
                        messageRef: messageMatch[1],
                        nodeId: location.nodeId,
                    },
                });
            }

            // Check for timer start events
            const timerStartEventRegex = new RegExp(`<bpmn:startEvent[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:timerEventDefinition[^>]*>.*?<bpmn:timeDuration[^>]*>([^<]*)</bpmn:timeDuration>`, "s");
            const timerMatch = timerStartEventRegex.exec(bpmnData);
            if (timerMatch) {
                triggers.push({
                    id: `timer_trigger_${location.nodeId}`,
                    type: "timer",
                    config: {
                        duration: timerMatch[1],
                        nodeId: location.nodeId,
                    },
                });
            }

            // Check for signal start events
            const signalStartEventRegex = new RegExp(`<bpmn:startEvent[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:signalEventDefinition[^>]*signalRef="([^"]*)"[^>]*/>`, "s");
            const signalMatch = signalStartEventRegex.exec(bpmnData);
            if (signalMatch) {
                triggers.push({
                    id: `signal_trigger_${location.nodeId}`,
                    type: "signal",
                    name: signalMatch[1],
                    config: {
                        signalRef: signalMatch[1],
                        nodeId: location.nodeId,
                    },
                });
            }

            // Check for conditional start events
            const conditionalStartEventRegex = new RegExp(`<bpmn:startEvent[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:conditionalEventDefinition[^>]*>.*?<bpmn:condition[^>]*>([^<]*)</bpmn:condition>`, "s");
            const conditionalMatch = conditionalStartEventRegex.exec(bpmnData);
            if (conditionalMatch) {
                triggers.push({
                    id: `condition_trigger_${location.nodeId}`,
                    type: "condition",
                    config: {
                        condition: conditionalMatch[1],
                        nodeId: location.nodeId,
                    },
                });
            }

            // Check for intermediate catch events
            const intermediateCatchEventRegex = new RegExp(`<bpmn:intermediateCatchEvent[^>]*id="${location.nodeId}"[^>]*>`, "s");
            if (intermediateCatchEventRegex.test(bpmnData)) {
                // Could be message, timer, signal, etc.
                const messageIntermediateRegex = new RegExp(`<bpmn:intermediateCatchEvent[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:messageEventDefinition[^>]*messageRef="([^"]*)"[^>]*/>`, "s");
                const messageIntermediateMatch = messageIntermediateRegex.exec(bpmnData);
                if (messageIntermediateMatch) {
                    triggers.push({
                        id: `message_intermediate_${location.nodeId}`,
                        type: "message",
                        name: messageIntermediateMatch[1],
                        config: {
                            messageRef: messageIntermediateMatch[1],
                            nodeId: location.nodeId,
                            eventType: "intermediate",
                        },
                    });
                }
            }

            this.logger.debug("[BpmnNavigator] Found triggers for location", {
                locationId: location.id,
                triggerCount: triggers.length,
                triggerTypes: triggers.map(t => t.type),
            });

            return triggers;
        } catch (error) {
            this.logger.error("[BpmnNavigator] Error getting location triggers", {
                locationId: location.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Gets timeouts for a BPMN location
     */
    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        try {
            const routineConfig = await this.getCachedConfig(location.routineId);
            const bpmnConfig = this.getBpmnConfig(routineConfig);
            const bpmnData = bpmnConfig.schema.data;
            const timeouts: NavigationTimeout[] = [];

            // Check for timer events with durations
            const timerEventRegex = new RegExp(`<bpmn:.*Event[^>]*id="${location.nodeId}"[^>]*>.*?<bpmn:timerEventDefinition[^>]*>.*?<bpmn:timeDuration[^>]*>([^<]*)</bpmn:timeDuration>`, "s");
            const timerMatch = timerEventRegex.exec(bpmnData);
            if (timerMatch) {
                const durationString = timerMatch[1].trim();
                const duration = this.parseDuration(durationString);
                
                if (duration > 0) {
                    timeouts.push({
                        id: `timer_timeout_${location.nodeId}`,
                        duration,
                        onTimeout: "continue", // Timer events typically continue on timeout
                        config: {
                            originalDuration: durationString,
                            nodeId: location.nodeId,
                        },
                    });
                }
            }

            // Check for task-level timeouts (custom extension)
            const taskTimeoutRegex = new RegExp(`<bpmn:.*Task[^>]*id="${location.nodeId}"[^>]*timeout="([^"]*)"`, "s");
            const taskTimeoutMatch = taskTimeoutRegex.exec(bpmnData);
            if (taskTimeoutMatch) {
                const duration = this.parseDuration(taskTimeoutMatch[1]);
                if (duration > 0) {
                    timeouts.push({
                        id: `task_timeout_${location.nodeId}`,
                        duration,
                        onTimeout: "fail", // Tasks typically fail on timeout
                        config: {
                            originalDuration: taskTimeoutMatch[1],
                            nodeId: location.nodeId,
                        },
                    });
                }
            }

            this.logger.debug("[BpmnNavigator] Found timeouts for location", {
                locationId: location.id,
                timeoutCount: timeouts.length,
                timeouts: timeouts.map(t => ({ id: t.id, duration: t.duration, onTimeout: t.onTimeout })),
            });

            return timeouts;
        } catch (error) {
            this.logger.error("[BpmnNavigator] Error getting location timeouts", {
                locationId: location.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Checks if an event can trigger at a BPMN location
     */
    async canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean> {
        try {
            const triggers = await this.getLocationTriggers(location);
            
            for (const trigger of triggers) {
                // Check type match
                if (trigger.type !== event.type) {
                    continue;
                }

                // Type-specific matching
                switch (event.type) {
                    case "message":
                        // Match message reference or name
                        const messageRef = trigger.config.messageRef as string;
                        const eventMessage = event.payload?.messageRef as string || event.payload?.message as string;
                        if (messageRef && eventMessage && messageRef === eventMessage) {
                            return true;
                        }
                        break;

                    case "signal":
                        // Match signal reference
                        const signalRef = trigger.config.signalRef as string;
                        const eventSignal = event.payload?.signalRef as string || event.payload?.signal as string;
                        if (signalRef && eventSignal && signalRef === eventSignal) {
                            return true;
                        }
                        break;

                    case "condition":
                        // Evaluate condition
                        const condition = trigger.config.condition as string;
                        if (condition && event.payload) {
                            try {
                                const result = this.evaluateCondition(condition, event.payload);
                                if (result) {
                                    return true;
                                }
                            } catch (error) {
                                this.logger.warn("[BpmnNavigator] Failed to evaluate trigger condition", {
                                    locationId: location.id,
                                    condition,
                                    error: error instanceof Error ? error.message : String(error),
                                });
                            }
                        }
                        break;

                    case "timer":
                        // Timer events are time-based, not event-triggered
                        // They would be handled by the timeout mechanism
                        return false;

                    case "webhook":
                    case "custom":
                        // Custom logic for webhook and custom triggers
                        // Can be extended based on specific requirements
                        return true;
                }
            }

            return false;
        } catch (error) {
            this.logger.error("[BpmnNavigator] Error checking event trigger capability", {
                locationId: location.id,
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Private helper methods for event-driven navigation
     */
    private parseDuration(durationString: string): number {
        // Parse ISO 8601 duration format (PT30S, PT5M, PT1H) or simple formats (30s, 5m, 1h)
        try {
            const isoMatch = durationString.match(/^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$/);
            if (isoMatch) {
                const hours = parseInt(isoMatch[1] || "0", 10);
                const minutes = parseInt(isoMatch[2] || "0", 10);
                const seconds = parseInt(isoMatch[3] || "0", 10);
                return (hours * 3600 + minutes * 60 + seconds) * 1000; // Convert to milliseconds
            }

            // Simple format parsing
            const simpleMatch = durationString.match(/^(\d+)([smh])$/);
            if (simpleMatch) {
                const value = parseInt(simpleMatch[1], 10);
                const unit = simpleMatch[2];
                switch (unit) {
                    case "s": return value * 1000;
                    case "m": return value * 60 * 1000;
                    case "h": return value * 3600 * 1000;
                }
            }

            // If it's just a number, assume milliseconds
            const numericValue = parseInt(durationString, 10);
            if (!isNaN(numericValue)) {
                return numericValue;
            }

            return 0;
        } catch (error) {
            this.logger.warn("[BpmnNavigator] Failed to parse duration", {
                durationString,
                error: error instanceof Error ? error.message : String(error),
            });
            return 0;
        }
    }
}
