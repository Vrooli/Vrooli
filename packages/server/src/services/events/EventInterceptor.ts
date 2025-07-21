/**
 * Event Interceptor
 * 
 * Handles bot interception of events with proper concurrency control,
 * pattern matching, and decision making.
 */

import type { BehaviourSpec, BotParticipant, EmitAction, InvokeAction, RoutineAction, RoutineExecutionInput, SwarmState, TierExecutionRequest, TriggerContext } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import type { ISwarmContextManager } from "../execution/shared/SwarmContextManager.js";
import { SwarmStateAccessor } from "../execution/shared/SwarmStateAccessor.js";
import { RoutineExecutor } from "../execution/tier2/routineExecutor.js";
import { StepExecutor } from "../execution/tier3/stepExecutor.js";
import { DefaultDecisionMaker } from "./BotPriority.js";
import { aggregateProgression, aggregateReasons } from "./publisher.js";
import { getEventBehavior } from "./registry.js";
import {
    type BotDecisionContext,
    type BotEventResponse,
    extractChatId,
    type IDecisionMaker,
    type ILockService,
    type InterceptionResult,
    type ProgressionControl,
    type ServiceEvent,
} from "./types.js";

/**
 * Bot interceptor registration
 */
interface BotInterceptor {
    bot: BotParticipant;
    patterns: string[];
    decisionMaker: IDecisionMaker; //TODO decision should be made either if 1: invoke trigger response calls tool to make decision (need to add that tool), or routine trigger output (might need to update trigger type) matches certain criteria (another jexl expression but for the routine output)
    priority: number;
}

/**
 * Evaluate a JEXL expression in the given context
 * 
 * @param expression JEXL expression to evaluate
 * @param context TriggerContext for evaluation
 * @returns Evaluation result
 */
async function evaluateJexlExpression(
    expression: string,
    context: TriggerContext,
): Promise<any> {
    const jexl = await import("jexl");
    return await jexl.default.eval(expression, context);
}


/**
 * Event interception service
 */
export class EventInterceptor {
    private interceptors: Map<string, BotInterceptor[]> = new Map();
    private patternIndex: Map<string, Set<string>> = new Map(); // pattern -> botIds
    private activeExecutions = new Map<string, RoutineExecutor>(); // Track active routine executions
    private stateAccessor = new SwarmStateAccessor();

    constructor(
        private lockService: ILockService,
        private contextManager: ISwarmContextManager,
    ) {
        // Add any other initialization logic here
    }

    /**
     * Register a bot for event interception
     */
    registerBot(bot: BotParticipant): void {
        const patterns = this.extractPatternsFromBot(bot);
        const decisionMaker = new DefaultDecisionMaker();

        const interceptor: BotInterceptor = {
            bot,
            patterns,
            decisionMaker,
            priority: 0, // Will be calculated per-event
        };

        // Register for each pattern
        for (const pattern of patterns) {
            if (!this.interceptors.has(pattern)) {
                this.interceptors.set(pattern, []);
            }
            const interceptorList = this.interceptors.get(pattern);
            if (interceptorList) {
                interceptorList.push(interceptor);
            }

            // Update pattern index
            if (!this.patternIndex.has(pattern)) {
                this.patternIndex.set(pattern, new Set());
            }
            const patternSet = this.patternIndex.get(pattern);
            if (patternSet) {
                patternSet.add(bot.id);
            }
        }

        logger.info("[EventInterceptor] Bot registered for event interception", {
            botId: bot.id,
            botName: bot.name,
            patterns,
            behaviors: bot.config?.agentSpec?.behaviors?.length || 0,
        });
    }

    /**
     * Unregister a bot from event interception
     */
    unregisterBot(botId: string): void {
        // Remove from all patterns
        this.interceptors.forEach((interceptors, pattern) => {
            this.interceptors.set(
                pattern,
                interceptors.filter(i => i.bot.id !== botId),
            );

            // Clean up empty patterns
            const interceptorList = this.interceptors.get(pattern);
            if (interceptorList && interceptorList.length === 0) {
                this.interceptors.delete(pattern);
            }

            // Update pattern index
            this.patternIndex.get(pattern)?.delete(botId);
            if (this.patternIndex.get(pattern)?.size === 0) {
                this.patternIndex.delete(pattern);
            }
        });

        logger.info("Bot unregistered from event interception", { botId });
    }

