/**
 * Event System Utilities
 * 
 * Common utility functions for the unified event system to reduce code duplication
 * and improve consistency.
 */

import { SECONDS_1_MS } from "@vrooli/shared";
import { EVENT_BUS_CONSTANTS, PRIORITY_LEVELS } from "./constants.js";
import type { BarrierSyncConfig, EventMetadata, ServiceEvent } from "./types.js";

/**
 * Retry strategy configuration
 */
export interface RetryStrategy {
    maxAttempts: number;
    backoffMs: (attempt: number) => number;
    shouldRetry?: (error: Error, attempt: number) => boolean;
}

/**
 * Default retry strategies
 */
export const DEFAULT_RETRY_STRATEGIES = {
    /** Linear backoff with fixed delay */
    linear: (delayMs = SECONDS_1_MS): RetryStrategy => ({
        maxAttempts: 3,
        backoffMs: () => delayMs,
    }),

    /** Exponential backoff starting from base delay */
    exponential: (baseDelayMs: number = EVENT_BUS_CONSTANTS.RETRY_BASE_DELAY_MS): RetryStrategy => ({
        maxAttempts: 3,
        backoffMs: (attempt) => baseDelayMs * Math.pow(EVENT_BUS_CONSTANTS.RETRY_EXPONENTIAL_FACTOR, attempt - 1),
    }),

    /** No retry */
    none: (): RetryStrategy => ({
        maxAttempts: 1,
        backoffMs: () => 0,
    }),
} as const;

/**
 * Execute a function with retry logic
 */
export async function withRetry<T>(
    fn: () => Promise<T>,
    strategy: RetryStrategy,
    logger?: { warn: (message: string, meta?: any) => void },
): Promise<T> {
    let lastError: Error | undefined;

    for (let attempt = 1; attempt <= strategy.maxAttempts; attempt++) {
        try {
            return await fn();
        } catch (error) {
            lastError = error as Error;

            if (attempt === strategy.maxAttempts) {
                break;
            }

            if (strategy.shouldRetry && !strategy.shouldRetry(lastError, attempt)) {
                break;
            }

            const delayMs = strategy.backoffMs(attempt);

            logger?.warn(`Retry attempt ${attempt}/${strategy.maxAttempts} after ${delayMs}ms`, {
                error: lastError.message,
                attempt,
                delayMs,
            });

            await delay(delayMs);
        }
    }

    throw lastError || new Error("Retry failed with unknown error");
}

/**
 * Simple delay helper
 */
export function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}


/**
 * Pattern matching for MQTT-style event types
 */
export class EventPatternMatcher {
    private readonly pattern: string;
    private readonly regex: RegExp;

    constructor(pattern: string) {
        this.pattern = pattern;
        this.regex = this.patternToRegex(pattern);
    }

    /**
     * Check if an event type matches this pattern
     */
    matches(eventType: string): boolean {
        return this.regex.test(eventType);
    }

    /**
     * Get the original pattern string
     */
    getPattern(): string {
        return this.pattern;
    }

    /**
     * Convert MQTT-style pattern to regex
     */
    private patternToRegex(pattern: string): RegExp {
        // Special case: # matches everything
        if (pattern === "#") {
            return /^.*$/;
        }

        // Escape special regex characters except * and /
        let regexStr = pattern.replace(/[.+?^${}()|[\]\\]/g, "\\$&");

        // Replace MQTT wildcards with regex equivalents
        // * matches a single level (no slashes)
        regexStr = regexStr.replace(/\*/g, "[^/]+");

        // # matches multiple levels (must be at end)
        const HASH_SUFFIX_LENGTH = 2;
        if (regexStr.endsWith("/#")) {
            regexStr = regexStr.slice(0, -HASH_SUFFIX_LENGTH) + "(/.*)?";
        }

        return new RegExp(`^${regexStr}$`);
    }
}

/**
 * Create a pattern matcher for event subscriptions
 */
export function createPatternMatcher(pattern: string): EventPatternMatcher {
    return new EventPatternMatcher(pattern);
}

/**
 * Batch pattern matcher for multiple patterns
 */
export class BatchPatternMatcher {
    private readonly matchers: EventPatternMatcher[];

    constructor(patterns: string[]) {
        this.matchers = patterns.map(p => new EventPatternMatcher(p));
    }

    /**
     * Check if event type matches any pattern
     */
    matches(eventType: string): boolean {
        return this.matchers.some(m => m.matches(eventType));
    }

