/**
 * Scenario Runner
 * 
 * Executes test scenarios and collects results
 */

import { EventPublisher } from "../../../../services/events/publisher.js";
import { getEventBus } from "../../../../services/events/eventBus.js";
import type { ScenarioContext, ScenarioEvent, RoutineCall, ScenarioResult } from "./types.js";
import type { EventSubscriptionId } from "../../../../services/events/types.js";
import { ScenarioExecutionError } from "../../types.js";
import { ResourceManager, type ResourceLimits } from "./ResourceManager.js";
import { ErrorHandler } from "./ErrorHandler.js";
import { logger } from "../../../../events/logger.js";

export interface ScenarioRunOptions {
    initialEvent: {
        type: string;
        data: Record<string, unknown>;
    };
    timeout?: number;
    stepDebugging?: boolean;
    resourceLimits?: Partial<ResourceLimits>;
}


export class ScenarioRunner {
    private context: ScenarioContext;
    private eventSubscriptions: Map<string, EventSubscriptionId> = new Map();
    private routineInterceptor: any;
    private completionPromise: Promise<{ attempts: number; finalStatus: string }> | null = null;
    private completionResolve: ((result: { attempts: number; finalStatus: string }) => void) | null = null;
    private resourceManager: ResourceManager | null = null;
    private errorHandler: ErrorHandler;

    constructor(context: ScenarioContext) {
        this.context = context;
        this.errorHandler = new ErrorHandler({
            maxRetries: 2,
            retryDelay: 500,
            timeoutMs: 30000,
            onError: async (error, context) => {
                console.error(`Scenario error in ${context.operation}:`, error);
            },
        });
    }

    async execute(options: ScenarioRunOptions): Promise<ScenarioResult> {
        const startTime = Date.now();
        const errors: Error[] = [];
        
        // Initialize resource manager
        const resourceLimits: ResourceLimits = {
            maxMemoryMB: 256,
            maxDurationMs: options.timeout || 30000,
            maxEventSubscriptions: 50,
            maxConcurrentOperations: 10,
            ...options.resourceLimits,
        };
        
        this.resourceManager = new ResourceManager(resourceLimits);
        
        try {
            // Set up event capture with error handling
            await this.errorHandler.withErrorHandling(
                () => this.setupEventCapture(),
                {
                    scenarioName: this.context.name,
                    operation: "setupEventCapture",
                    startTime: new Date(),
                },
            );
            
            // Set up routine call interception with error handling
            await this.errorHandler.withErrorHandling(
                () => this.setupRoutineInterception(),
                {
                    scenarioName: this.context.name,
                    operation: "setupRoutineInterception",
                    startTime: new Date(),
                },
            );
            
            // Create completion promise
            this.completionPromise = new Promise((resolve) => {
                this.completionResolve = resolve;
            });
            
            // Emit initial event to start the scenario
            const eventBus = getEventBus();
            const eventResult = await this.errorHandler.withErrorHandling(
                () => eventBus.publish({
                    type: options.initialEvent.type,
                    data: options.initialEvent.data,
                    source: "scenario_runner",
                }),
                {
                    scenarioName: this.context.name,
                    operation: "emitInitialEvent",
                    startTime: new Date(),
                },
            );
            
            if (!eventResult.success) {
                throw new ScenarioExecutionError(
                    `Initial event failed: ${eventResult.error?.message || "Unknown error"}`,
                    this.context.name,
                );
            }
            
            logger.info("[ScenarioRunner] Initial event emitted successfully", {
                scenarioName: this.context.name,
                eventType: options.initialEvent.type,
                eventId: eventResult.eventId,
            });
            
            // Wait for scenario completion or timeout with resource monitoring
            const result = await this.errorHandler.withErrorHandling(
                () => Promise.race([
                    this.completionPromise!,
                    this.createTimeoutPromise(options.timeout || 30000),
                ]),
                {
                    scenarioName: this.context.name,
                    operation: "waitForCompletion",
                    startTime: new Date(),
                },
            );
            
            return {
                success: this.validateExpectations(),
                blackboard: this.blackboardToObject(),
                events: this.context.events,
                routineCalls: this.context.routineCalls,
                duration: Date.now() - startTime,
                resourceUsage: this.resourceManager.getCurrentUsage(),
                ...result,
                errors: errors.length > 0 ? errors : undefined,
            };
        } catch (error) {
            errors.push(error as Error);
            return {
                success: false,
                blackboard: this.blackboardToObject(),
                events: this.context.events,
                routineCalls: this.context.routineCalls,
                duration: Date.now() - startTime,
                resourceUsage: this.resourceManager?.getCurrentUsage(),
                errors,
            };
        } finally {
            await this.cleanup();
        }
    }

    private async setupEventCapture(): Promise<void> {
        const eventBus = getEventBus();
        
        // Subscribe to all expected event topics
        if (this.context.expectations.eventSequence) {
            for (const topic of this.context.expectations.eventSequence) {
                const subscriptionId = await eventBus.subscribe(
                    topic,
                    (event) => {
                        this.context.events.push({
                            topic,
                            data: event.data,
                            timestamp: new Date(),
                            source: event.source || "unknown",
                        });
                        
                        logger.debug("[ScenarioRunner] Event captured", {
                            scenarioName: this.context.name,
                            eventType: topic,
                            eventId: event.id,
                            totalEvents: this.context.events.length,
                        });
                        
                        // Check if scenario is complete after each event
                        if (this.isScenarioComplete() && this.completionResolve) {
                            logger.info("[ScenarioRunner] Scenario completion detected", {
                                scenarioName: this.context.name,
                                totalEvents: this.context.events.length,
                            });
                            this.completionResolve({
                                attempts: this.getAttemptCount(),
                                finalStatus: this.getFinalStatus(),
                            });
                        }
                    },
                    { mode: "standard" },
                );
                
                // Store subscription ID for cleanup
                this.eventSubscriptions.set(topic, subscriptionId);
                
                // Add to resource manager
                if (this.resourceManager) {
                    await this.resourceManager.addResource({
                        id: `subscription_${topic}`,
                        type: "subscription",
                        cleanup: async () => {
                            await eventBus.unsubscribe(subscriptionId);
                        },
                        metadata: { topic },
                    });
                }
            }
        }
        
        logger.info("[ScenarioRunner] Event capture setup complete", {
            scenarioName: this.context.name,
            subscribedTopics: this.context.expectations.eventSequence || [],
        });
    }