    /**
     * Check for event interception by bots
     */
    async checkInterception(event: ServiceEvent, swarmState: SwarmState): Promise<InterceptionResult> {
        // Get event behavior to check if interception is enabled
        const eventBehavior = getEventBehavior(event.type);
        if (!eventBehavior.interceptable) {
            logger.debug("Event interception disabled for type", { eventType: event.type });
            return {
                intercepted: false,
                progression: "continue", // Default progression when not interceptable
                responses: [],
            };
        }

        // Acquire distributed lock to prevent race conditions
        const lockKey = `event_interception:${event.id}`;
        const lock = await this.lockService.acquire(lockKey, {
            ttl: 5000,
            retries: 3,
        });

        try {
            // Check if already processed (from event progression state)
            if (event.progression && event.progression.finalDecision) {
                return {
                    intercepted: true,
                    progression: event.progression.finalDecision,
                    responses: event.progression.processedBy.map(p => ({
                        botId: p.botId,
                        response: p.response,
                    })),
                };
            }

            // Get bots with matching patterns
            const matchingBots = await this.findMatchingBots(event);

            if (matchingBots.length === 0) {
                logger.debug("No bots match event patterns", {
                    eventType: event.type,
                    eventId: event.id,
                });
                return {
                    intercepted: false,
                    progression: "continue", // Default progression when lock acquisition fails
                    responses: [],
                };
            }

            // Sort bots by priority
            const sortedBots = this.sortByPriority(matchingBots, event);

            logger.debug("Checking bot interception", {
                eventType: event.type,
                eventId: event.id,
                candidateBots: sortedBots.length,
            });

            // Collect responses from all bots
            const responses: Array<{ botId: string; response: BotEventResponse }> = [];

            // Process bots based on event behavior
            if (eventBehavior.barrierConfig?.blockOnFirst) {
                // Process until first block response
                for (const interceptor of sortedBots) {
                    const response = await this.processBotInterception(interceptor, event, swarmState);
                    if (response) {
                        responses.push({ botId: interceptor.bot.id, response });

                        if (response.progression === "block") {
                            // Stop processing on first block
                            break;
                        }

                        if (response.exclusive) {
                            // Stop processing if bot claims exclusivity
                            break;
                        }
                    }
                }
            } else {
                // Process all bots (unless one claims exclusivity)
                for (const interceptor of sortedBots) {
                    const response = await this.processBotInterception(interceptor, event, swarmState);
                    if (response) {
                        responses.push({ botId: interceptor.bot.id, response });

                        if (response.exclusive) {
                            // Stop processing if bot claims exclusivity
                            break;
                        }
                    }
                }
            }

            // Determine final progression
            const progression = aggregateProgression(
                responses.map(r => r.response),
                eventBehavior.barrierConfig,
            );

            // Update event progression state
            if (event.progression) {
                event.progression.processedBy = responses.map(r => ({
                    botId: r.botId,
                    response: r.response,
                    timestamp: new Date(),
                }));
                event.progression.finalDecision = progression;
                event.progression.finalReason = aggregateReasons(responses.map(r => r.response));
            }

            return {
                intercepted: responses.length > 0,
                progression,
                responses,
                aggregatedData: this.aggregateResponseData(responses),
            };

        } finally {
            await lock.release();
        }
    }

    /**
     * Find bots that match the event patterns
     */
    private async findMatchingBots(event: ServiceEvent): Promise<BotInterceptor[]> {
        const matchingBots: BotInterceptor[] = [];
        const seenBotIds = new Set<string>();

        // Check all registered patterns
        this.interceptors.forEach((interceptors, pattern) => {
            if (this.matchesPattern(pattern, event.type)) {
                interceptors.forEach(interceptor => {
                    // Avoid duplicates if bot matches multiple patterns
                    if (!seenBotIds.has(interceptor.bot.id)) {
                        seenBotIds.add(interceptor.bot.id);
                        matchingBots.push(interceptor);
                    }
                });
            }
        });

        return matchingBots;
    }

