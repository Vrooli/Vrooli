/**
 * Event Bus Rate Limiter
 * 
 * Provides rate limiting for events flowing through the unified event system.
 * Uses the proven token bucket algorithm from the existing RequestService
 * but adapted for event-based resource management.
 * 
 * This is the critical missing piece that prevents resource exhaustion and
 * ensures fair usage across the three-tier execution architecture.
 */

import * as fs from "fs";
import { type Cluster, type Redis } from "ioredis";
import * as path from "path";
import { fileURLToPath } from "url";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { type CoordinationEvent, type ExecutionEvent, extractChatId, type ProcessEvent, type SafetyEvent, type ServiceEvent } from "./types.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const tokenBucketScriptFile = path.join(__dirname, "../../utils/tokenBucketScript.lua");

/**
 * Rate limit configuration for different event types
 */
export interface EventRateLimit {
    /** Maximum events per second */
    eventsPerSecond: number;
    /** Burst capacity (max tokens in bucket) */
    burstCapacity: number;
    /** Cost per event in credits */
    creditCost: number;
    /** Whether this event type bypasses rate limits */
    bypass?: boolean;
}

/**
 * Rate limit result
 */
export interface RateLimitResult {
    /** Whether the event is allowed */
    allowed: boolean;
    /** Remaining quota until next reset */
    remainingQuota?: number;
    /** When the quota resets */
    resetTime?: Date;
    /** If rate limited, how long to wait (ms) */
    retryAfterMs?: number;
    /** Which rate limit was triggered */
    limitType?: "user" | "event_type" | "global" | "cost";
}

/**
 * Enhanced event publish result with rate limiting info
 */
export interface EventPublishResultWithRateLimit {
    success: boolean;
    error?: Error;
    duration: number;
    rateLimited?: boolean;
    remainingQuota?: number;
    resetTime?: Date;
}

/**
 * Event cost configuration - maps event types to resource costs
 */
export interface EventCostConfig {
    /** Base cost for coordination events (Tier 1) */
    coordination: number;
    /** Base cost for process events (Tier 2) */
    process: number;
    /** Base cost for execution events (Tier 3) */
    execution: number;
    /** Cost multipliers for specific event types */
    eventTypeMultipliers: Record<string, number>;
    /** Events that consume external API credits */
    externalApiEvents: Record<string, number>;
}

/**
 * Default event cost configuration
 */
export const DEFAULT_EVENT_COSTS: EventCostConfig = {
    // Tier costs (base credits per event)
    coordination: 1,    // Tier 1: Planning and goal events
    process: 5,        // Tier 2: Routine execution events  
    execution: 10,     // Tier 3: Tool calls and step execution

    // Event type multipliers
    eventTypeMultipliers: {
        // High-cost operations
        "tool/called": 10,           // Tool execution is expensive
        "swarm/goal/created": 5,     // Creating swarms has overhead
        "routine/started": 3,        // Starting routines has setup cost

        // Medium-cost operations
        "step/completed": 2,         // Step completion has processing cost
        "context/updated": 2,        // Context updates require sync

        // Low-cost operations
        "state/changed": 1,          // State changes are cheap
        "tool/approval_required": 1, // Approval requests are cheap

        // Zero-cost monitoring events
        "resource/allocated": 0,     // Resource tracking is free
        "resource/usage": 0,         // Usage monitoring is free
        "performance/measured": 0,   // Performance metrics are free
    },

    // External API events with specific credit costs
    externalApiEvents: {
        "tool/openai_completion": 100,    // OpenAI API calls
        "tool/anthropic_completion": 150, // Claude API calls
        "tool/web_search": 10,           // Web search API
        "tool/image_generation": 200,    // Image generation APIs
    },
};

/**
 * Rate limit configurations for different event categories
 */
export const DEFAULT_RATE_LIMITS: Record<string, EventRateLimit> = {
    // Tier-based limits
    "coordination": {
        eventsPerSecond: 50,    // Tier 1: High throughput for planning
        burstCapacity: 200,
        creditCost: 1,
    },
    "process": {
        eventsPerSecond: 100,   // Tier 2: Medium throughput for execution
        burstCapacity: 500,
        creditCost: 5,
    },
    "execution": {
        eventsPerSecond: 10,    // Tier 3: Lower throughput for expensive operations
        burstCapacity: 50,
        creditCost: 10,
    },

    // Safety events never rate limited
    "safety": {
        eventsPerSecond: Infinity,
        burstCapacity: Infinity,
        creditCost: 0,
        bypass: true,
    },

    // Specific high-cost events
    "tool/called": {
        eventsPerSecond: 5,     // Limit tool calls aggressively
        burstCapacity: 20,
        creditCost: 100,
    },

    // External API events
    "external_api": {
        eventsPerSecond: 2,     // Very conservative for external APIs
        burstCapacity: 10,
        creditCost: 150,
    },

    // Global fallback
    "default": {
        eventsPerSecond: 20,
        burstCapacity: 100,
        creditCost: 5,
    },
};

