/**
 * Bot Priority and Decision Making
 * 
 * Handles priority calculation for bot selection and provides the framework
 * for bots to make decisions about event handling.
 */

import type { BotParticipant } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import { PATTERN_SCORING, PRIORITY_WEIGHTS, ROLE_WEIGHTS } from "./constants.js";
import { getEventBehavior } from "./registry.js";
import { type BotDecision, type BotDecisionContext, type IDecisionMaker, type ServiceEvent } from "./types.js";

/**
 * Bot priority calculator
 */
export class BotPriorityCalculator {
    /**
     * Calculate priority for a bot based on various factors
     */
    calculatePriority(bot: BotParticipant, event: ServiceEvent): number {
        let priority = 0;

        // Factor 1: Explicit priority in bot configuration (highest weight)
        if (bot.priority) {
            priority += bot.priority * PRIORITY_WEIGHTS.CONFIG_PRIORITY;
        }

        // Factor 2: Pattern specificity (more specific patterns get higher priority)
        const patternSpecificity = this.calculatePatternSpecificity(bot);
        priority += patternSpecificity * PRIORITY_WEIGHTS.PATTERN_SPECIFICITY;

        // Factor 3: Bot role (arbitrator > leader > member)
        const roleWeight = this.calculateRoleWeight(bot);
        priority += roleWeight * PRIORITY_WEIGHTS.ROLE_WEIGHT;

        // Factor 4: Bot domain match based on behaviors
        const domainMatch = this.calculateDomainMatch(bot, event);
        priority += domainMatch * PRIORITY_WEIGHTS.EXPERTISE_MATCH;

        logger.debug("Bot priority calculated", {
            botId: bot.id,
            eventType: event.type,
            priority,
            factors: {
                configPriority: bot.priority,
                patternSpecificity,
                roleWeight,
                domainMatch,
            },
        });

        return Math.max(0, priority); // Ensure non-negative
    }

    /**
     * Sort bots by priority (highest first)
     */
    sortByPriority(bots: BotParticipant[], event: ServiceEvent): BotParticipant[] {
        return bots
            .map(bot => ({
                bot,
                priority: this.calculatePriority(bot, event),
            }))
            .sort((a, b) => b.priority - a.priority)
            .map(item => item.bot);
    }

    /**
     * Calculate pattern specificity score
     */
    private calculatePatternSpecificity(bot: BotParticipant): number {
        if (!bot.config.agentSpec?.behaviors || !Array.isArray(bot.config.agentSpec.behaviors)) {
            return 0;
        }

        let maxSpecificity = 0;

        for (const behavior of bot.config.agentSpec.behaviors) {
            if (!behavior.trigger?.topic) continue;

            const specificity = this.getPatternSpecificity(behavior.trigger.topic);
            maxSpecificity = Math.max(maxSpecificity, specificity);
        }

        return maxSpecificity;
    }

    /**
     * Calculate specificity score for a pattern match
     */
    private getPatternSpecificity(pattern: string): number {
        // More specific patterns (fewer wildcards) get higher scores
        const wildcardCount = (pattern.match(/[*+]/g) || []).length;
        const segmentCount = pattern.split("/").length;
        const exactMatches = pattern.split("/").filter(segment => !segment.includes("*") && !segment.includes("+")).length;

        // Base score on exact matches and penalize wildcards
        return (exactMatches * PATTERN_SCORING.EXACT_MATCH_SCORE) + (segmentCount * PATTERN_SCORING.SEGMENT_SCORE) - (wildcardCount * PATTERN_SCORING.WILDCARD_PENALTY);
    }

    /**
     * Calculate role-based weight
     */
    private calculateRoleWeight(bot: BotParticipant): number {
        const role = bot.role || "member";

        switch (role.toLowerCase()) {
            case "arbitrator": return ROLE_WEIGHTS.ARBITRATOR;
            case "leader": return ROLE_WEIGHTS.LEADER;
            case "coordinator": return ROLE_WEIGHTS.COORDINATOR;
            case "specialist": return ROLE_WEIGHTS.SPECIALIST;
            case "member": return ROLE_WEIGHTS.MEMBER;
            default: return ROLE_WEIGHTS.DEFAULT;
        }
    }

    /**
     * Calculate domain match score based on bot behaviors
     */
    private calculateDomainMatch(bot: BotParticipant, event: ServiceEvent): number {
        const botDomains = this.extractBotDomains(bot);
        const eventDomains = this.extractEventDomain(event.type);

        if (botDomains.length === 0) {
            return 0;
        }

        // Check for domain matches between bot and event
        let matchScore = 0;
        for (const botDomain of botDomains) {
            if (eventDomains.includes(botDomain.toLowerCase())) {
                matchScore += PATTERN_SCORING.EXPERTISE_MATCH_SCORE;
            }
        }

        return matchScore;
    }

    /**
     * Extract domains from bot behaviors
     */
    private extractBotDomains(bot: BotParticipant): string[] {
        const behaviors = bot.config?.agentSpec?.behaviors || [];
        const domains = new Set<string>();

        for (const behavior of behaviors) {
            if (behavior.trigger?.topic) {
                // Extract first segment as domain (e.g., "finance" from "finance/transaction/completed")
                const domain = behavior.trigger.topic.split("/")[0];
                if (domain && domain !== "*" && domain !== "#") {
                    domains.add(domain);
                }
            }
        }

        return Array.from(domains);
    }