    /**
     * Check if event type matches pattern (MQTT-style)
     */
    private matchesPattern(pattern: string, eventType: string): boolean {
        // Handle special case of catch-all pattern
        if (pattern === "#") {
            return true;
        }
        
        // Convert MQTT-style pattern to regex
        // We need to escape existing regex special characters first
        let regexPattern = pattern
            .replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); // Escape regex special chars
        
        // Now apply MQTT-style replacements
        regexPattern = regexPattern
            .replace(/\\\+/g, "[^/]+")    // + matches single level
            .replace(/\\\#/g, ".*")       // # matches multi-level (escaped)
            .replace(/\\\*/g, "[^/]*");   // * matches within level (escaped)

        const regex = new RegExp(`^${regexPattern}$`);
        return regex.test(eventType);
    }

    /**
     * Sort bots by priority for the specific event
     */
    private sortByPriority(
        interceptors: BotInterceptor[],
        event: ServiceEvent,
    ): BotInterceptor[] {
        return interceptors
            .map(interceptor => ({
                ...interceptor,
                // Calculate priority based on bot role and event type
                priority: this.calculateBotPriority(interceptor.bot, event),
            }))
            .sort((a, b) => b.priority - a.priority);
    }

    /**
     * Calculate bot priority for event handling
     */
    private calculateBotPriority(bot: BotParticipant, event: ServiceEvent): number {
        let priority = 0;

        // Role-based priority constants
        const COORDINATOR_PRIORITY = 100;
        const MONITOR_PRIORITY = 50;
        const SPECIALIST_PRIORITY = 25;

        // Role-based priority
        const role = bot.config?.agentSpec?.role;
        if (role === "coordinator") {
            priority += COORDINATOR_PRIORITY; // Coordinators have highest priority
        } else if (role === "monitor") {
            priority += MONITOR_PRIORITY; // Monitors have medium priority
        } else if (role === "specialist") {
            priority += SPECIALIST_PRIORITY; // Specialists have lower priority
        }

        // Specific pattern match gets bonus
        const EXACT_MATCH_BONUS = 50;

        const behaviors = bot.config?.agentSpec?.behaviors || [];
        for (const behavior of behaviors) {
            if (behavior.trigger?.topic === event.type) {
                priority += EXACT_MATCH_BONUS; // Exact match bonus
                break;
            }
        }

        return priority;
    }

    /**
     * Process a single bot's interception
     */
    private async processBotInterception(
        interceptor: BotInterceptor,
        event: ServiceEvent,
        swarmState: SwarmState,
    ): Promise<BotEventResponse | null> {
        try {
            // Find matching behavior
            const behavior = this.findMatchingBehavior(interceptor, event);
            if (!behavior) {
                return null;
            }

            // Build context for JEXL evaluation
            const jexlContext = this.stateAccessor.buildTriggerContext(swarmState, event, interceptor.bot);

            // Evaluate JEXL conditions if present
            if (behavior.trigger.when) {
                const shouldTrigger = await evaluateJexlExpression(
                    behavior.trigger.when,
                    jexlContext,
                );
                if (!shouldTrigger) {
                    return null;
                }
            }

            // Let bot's decision maker decide
            const context: BotDecisionContext = {
                event,
                bot: interceptor.bot,
                swarmState,
            };

            const decision = await interceptor.decisionMaker.decide(context);

            if (!decision.shouldHandle) {
                return null;
            }

            // Execute bot's action
            const result = await this.executeBotAction(interceptor, event, behavior);

            // Determine progression based on behavior configuration
            let progression: ProgressionControl = "continue";

            if (behavior.trigger.progression) {
                const { control, condition, exclusive } = behavior.trigger.progression;

                if (control === "block") {
                    progression = "block";
                } else if (control === "conditional" && condition) {
                    // Evaluate progression condition with result
                    const progressionContext = { ...jexlContext, result };
                    const shouldContinue = await evaluateJexlExpression(condition, progressionContext);
                    progression = shouldContinue ? "continue" : "block";
                } else {
                    progression = "continue";
                }

                return {
                    progression,
                    reason: decision.response?.reason,
                    data: result,
                    exclusive: exclusive || false,
                };
            }

            // Use decision's response if no behavior progression specified
            return decision.response || {
                progression: "continue",
                data: result,
            };

        } catch (error) {
            logger.error("Bot interception error", {
                botId: interceptor.bot.id,
                eventType: event.type,
                error: error.message,
            });

            // Default response on error
            return {
                progression: "block",
                reason: "Bot error during interception",
            };
        }
    }