    private async setupRoutineInterception(): Promise<void> {
        // Intercept routine calls to track them
        // This would integrate with the actual routine execution system
        // For now, create a mock interceptor that simulates routine calls
        const eventBus = getEventBus();
        
        const subscriptionId = await eventBus.subscribe(
            "routine/executed",
            (event) => {
                const routineCall = {
                    routineId: event.data.routineId || "unknown",
                    routineLabel: event.data.routineLabel || "unknown",
                    input: event.data.input || {},
                    output: event.data.output || {},
                    timestamp: new Date(),
                    duration: event.data.duration || 0,
                    success: event.data.success !== false,
                };
                
                this.context.routineCalls.push(routineCall);
                
                logger.debug("[ScenarioRunner] Routine call captured", {
                    scenarioName: this.context.name,
                    routineLabel: routineCall.routineLabel,
                    success: routineCall.success,
                    totalCalls: this.context.routineCalls.length,
                });
            },
            { mode: "standard" },
        );
        
        this.eventSubscriptions.set("routine/executed", subscriptionId);
        
        logger.info("[ScenarioRunner] Routine interception setup complete", {
            scenarioName: this.context.name,
        });
    }

    private createTimeoutPromise(timeout: number): Promise<never> {
        return new Promise((_, reject) => {
            setTimeout(() => {
                reject(new ScenarioExecutionError(
                    `Scenario timed out after ${timeout}ms`,
                    this.context.name,
                ));
            }, timeout);
        });
    }

    private isScenarioComplete(): boolean {
        // Check if all expected events have occurred
        if (this.context.expectations.eventSequence) {
            const receivedTopics = this.context.events.map(e => e.topic);
            const allEventsReceived = this.context.expectations.eventSequence.every(
                topic => receivedTopics.includes(topic),
            );
            if (!allEventsReceived) return false;
        }

        // Check if expected routine calls have been made
        if (this.context.expectations.routineCalls) {
            for (const expected of this.context.expectations.routineCalls) {
                const actualCalls = this.context.routineCalls.filter(
                    call => call.routineLabel === expected.routine,
                ).length;
                if (actualCalls < expected.times) return false;
            }
        }

        return true;
    }

    private validateExpectations(): boolean {
        // Validate final blackboard state
        if (this.context.expectations.finalBlackboard) {
            for (const [key, expectedValue] of Object.entries(this.context.expectations.finalBlackboard)) {
                const actualValue = this.context.blackboard.get(key);
                if (JSON.stringify(actualValue) !== JSON.stringify(expectedValue)) {
                    return false;
                }
            }
        }

        // Validate event sequence
        if (this.context.expectations.eventSequence) {
            const receivedTopics = this.context.events.map(e => e.topic);
            const expectedSequence = this.context.expectations.eventSequence;
            
            // Check if events occurred in order
            let sequenceIndex = 0;
            for (const topic of receivedTopics) {
                if (topic === expectedSequence[sequenceIndex]) {
                    sequenceIndex++;
                    if (sequenceIndex === expectedSequence.length) break;
                }
            }
            
            if (sequenceIndex !== expectedSequence.length) {
                return false;
            }
        }

        return true;
    }

    private blackboardToObject(): Record<string, unknown> {
        const obj: Record<string, unknown> = {};
        for (const [key, value] of this.context.blackboard.entries()) {
            obj[key] = value;
        }
        return obj;
    }

    private getAttemptCount(): number {
        // Count unique routine execution attempts
        const uniqueRoutines = new Set(this.context.routineCalls.map(c => c.routineLabel));
        return Math.max(...Array.from(uniqueRoutines).map(routine => {
            return this.context.routineCalls.filter(c => c.routineLabel === routine).length;
        }), 0);
    }

    private getFinalStatus(): string {
        // Determine final status based on blackboard state
        const validationStatus = this.context.blackboard.get("validation_status");
        if (validationStatus && typeof validationStatus === "object" && "success" in validationStatus) {
            return validationStatus.success === true ? "success" : "failed";
        }
        return "unknown";
    }

    private async cleanup(): Promise<void> {
        try {
            // Use ResourceManager for cleanup if available
            if (this.resourceManager) {
                await this.resourceManager.destroy();
            } else {
                // Fallback cleanup
                const eventBus = getEventBus();
                
                // Unsubscribe from events
                for (const [topic, subscriptionId] of this.eventSubscriptions.entries()) {
                    try {
                        await eventBus.unsubscribe(subscriptionId);
                    } catch (error) {
                        console.error(`Error unsubscribing from ${topic}:`, error);
                    }
                }
            }
            
            // Clean up routine interceptor
            if (this.routineInterceptor) {
                // Cleanup logic here
            }
            
            this.eventSubscriptions.clear();
            this.context.endTime = new Date();
            this.completionPromise = null;
            this.completionResolve = null;
            this.resourceManager = null;
        } catch (error) {
            console.error("Error during scenario cleanup:", error);
        }
    }
}