/**
 * Event Bus Rate Limiter
 * 
 * Integrates with the existing token bucket infrastructure but adds
 * event-specific logic for the three-tier execution architecture.
 */
export class EventBusRateLimiter {
    private static tokenBucketScript = "";
    private static scriptSha: string | null = null;
    private static scriptLoading: Promise<string> | null = null;

    constructor(
        private readonly eventCosts: EventCostConfig = DEFAULT_EVENT_COSTS,
        private readonly rateLimits: Record<string, EventRateLimit> = DEFAULT_RATE_LIMITS,
    ) {
        // Load the token bucket script (reusing existing infrastructure)
        try {
            if (!EventBusRateLimiter.tokenBucketScript) {
                EventBusRateLimiter.tokenBucketScript = fs.readFileSync(tokenBucketScriptFile, "utf8");
            }
        } catch (error) {
            logger.error("[EventBusRateLimiter] Could not load token bucket script", {
                scriptPath: tokenBucketScriptFile,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new CustomError("0369", "InternalError", { error });
        }
    }

    /**
     * Check if an event is allowed based on rate limits
     * 
     * This is the main entry point called by the event bus before publishing events.
     * It checks multiple rate limit dimensions and returns comprehensive result.
     */
    async checkEventRateLimit<T extends ServiceEvent>(event: T): Promise<RateLimitResult> {
        try {
            const client = await CacheService.get().raw();
            if (!client) {
                // If Redis is unavailable, allow events but log warning
                logger.warn("[EventBusRateLimiter] Redis unavailable, bypassing rate limits");
                return { allowed: true };
            }

            // Determine rate limit configuration for this event
            const config = this.getEventRateLimitConfig(event);

            // Safety events always bypass rate limits
            if (config.bypass) {
                return { allowed: true };
            }

            // Extract rate limiting keys for this event
            const rateLimitKeys = this.buildRateLimitKeys(event);

            // Check all applicable rate limits
            const results = await this.checkMultipleRateLimits(
                client,
                rateLimitKeys,
                config,
            );

            // Determine overall result
            const overallResult = this.evaluateRateLimitResults(results, config);

            logger.debug("[EventBusRateLimiter] Rate limit check completed", {
                eventType: event.type,
                eventId: event.id,
                allowed: overallResult.allowed,
                limitType: overallResult.limitType,
                retryAfterMs: overallResult.retryAfterMs,
            });

            return overallResult;

        } catch (error) {
            logger.error("[EventBusRateLimiter] Rate limit check failed", {
                eventType: event.type,
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });

            // Fail-closed: if rate limiting fails, deny the event
            return {
                allowed: false,
                limitType: "global",
                retryAfterMs: 5000, // Default retry after 5 seconds
            };
        }
    }

    /**
     * Get rate limit configuration for a specific event
     */
    private getEventRateLimitConfig<T extends ServiceEvent>(event: T): EventRateLimit {
        // Check for specific event type limits first
        if (this.rateLimits[event.type]) {
            return this.rateLimits[event.type];
        }

        // Check for tier-based limits
        const tier = this.getEventTier(event);
        if (tier && this.rateLimits[tier]) {
            return this.rateLimits[tier];
        }

        // Check for category-based limits
        const category = this.getEventCategory(event);
        if (category && this.rateLimits[category]) {
            return this.rateLimits[category];
        }

        // Fallback to default
        return this.rateLimits.default;
    }

    /**
     * Get the tier for an event (coordination, process, execution)
     */
    private getEventTier<T extends ServiceEvent>(event: T): string | null {
        // Type guards for different event types
        if (this.isCoordinationEvent(event)) return "coordination";
        if (this.isProcessEvent(event)) return "process";
        if (this.isExecutionEvent(event)) return "execution";
        if (this.isSafetyEvent(event)) return "safety";

        return null;
    }

    /**
     * Get the category for an event (e.g., "external_api", "tool")
     */
    private getEventCategory<T extends ServiceEvent>(event: T): string | null {
        // External API events
        if (this.eventCosts.externalApiEvents[event.type]) {
            return "external_api";
        }

        // Tool events
        if (event.type.startsWith("tool/")) {
            return "tool";
        }

        // Resource events  
        if (event.type.startsWith("resource/")) {
            return "resource";
        }

        return null;
    }

    /**
     * Build rate limiting keys for an event
     * 
     * Returns multiple keys to enforce different rate limit dimensions:
     * - Per user
     * - Per event type
     * - Global system limits
     */
    private buildRateLimitKeys<T extends ServiceEvent>(event: T): string[] {
        const keys: string[] = [];
        const baseKey = "event-rate-limit";

        // Global system rate limit
        keys.push(`${baseKey}:global`);

        // Per-event-type rate limit
        keys.push(`${baseKey}:event-type:${event.type}`);

        // Per-tier rate limit
        const tier = this.getEventTier(event);
        if (tier) {
            keys.push(`${baseKey}:tier:${tier}`);
        }

        // Per-user rate limit (if user context available)
        const userId = this.extractUserId(event);
        if (userId) {
            keys.push(`${baseKey}:user:${userId}`);

            // Per-user-per-event-type rate limit for expensive events
            if (this.isExpensiveEvent(event)) {
                keys.push(`${baseKey}:user:${userId}:event-type:${event.type}`);
            }
        }

        // Per-conversation rate limit (for swarm events)
        const chatId = this.extractChatId(event);
        if (chatId) {
            keys.push(`${baseKey}:conversation:${chatId}`);
        }

        return keys;
    }

    /**
     * Check multiple rate limits using the token bucket script
     */
    private async checkMultipleRateLimits(
        client: Redis | Cluster,
        keys: string[],
        config: EventRateLimit,
    ): Promise<Array<{ key: string; allowed: boolean; waitTimeMs: number }>> {
        const nowMs = Date.now();
        const args: string[] = [];

        // Build ARGV for the Lua script
        for (const key of keys) {
            args.push(config.burstCapacity.toString());
            args.push(config.eventsPerSecond.toString());
        }
        args.push(nowMs.toString());

        // Get or load the script SHA
        const scriptSha = await this.getScriptSha(client);

        let result: number[] = [];
        try {
            result = await client.evalsha(scriptSha, keys.length, ...keys, ...args) as number[];
        } catch (error) {
            if (error instanceof Error && error.message.startsWith("NOSCRIPT")) {
                // Reload script and retry
                const newScriptSha = await this.getScriptSha(client, true);
                result = await client.evalsha(newScriptSha, keys.length, ...keys, ...args) as number[];
            } else {
                throw error;
            }
        }

        // Parse results
        const results: Array<{ key: string; allowed: boolean; waitTimeMs: number }> = [];
        for (let i = 0; i < keys.length; i++) {
            const allowed = result[2 * i] === 1;
            const waitTimeMs = result[2 * i + 1];
            results.push({
                key: keys[i],
                allowed,
                waitTimeMs,
            });
        }

        return results;
    }

    /**
     * Evaluate multiple rate limit results to determine overall outcome
     */
    private evaluateRateLimitResults(
        results: Array<{ key: string; allowed: boolean; waitTimeMs: number }>,
        config: EventRateLimit,
    ): RateLimitResult {
        const DEFAULT_RESET_INTERVAL_MS = 1000;

        // If any rate limit is exceeded, deny the event
        for (const result of results) {
            if (!result.allowed) {
                const limitType = this.inferLimitTypeFromKey(result.key);
                return {
                    allowed: false,
                    limitType,
                    retryAfterMs: result.waitTimeMs,
                };
            }
        }

        // All rate limits passed
        return {
            allowed: true,
            remainingQuota: config.burstCapacity, // Simplified - could calculate actual remaining
            resetTime: new Date(Date.now() + DEFAULT_RESET_INTERVAL_MS), // Simplified - could calculate actual reset time
        };
    }

    /**
     * Infer the type of rate limit from the key
     */
    private inferLimitTypeFromKey(key: string): "user" | "event_type" | "global" | "cost" {
        if (key.includes(":user:")) return "user";
        if (key.includes(":event-type:")) return "event_type";
        if (key.includes(":global")) return "global";
        return "global";
    }

    /**
     * Extract user ID from event context
     */
    private extractUserId<T extends ServiceEvent>(event: T): string | null {
        const metadata = event.metadata;
        const data = event.data as any;

        return metadata?.userId ||
            data?.userId ||
            data?.initiatingUser ||
            data?.sessionUser?.id ||
            null;
    }

    /**
     * Extract conversation ID from event context  
     */
    private extractChatId<T extends ServiceEvent>(event: T): string | null {
        return extractChatId(event);
    }

    /**
     * Check if an event is expensive (requires stricter rate limiting)
     */
    private isExpensiveEvent<T extends ServiceEvent>(event: T): boolean {
        // External API events are always expensive
        if (this.eventCosts.externalApiEvents[event.type]) {
            return true;
        }

        // Tool calls are expensive
        if (event.type.startsWith("tool/") && event.type !== "tool/approval_required") {
            return true;
        }

        // Swarm creation is expensive
        if (event.type === "swarm/goal/created") {
            return true;
        }

        return false;
    }

    /**
     * Type guards for different event types
     */
    private isCoordinationEvent(event: ServiceEvent): event is CoordinationEvent {
        return event.type.startsWith("swarm/") ||
            event.type.startsWith("goal/") ||
            event.type.startsWith("team/") ||
            event.type.startsWith("resource/");
    }

    private isProcessEvent(event: ServiceEvent): event is ProcessEvent {
        return event.type.startsWith("routine/") ||
            event.type.startsWith("state/") ||
            event.type.startsWith("context/");
    }

    private isExecutionEvent(event: ServiceEvent): event is ExecutionEvent {
        return event.type.startsWith("step/") ||
            event.type.startsWith("tool/") ||
            event.type.startsWith("strategy/");
    }

    private isSafetyEvent(event: ServiceEvent): event is SafetyEvent {
        return event.type.startsWith("safety/") ||
            event.type.startsWith("emergency/") ||
            event.type.startsWith("threat/");
    }

    /**
     * Get script SHA for the token bucket Lua script
     * Reuses the existing infrastructure from RequestService
     */
    private async getScriptSha(client: Redis | Cluster, isReload = false): Promise<string> {
        if (isReload) {
            EventBusRateLimiter.scriptSha = null;
            EventBusRateLimiter.scriptLoading = null;
        }

        if (EventBusRateLimiter.scriptSha) {
            return EventBusRateLimiter.scriptSha;
        }

        if (EventBusRateLimiter.scriptLoading) {
            return EventBusRateLimiter.scriptLoading;
        }

        const loadPromise = client.script("LOAD", EventBusRateLimiter.tokenBucketScript) as Promise<string>;
        EventBusRateLimiter.scriptLoading = loadPromise
            .then((sha: unknown) => {
                const loadedSha = sha as string;
                EventBusRateLimiter.scriptSha = loadedSha;
                EventBusRateLimiter.scriptLoading = null;
                return loadedSha;
            })
            .catch((err: Error) => {
                EventBusRateLimiter.scriptLoading = null;
                throw err;
            });

        return EventBusRateLimiter.scriptLoading;
    }

    /**
     * Get current rate limit status for monitoring
     */
    async getRateLimitStatus(userId?: string, eventType?: string): Promise<{
        global: { allowed: number; total: number };
        user?: { allowed: number; total: number };
        eventType?: { allowed: number; total: number };
    }> {
        try {
            const client = await CacheService.get().raw();
            if (!client) {
                return { global: { allowed: 0, total: 0 } };
            }

            // This is a simplified implementation
            // In production, you'd query the actual token bucket states
            return {
                global: { allowed: 1000, total: 1000 },
                user: userId ? { allowed: 100, total: 100 } : undefined,
                eventType: eventType ? { allowed: 50, total: 50 } : undefined,
            };
        } catch (error) {
            logger.error("[EventBusRateLimiter] Failed to get rate limit status", { error });
            return { global: { allowed: 0, total: 0 } };
        }
    }

    /**
     * Create a rate limited event that gets emitted when rate limits are exceeded
     */
    createRateLimitedEvent(
        originalEvent: ServiceEvent,
        rateLimitResult: RateLimitResult,
    ): ServiceEvent {
        return {
            id: `rate-limited-${originalEvent.id}`,
            type: "resource/rate_limited",
            timestamp: new Date(),
            data: {
                originalEventId: originalEvent.id,
                originalEventType: originalEvent.type,
                limitType: rateLimitResult.limitType,
                retryAfterMs: rateLimitResult.retryAfterMs,
                userId: this.extractUserId(originalEvent) || undefined,
                chatId: this.extractChatId(originalEvent) || undefined,
            } as any,
            metadata: {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
            },
        };
    }
}