    /**
     * Aggregate data from multiple bot responses
     */
    private aggregateResponseData(
        responses: Array<{ botId: string; response: BotEventResponse }>,
    ): any {
        // Merge data from all responses
        const aggregated: Record<string, any> = {};

        for (const { botId, response } of responses) {
            if (response.data) {
                aggregated[botId] = response.data;
            }
        }

        return Object.keys(aggregated).length > 0 ? aggregated : undefined;
    }

    /**
     * Find behavior that matches the event
     */
    private findMatchingBehavior(interceptor: BotInterceptor, event: ServiceEvent): BehaviourSpec | null {
        const behaviors = interceptor.bot.config?.agentSpec?.behaviors || [];
        for (const behavior of behaviors) {
            if (behavior.trigger?.topic && this.matchesPattern(behavior.trigger.topic, event.type)) {
                return behavior;
            }
        }
        return null;
    }

    /**
     * Execute bot's action for the event
     */
    private async executeBotAction(
        interceptor: BotInterceptor,
        event: ServiceEvent,
        behavior: BehaviourSpec,
    ): Promise<any> {
        try {
            if (!behavior?.action) {
                return null;
            }

            logger.info("Executing bot action", {
                botId: interceptor.bot.id,
                eventType: event.type,
                actionType: behavior.action.type,
            });

            if (behavior.action.type === "routine") {
                return await this.executeRoutineAction(behavior.action, event, interceptor.bot);
            } else if (behavior.action.type === "invoke") {
                return await this.executeInvokeAction(behavior.action, event, interceptor.bot);
            } else if (behavior.action.type === "emit") {
                return await this.executeEmitAction(behavior.action, event, interceptor.bot);
            }

            return null;

        } catch (error) {
            logger.error("Bot action execution error", {
                botId: interceptor.bot.id,
                eventType: event.type,
                error: error.message,
            });
            throw error;
        }
    }

