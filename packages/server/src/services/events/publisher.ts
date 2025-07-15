/**
 * Simplified Event Publisher
 * 
 * Provides type-safe, simplified methods for publishing events
 * with automatic behavior lookup from the EventRegistry.
 */
import { logger } from "../../events/logger.js";
import { getEventBus } from "./eventBus.js";
import { EventUsageValidator, getEventBehavior } from "./registry.js";
import type { BotEventResponse, EventMetadata, EventPublishResult, ProgressionControl, ServiceEvent } from "./types.js";

/**
 * Unified result structure for all event emissions
 */
export interface EmitResult {
    /** Whether the operation should proceed */
    proceed: boolean;
    /** Event ID for tracking */
    eventId: string;
    /** Reason for blocking (if proceed is false) */
    reason?: string;
    /** Whether this event was processed by blocking handlers */
    wasBlocking: boolean;
    /** Full event publish result for advanced usage */
    publishResult: EventPublishResult;
}

/**
 * Unified event publishing interface
 */
export class EventPublisher {
    /**
     * Unified event emission method - ALL events should use this method
     * 
     * @example
     * // Simple emission (always check proceed!)
     * const { proceed, reason } = await EventPublisher.emit("step/completed", { stepId: "123", result: {...} });
     * if (!proceed) {
     *     logger.warn(`Step completion blocked: ${reason}`);
     *     return; // or throw, or handle as appropriate
     * }
     * 
     * // Blocking event (auto-detected from registry)
     * const { proceed, reason } = await EventPublisher.emit("tool/approval/requested", { toolName: "execute_code" });
     * if (!proceed) {
     *     throw new Error(`Tool execution blocked: ${reason}`);
     * }
     */
    static async emit<T extends ServiceEvent["data"]>(
        eventType: string,
        data: T,
        metadata?: Partial<EventMetadata>,
    ): Promise<EmitResult> {
        const behavior = getEventBehavior(eventType);

        const event: Omit<ServiceEvent, "id" | "timestamp"> = {
            type: eventType,
            data,
            metadata: {
                priority: behavior.defaultPriority || "medium",
                userId: metadata?.userId,
            },
            progression: {
                state: "continue", // Default state
                processedBy: [],
            },
        };

        logger.debug("[EventPublisher] Emitting event", {
            eventType,
            mode: behavior.mode,
            interceptable: behavior.interceptable,
        });

        const eventBus = getEventBus();
        const publishResult = await eventBus.publish(event);

        // Track all events for validation (any event can be dynamically blocked)
        if (process.env.NODE_ENV !== "production" && publishResult.eventId) {
            EventUsageValidator.trackEmission(eventType, publishResult.eventId, publishResult.wasBlocking);
        }

        // Determine if operation should proceed
        const proceed = publishResult.progression === "continue";

        // Generate user-friendly reason if blocked
        const reason = proceed ? undefined : this.aggregateReasons(publishResult.responses);

        // NOTE: We do NOT automatically mark progression as checked here!
        // The calling code must actually check the 'proceed' flag for validation to pass.
        // This ensures that all callers properly handle the possibility of blocking.

        const result: EmitResult = {
            proceed,
            eventId: publishResult.eventId || "",
            reason,
            wasBlocking: publishResult.wasBlocking || false,
            publishResult,
        };

        // In development, create a proxy to track if 'proceed' is actually checked
        if (process.env.NODE_ENV !== "production" && publishResult.eventId) {
            return new Proxy(result, {
                get(target, prop) {
                    if (prop === "proceed" && publishResult.eventId) {
                        // Mark as checked when 'proceed' is accessed
                        EventUsageValidator.markProgressionChecked(publishResult.eventId);
                    }
                    return target[prop as keyof EmitResult];
                },
            });
        }

        return result;
    }

    /**
     * Aggregate reasons from bot responses into a human-readable string
     */
    private static aggregateReasons(responses?: Array<{
        responderId: string;
        response: BotEventResponse;
        timestamp: Date;
    }>): string {
        if (!responses || responses.length === 0) {
            return "No specific reason provided";
        }

        const reasons = responses
            .filter(r => r.response.reason)
            .map(r => r.response.reason)
            .filter(Boolean);

        if (reasons.length === 0) {
            return "Event was blocked but no reasons were provided";
        }

        if (reasons.length === 1) {
            return reasons[0] || "Unknown reason";
        }

        // Multiple reasons - format nicely
        return reasons.join("; ");
    }
}

/**
 * Helper to determine final progression from multiple bot responses
 */
export function aggregateProgression(
    responses: BotEventResponse[],
    barrierConfig?: {
        continueThreshold?: number;
        blockOnFirst?: boolean;
    },
): ProgressionControl {
    if (responses.length === 0) {
        return "continue"; // No responses, use default
    }

    // If blockOnFirst is true, any block response blocks immediately
    if (barrierConfig?.blockOnFirst) {
        const blockResponse = responses.find(r => r.progression === "block");
        if (blockResponse) {
            return "block";
        }
    }

    // Count progression types
    const counts = responses.reduce((acc, response) => {
        acc[response.progression] = (acc[response.progression] || 0) + 1;
        return acc;
    }, {} as Record<ProgressionControl, number>);

    // If we have a continue threshold, check if we meet it
    if (barrierConfig?.continueThreshold !== undefined) {
        const continueCount = counts.continue || 0;
        if (continueCount >= barrierConfig.continueThreshold) {
            return "continue";
        }
        return "block";
    }

    // Default: Majority wins, ties go to "block" for safety
    const total = responses.length;
    if ((counts.continue || 0) > total / 2) {
        return "continue";
    }
    if ((counts.block || 0) >= total / 2) {
        return "block";
    }
    if ((counts.defer || 0) > total / 2) {
        return "defer";
    }
    if ((counts.retry || 0) > total / 2) {
        return "retry";
    }

    // Fallback to block for safety
    return "block";
}

/**
 * Helper to aggregate reasons from multiple bot responses
 */
export function aggregateReasons(responses: BotEventResponse[]): string | undefined {
    const reasons = responses
        .filter(r => r.reason)
        .map(r => `${r.progression}: ${r.reason}`);

    if (reasons.length === 0) {
        return undefined;
    }

    return reasons.join("; ");
}