    /**
     * Get all patterns
     */
    getPatterns(): string[] {
        return this.matchers.map(m => m.getPattern());
    }
}

/**
 * Validate event structure
 */
export function validateEventStructure(event: unknown): event is ServiceEvent {
    if (!event || typeof event !== "object") {
        return false;
    }

    const e = event as any;

    return (
        typeof e.id === "string" &&
        typeof e.type === "string" &&
        e.timestamp instanceof Date &&
        e.data !== undefined &&
        (!e.progression || typeof e.progression === "object") &&
        (!e.metadata || typeof e.metadata === "object")
    );
}

/**
 * Extract tier from event type
 */
export function getTierFromEventType(eventType: string): 1 | 2 | 3 | "cross-cutting" | "safety" | undefined {
    if (eventType.startsWith("swarm/") || eventType.startsWith("goal/") ||
        eventType.startsWith("team/") || eventType.startsWith("resource/")) {
        return 1;
    }

    if (eventType.startsWith("routine/") || eventType.startsWith("state/") ||
        eventType.startsWith("context/")) {
        return 2;
    }

    if (eventType.startsWith("step/") || eventType.startsWith("tool/") ||
        eventType.startsWith("strategy/")) {
        return 3;
    }

    if (eventType.startsWith("safety/") || eventType.startsWith("emergency/") ||
        eventType.startsWith("threat/")) {
        return "safety";
    }

    if (eventType.startsWith("execution/") || eventType.startsWith("recovery/") ||
        eventType.startsWith("fallback/") || eventType.startsWith("circuit_breaker/")) {
        return "cross-cutting";
    }

    return undefined;
}

/**
 * Create default metadata based on event type
 */
export function createDefaultMetadata(eventType: string): EventMetadata {
    // Safety events get critical priority
    if (eventType.startsWith("safety/") || eventType.startsWith("emergency/")) {
        return {
            priority: PRIORITY_LEVELS.CRITICAL,
        };
    }

    // Approval events get high priority
    if (eventType.includes("approval_required")) {
        return {
            priority: PRIORITY_LEVELS.HIGH,
        };
    }

    // Default to medium priority
    return {
        priority: PRIORITY_LEVELS.MEDIUM,
    };
}

/**
 * Format event for logging
 */
export function formatEventForLogging(event: ServiceEvent): Record<string, unknown> {
    return {
        id: event.id,
        type: event.type,
        timestamp: event.timestamp.toISOString(),
        metadata: event.metadata,
        // Don't log full data payload for security/size reasons
        dataKeys: event.data && typeof event.data === "object" ? Object.keys(event.data) : undefined,
        progression: event.progression?.state,
        execution: event.execution ? {
            runId: event.execution.runId,
            parentSwarmId: event.execution.parentSwarmId,
            toolCallId: event.execution.toolCallId,
        } : undefined,
    };
}

/**
 * Create a barrier configuration with defaults
 */
export function createBarrierConfig(options: Partial<BarrierSyncConfig> = {}): BarrierSyncConfig {
    return {
        quorum: options.quorum || 1,
        timeoutMs: options.timeoutMs || EVENT_BUS_CONSTANTS.DEFAULT_BARRIER_TIMEOUT_MS,
        timeoutAction: options.timeoutAction || "block",
        requiredResponders: options.requiredResponders,
    };
}


/**
 * Calculate priority score for event ordering (higher = more important)
 */
export function calculatePriorityScore(event: ServiceEvent): number {
    const priorityScores = {
        [PRIORITY_LEVELS.CRITICAL]: 1000,
        [PRIORITY_LEVELS.HIGH]: 100,
        [PRIORITY_LEVELS.MEDIUM]: 10,
        [PRIORITY_LEVELS.LOW]: 1,
    };

    const basePriority = priorityScores[event.metadata?.priority || PRIORITY_LEVELS.MEDIUM];

    // Boost score for safety events
    const SAFETY_BOOST_SCORE = 500;
    const safetyBoost = event.type.startsWith("safety/") || event.type.startsWith("emergency/") ? SAFETY_BOOST_SCORE : 0;

    // Boost score for approval events
    const APPROVAL_BOOST_SCORE = 200;
    const approvalBoost = event.type.includes("approval") ? APPROVAL_BOOST_SCORE : 0;

    return basePriority + safetyBoost + approvalBoost;
}