    /**
     * Execute routine action using the Tier 2 execution architecture
     */
    private async executeRoutineAction(action: RoutineAction, event: ServiceEvent, bot: BotParticipant): Promise<any> {
        const contextId = `${bot.id}_${event.id}_${Date.now()}`;
        const runId = `run_${Date.now()}`;

        logger.info("Executing routine action via RoutineExecutor", {
            routineId: action.routineId,
            label: action.label,
            inputMap: action.inputMap,
            botId: bot.id,
            eventType: event.type,
            contextId,
            runId,
        });

        try {
            // Get current swarm context to derive swarm ID and resource constraints
            // For now, we'll extract swarmId from the event context or generate one
            // In a real implementation, this should come from the event context
            const parentSwarmId = extractChatId(event) || "default-swarm";

            // Allocate resources for this routine execution from the existing swarm
            const resourceAllocation = await this.contextManager.allocateResources(parentSwarmId, {
                consumerId: runId,
                consumerType: "run",
                limits: {
                    maxCredits: "1000", // 1000 credits for bot routine execution
                    maxDurationMs: 300000, // 5 minutes max
                    maxMemoryMB: 256, // 256MB memory limit
                    maxConcurrentSteps: 3, // Max 3 concurrent steps
                },
                allocated: {
                    credits: 1000, // Actual amount allocated
                    timestamp: new Date(),
                },
                priority: bot.role === "arbitrator" ? "high" : "normal",
            });

            // Validate that routine ID is provided
            if (!action.routineId) {
                throw new Error(`Routine action missing required routineId for label: ${action.label}`);
            }

            // Create proper execution context request with resource allocation
            const tierExecutionRequest: TierExecutionRequest<RoutineExecutionInput> = {
                context: {
                    swarmId: parentSwarmId,
                    userData: {
                        __typename: "SessionUser" as const,
                        id: bot.id,
                        credits: "1000", // Default credits for bot execution
                        creditAccountId: null,
                        creditSettings: null,
                        handle: null, // BotParticipant doesn't have handle
                        hasPremium: bot.role === "arbitrator", // Arbitrators get premium status
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"], // Default to English
                        name: bot.name || null,
                        phoneNumberVerified: false,
                        profileImage: null,
                        publicId: bot.id, // Use bot id as publicId
                        session: {
                            __typename: "SessionUserSession" as const,
                            id: `${bot.id}_session`,
                            lastRefreshAt: new Date().toISOString(),
                        },
                        theme: null,
                        updatedAt: new Date().toISOString(),
                    },
                    // Include metadata for outputOperations processing
                    parentSwarmId,
                    timestamp: new Date(),
                    // Include outputOperations metadata for routine execution to process
                    // Use type assertion to add custom metadata field
                    ...({
                        outputOperationsMetadata: {
                            outputOperations: action.outputOperations,
                            originatingBotId: bot.id,
                            triggerEventId: event.id,
                            triggerEventType: event.type,
                        },
                    } as any),
                },
                input: {
                    resourceVersionId: action.routineId, // Bot executions target specific resource versions
                    parameters: action.inputMap || {},
                    workflow: {
                        steps: [],
                        dependencies: [],
                        parallelBranches: [],
                    },
                    // No runId for bot executions (always new)
                },
                allocation: resourceAllocation.limits,
            };

            // Create step executor
            const stepExecutor = new StepExecutor();

            // Create RoutineExecutor using the existing context manager
            const routineExecutor = new RoutineExecutor(
                this.contextManager, // Use existing context manager
                stepExecutor,
                contextId,
                undefined, // No run context manager for bot executions
                bot.id,
            );

            // Track active execution
            this.activeExecutions.set(runId, routineExecutor);

            // Start execution (non-blocking for event interception)
            const executionPromise = routineExecutor.execute(tierExecutionRequest);

            // Enhanced cleanup with resource release
            executionPromise.finally(async () => {
                try {
                    // Release allocated resources
                    await this.contextManager.releaseResources(parentSwarmId, resourceAllocation.id);

                    // Remove from tracking
                    this.activeExecutions.delete(runId);

                    logger.debug("Routine execution completed with resource cleanup", {
                        runId,
                        botId: bot.id,
                        resourcesReleased: resourceAllocation.limits.maxCredits,
                    });
                } catch (error) {
                    logger.error("Error during routine execution cleanup", {
                        runId,
                        botId: bot.id,
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            });

            return {
                type: "routine_started",
                routineId: action.routineId,
                label: action.label,
                runId,
                contextId,
                botId: bot.id,
                resourceAllocation,
                executionPromise, // Return promise for async tracking if needed
            };

        } catch (error) {
            logger.error("Failed to start routine execution", {
                routineId: action.routineId,
                botId: bot.id,
                error: error instanceof Error ? error.message : String(error),
            });

            throw error;
        }
    }

    /**
     * Execute invoke action using Tier 3 step execution for direct strategy invocation
     */
    private async executeInvokeAction(action: InvokeAction, event: ServiceEvent, bot: BotParticipant): Promise<any> {
        const contextId = `${bot.id}_invoke_${event.id}_${Date.now()}`;
        const invocationId = `invoke_${Date.now()}`;

        logger.info("Executing invoke action via StepExecutor", {
            purpose: action.purpose,
            botId: bot.id,
            eventType: event.type,
            contextId,
            invocationId,
        });

        try {
            // Create step executor for direct strategy execution
            const stepExecutor = new StepExecutor();

            // Create a simplified execution context for strategy invocation
            const executionContext = {
                swarmId: contextId,
                botId: bot.id,
                purpose: action.purpose,
                eventData: event.data,
                eventType: event.type,
                timestamp: new Date(),
            };

            // For invoke actions, we create a direct LLM/strategy call
            // This represents the bot's immediate response to the event
            const strategyPrompt = this.buildStrategyPrompt(action, event, bot);

            // Execute strategy via step executor (simplified for direct invocation)
            const strategyResult = await this.executeStrategy(
                stepExecutor,
                strategyPrompt,
                executionContext,
            );

            return {
                type: "invoke_executed",
                purpose: action.purpose,
                invocationId,
                contextId,
                botId: bot.id,
                response: strategyResult.response,
                analysis: strategyResult.analysis,
                recommendations: strategyResult.recommendations,
                executedAt: new Date(),
            };

        } catch (error) {
            logger.error("Failed to execute invoke action", {
                purpose: action.purpose,
                botId: bot.id,
                error: error instanceof Error ? error.message : String(error),
            });

            throw error;
        }
    }

    /**
     * Build strategy prompt for invoke action
     */
    private buildStrategyPrompt(action: InvokeAction, event: ServiceEvent, bot: BotParticipant): string {
        return `
You are acting as bot "${bot.id}" with the purpose: ${action.purpose}

Event Context:
- Event Type: ${event.type}
- Event Data: ${JSON.stringify(event.data, null, 2)}
- Timestamp: ${event.timestamp}

Bot Configuration:
- Role: ${bot.config?.agentSpec?.role || "unspecified"}
- Behaviors: ${bot.config?.agentSpec?.behaviors?.length || 0} defined behaviors

Task: Analyze this event and provide your response based on your purpose and capabilities.
Provide a structured response with:
1. Analysis of the event
2. Your recommended action/response  
3. Any additional recommendations

Format your response as JSON with fields: analysis, response, recommendations.
        `.trim();
    }

    /**
     * Execute strategy using step executor
     */
    private async executeStrategy(
        stepExecutor: StepExecutor,
        prompt: string,
        context: any,
    ): Promise<{ response: string; analysis: string; recommendations: string[] }> {
        try {
            // For now, create a simplified strategy execution
            // In a full implementation, this would use the stepExecutor's LLM capabilities

            logger.debug("Executing strategy with step executor", {
                contextId: context.swarmId,
                botId: context.botId,
                purpose: context.purpose,
            });

            // Simplified response - in real implementation, would call stepExecutor.executeLLMStep()
            const response = {
                analysis: `Analyzed event ${context.eventType} for bot ${context.botId}`,
                response: `Executed strategy for purpose: ${context.purpose}`,
                recommendations: [
                    "Continue monitoring this event type",
                    "Consider escalating if pattern repeats",
                ],
            };

            return response;

        } catch (error) {
            logger.error("Strategy execution failed", {
                contextId: context.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Return fallback response
            return {
                analysis: "Strategy execution encountered an error",
                response: `Fallback response for purpose: ${context.purpose}`,
                recommendations: ["Review strategy execution configuration"],
            };
        }
    }

    /**
     * Execute emit action by publishing a new event to the event bus
     */
    private async executeEmitAction(action: EmitAction, event: ServiceEvent, bot: BotParticipant): Promise<any> {
        const emitId = `emit_${Date.now()}`;
        
        logger.info("Executing emit action", {
            eventType: action.eventType,
            botId: bot.id,
            originalEventType: event.type,
            emitId,
        });

        try {
            // Build event data using JEXL expressions from dataMapping
            const eventData: Record<string, any> = {};
            
            if (action.dataMapping) {
                // Build context for JEXL evaluation
                const swarmState = await this.contextManager.getContext(extractChatId(event) || "default-swarm");
                if (!swarmState) {
                    throw new Error("Swarm state not found for emit action");
                }
                const jexlContext = this.stateAccessor.buildTriggerContext(swarmState, event, bot);
                
                // Evaluate each mapping expression
                for (const [key, expression] of Object.entries(action.dataMapping)) {
                    try {
                        eventData[key] = await evaluateJexlExpression(expression, jexlContext);
                    } catch (error) {
                        logger.warn("Failed to evaluate JEXL expression in emit action", {
                            key,
                            expression,
                            error: error instanceof Error ? error.message : String(error),
                        });
                        eventData[key] = null; // Set to null on evaluation failure
                    }
                }
            }

            // Add bot context to event data
            eventData.originBot = {
                id: bot.id,
                role: bot.config?.agentSpec?.role,
                originalEvent: {
                    type: event.type,
                    id: event.id,
                },
            };

            // Prepare event metadata
            const eventMetadata: any = {
                priority: action.metadata?.priority || "medium",
                deliveryGuarantee: action.metadata?.deliveryGuarantee || "fire-and-forget",
                ttl: action.metadata?.ttl || 300000, // 5 minutes default
                emittedBy: bot.id,
                emittedAt: new Date().toISOString(),
            };

            // Create the new event
            const newEvent: ServiceEvent<any> = {
                id: emitId,
                type: action.eventType,
                data: eventData,
                timestamp: new Date(),
                metadata: eventMetadata,
                // No progression state for newly emitted events
            };

            // Emit the event through the event bus
            // Note: This would need to be connected to the actual EventPublisher
            // For now, we'll simulate the emit and return the result
            logger.info("Event emitted successfully", {
                eventId: emitId,
                eventType: action.eventType,
                botId: bot.id,
                dataFields: Object.keys(eventData),
            });

            // Prepare result data
            const result = {
                type: "event_emitted",
                eventId: emitId,
                eventType: action.eventType,
                botId: bot.id,
                emittedAt: new Date(),
                eventData,
                metadata: eventMetadata,
            };

            // Note: outputOperations for EmitAction would be handled by a similar 
            // pattern as RoutineAction, but EmitAction doesn't directly execute routines.
            // This could be implemented in the future if needed.

            return result;

        } catch (error) {
            logger.error("Failed to execute emit action", {
                eventType: action.eventType,
                botId: bot.id,
                error: error instanceof Error ? error.message : String(error),
            });

            throw error;
        }
    }

    /**
     * Extract patterns from bot configuration
     */
    private extractPatternsFromBot(bot: BotParticipant): string[] {
        const patterns: string[] = [];

        // Extract from agentSpec behaviors
        const behaviors = bot.config?.agentSpec?.behaviors;
        if (behaviors && Array.isArray(behaviors)) {
            for (const behavior of behaviors) {
                if (behavior.trigger?.topic) {
                    patterns.push(behavior.trigger.topic);
                }
            }
        }

        // Also check subscriptions from agentSpec
        const subscriptions = bot.config?.agentSpec?.subscriptions;
        if (subscriptions && Array.isArray(subscriptions)) {
            patterns.push(...subscriptions);
        }

        // Default patterns based on bot role
        if (patterns.length === 0) {
            const role = bot.config?.agentSpec?.role;
            if (role === "coordinator") {
                patterns.push("swarm/*", "goal/*"); // Coordination events
            } else if (role === "monitor") {
                patterns.push("#"); // Listen to all events
            } else {
                // Extract domains from behaviors for pattern generation
                const behaviors = bot.config?.agentSpec?.behaviors || [];
                const domains = new Set<string>();

                for (const behavior of behaviors) {
                    if (behavior.trigger?.topic) {
                        const domain = behavior.trigger.topic.split("/")[0];
                        if (domain && domain !== "*" && domain !== "#") {
                            domains.add(domain);
                        }
                    }
                }

                // Add domain-based patterns
                domains.forEach(domain => {
                    patterns.push(`${domain}/*`);
                });
            }
        }

        return Array.from(new Set(patterns)); // Remove duplicates
    }

    /**
     * Get active executions for monitoring
     */
    getActiveExecutions(): Map<string, RoutineExecutor> {
        return new Map(this.activeExecutions);
    }

    /**
     * Stop and clean up all active executions
     */
    async stopAllActiveExecutions(): Promise<void> {
        logger.info(`Stopping ${this.activeExecutions.size} active executions`);

        const stopPromises: Promise<void>[] = [];

        this.activeExecutions.forEach((executor, runId) => {
            stopPromises.push(
                executor.getStateMachine().stop({ reason: "Stopping all active executions" })
                    .then(() => undefined) // Convert StopResult to void
                    .catch(error => {
                        logger.error("Failed to stop execution", {
                            runId,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }),
            );
        });

        await Promise.allSettled(stopPromises);
        this.activeExecutions.clear();

        logger.info("All active executions stopped and cleared");
    }

    /**
     * Get interception statistics
     */
    getStats(): {
        registeredBots: number;
        totalPatterns: number;
        patternDistribution: Map<string, number>;
        activeExecutions: number;
    } {
        const patternDistribution = new Map<string, number>();

        this.interceptors.forEach((interceptors, pattern) => {
            patternDistribution.set(pattern, interceptors.length);
        });

        return {
            registeredBots: new Set(
                Array.from(this.interceptors.values())
                    .flat()
                    .map(i => i.bot.id),
            ).size,
            totalPatterns: this.interceptors.size,
            patternDistribution,
            activeExecutions: this.activeExecutions.size,
        };
    }
}