    /**
     * Extract domain from event type
     */
    private extractEventDomain(eventType: string): string[] {
        const segments = eventType.split("/");
        return segments.map(segment => segment.toLowerCase());
    }
}

/**
 * Default decision maker implementation
 */
export class DefaultDecisionMaker implements IDecisionMaker {
    /**
     * Check if a topic pattern matches an event type
     * Supports exact matches, wildcard patterns (/*), and catch-all (#)
     */
    private matchesEventPattern(pattern: string, eventType: string): boolean {
        if (pattern === "#") return true; // Matches all
        if (pattern === eventType) return true; // Exact match

        // Wildcard pattern matching (e.g., "finance/*" matches "finance/transaction")
        const WILDCARD_SUFFIX = "/*";
        if (pattern.endsWith(WILDCARD_SUFFIX)) {
            const prefix = pattern.slice(0, -WILDCARD_SUFFIX.length);
            return eventType.startsWith(prefix + "/");
        }

        return false;
    }
    async decide(context: BotDecisionContext): Promise<BotDecision> {
        const { event, bot } = context;

        try {
            // Calculate base decision factors
            const priority = new BotPriorityCalculator().calculatePriority(bot, event);
            const shouldHandle = await this.shouldHandleEvent(context);
            const markAsHandled = shouldHandle && this.shouldMarkAsHandled(context);

            logger.debug("Bot decision made", {
                botId: bot.id,
                eventType: event.type,
                shouldHandle,
                markAsHandled,
                priority,
            });

            const decision: BotDecision = {
                shouldHandle,
                priority,
            };

            // Add response with reason if bot should handle
            if (shouldHandle) {
                decision.response = {
                    progression: markAsHandled ? "block" : "continue",
                    reason: this.generateDecisionReason(context, shouldHandle, markAsHandled),
                };
            }

            return decision;

        } catch (error) {
            logger.error("Decision making error", {
                botId: bot.id,
                eventType: event.type,
                error: error.message,
            });

            // Conservative default: don't handle
            return {
                shouldHandle: false,
                response: {
                    progression: "continue",
                    reason: `Decision error: ${error.message}`,
                },
            };
        }
    }

    /**
     * Determine if bot should handle the event
     */
    private async shouldHandleEvent(context: BotDecisionContext): Promise<boolean> {
        const { event, bot } = context;

        // Check if event allows interception based on registry
        const eventBehavior = getEventBehavior(event.type);
        if (!eventBehavior.interceptable) {
            return false;
        }

        // Check if event progression state allows processing
        if (event.progression?.state === "block") {
            return false;
        }

        // Check if bot has behaviors that match this event
        const behaviors = bot.config?.agentSpec?.behaviors || [];
        const hasMatchingBehavior = behaviors.some(behavior => {
            if (!behavior.trigger?.topic) return false;
            return this.matchesEventPattern(behavior.trigger.topic, event.type);
        });

        // Bot should handle if it has matching behaviors or is an arbitrator
        return hasMatchingBehavior || bot.role === "arbitrator";
    }

    /**
     * Determine if event should be marked as handled
     */
    private shouldMarkAsHandled(context: BotDecisionContext): boolean {
        const { bot, event } = context;

        // Find the first matching behavior for this event
        const behaviors = bot.config?.agentSpec?.behaviors || [];
        const matchingBehavior = behaviors.find(behavior => {
            if (!behavior.trigger?.topic) return false;
            return this.matchesEventPattern(behavior.trigger.topic, event.type);
        });

        if (!matchingBehavior) {
            return false;
        }

        // Check progression control settings
        if (matchingBehavior.trigger.progression?.exclusive) {
            return true; // Exclusive handling prevents other bots
        }

        // Mark as handled for routine actions that don't need responses
        if (matchingBehavior.action?.type === "routine") {
            return true;
        }

        // Don't mark as handled for invoke actions (they generate responses)
        if (matchingBehavior.action?.type === "invoke") {
            return false;
        }

        // Default: don't mark as handled to allow other bots to respond
        return false;
    }

    /**
     * Generate human-readable reason for the decision
     */
    private generateDecisionReason(
        context: BotDecisionContext,
        shouldHandle: boolean,
        markAsHandled: boolean,
    ): string {
        const { event, bot } = context;

        if (!shouldHandle) {
            const eventBehavior = getEventBehavior(event.type);
            if (!eventBehavior.interceptable) {
                return "Event does not allow interception";
            }
            if (event.progression?.state === "block") {
                return "Event progression is blocked";
            }
            return "Bot capabilities don't match event requirements";
        }

        let reason = `Bot ${bot.id} handling ${event.type}`;

        // Find matching behavior to include action type in reason
        const behaviors = bot.config?.agentSpec?.behaviors || [];
        const matchingBehavior = behaviors.find(behavior => {
            if (!behavior.trigger?.topic) return false;
            return this.matchesEventPattern(behavior.trigger.topic, event.type);
        });

        if (matchingBehavior?.action?.type) {
            reason += ` with ${matchingBehavior.action.type} action`;
        }

        if (markAsHandled) {
            reason += " (marking as handled)";
        }

        return reason;
    }
}
