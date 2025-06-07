import { type EventBus } from "../../cross-cutting/eventBus.js";
import {
    type ResourceUsage,
    type StrategyType,
    type ExecutionConstraints,
} from "@vrooli/shared";

/**
 * Telemetry event types emitted by Tier 3
 */
export enum TelemetryEventType {
    // Performance events
    STEP_STARTED = "perf.step_started",
    STEP_COMPLETED = "perf.step_completed",
    TOOL_CALL = "perf.tool_call",
    LIMIT_EXCEEDED = "perf.limit_exceeded",
    QUOTA_ABORT = "perf.quota_abort",
    
    // Business events
    STRATEGY_SELECTED = "biz.strategy_selected",
    STRATEGY_OVERRIDE = "biz.strategy_override",
    RESOURCE_ALLOCATED = "biz.resource_allocated",
    OUTPUT_GENERATED = "biz.output_generated",
    
    // Safety events
    VALIDATION_FAIL = "safety.validation_fail",
    SECURITY_VIOLATION = "safety.security_violation",
    PII_DETECTED = "safety.pii_detected",
    ERROR_OCCURRED = "safety.error_occurred",
}

/**
 * Base telemetry event structure
 */
interface TelemetryEvent {
    type: TelemetryEventType;
    timestamp: Date;
    stepId: string;
    metadata: Record<string, unknown>;
}

/**
 * TelemetryShim - Stateless telemetry emission for Tier 3
 * 
 * This component publishes performance, business, and safety events
 * without maintaining state or performing analysis. Events are consumed
 * by monitoring systems, learning engines, and analytics platforms.
 * 
 * Key principles:
 * - Fire-and-forget event emission
 * - No state maintenance
 * - Minimal performance overhead
 * - Structured event format
 */
export class TelemetryShim {
    private readonly eventBus: EventBus;
    private readonly enabled: boolean;

    constructor(eventBus: EventBus, enabled = true) {
        this.eventBus = eventBus;
        this.enabled = enabled;
    }

    /**
     * Performance Events
     */
    async emitStepStarted(
        stepId: string,
        metadata: {
            stepType: string;
            strategy: StrategyType;
            estimatedResources?: ResourceUsage;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.STEP_STARTED, stepId, metadata);
    }

    async emitStepCompleted(
        stepId: string,
        metadata: {
            strategy: StrategyType;
            duration: number;
            resourceUsage: ResourceUsage;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.STEP_COMPLETED, stepId, metadata);
    }

    async emitToolCall(
        stepId: string,
        metadata: {
            toolName: string;
            duration: number;
            success: boolean;
            resourceUsage?: ResourceUsage;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.TOOL_CALL, stepId, metadata);
    }

    async emitLimitExceeded(
        stepId: string,
        constraints: ExecutionConstraints,
    ): Promise<void> {
        await this.emit(TelemetryEventType.LIMIT_EXCEEDED, stepId, {
            constraints,
            limitType: this.determineLimitType(constraints),
        });
    }

    async emitQuotaAbort(
        stepId: string,
        metadata: {
            resourceType: string;
            requested: number;
            available: number;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.QUOTA_ABORT, stepId, metadata);
    }

    /**
     * Business Events
     */
    async emitStrategySelected(
        stepId: string,
        metadata: {
            declared: StrategyType;
            selected: StrategyType;
            reason?: string;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.STRATEGY_SELECTED, stepId, metadata);
    }

    async emitStrategyOverride(
        stepId: string,
        metadata: {
            original: StrategyType;
            override: StrategyType;
            reason: string;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.STRATEGY_OVERRIDE, stepId, metadata);
    }

    async emitResourceAllocated(
        stepId: string,
        metadata: {
            credits: number;
            timeLimit?: number;
            tools: number;
            models: string[];
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.RESOURCE_ALLOCATED, stepId, metadata);
    }

    async emitOutputGenerated(
        stepId: string,
        metadata: {
            outputKeys: string[];
            size: number;
            validationPassed: boolean;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.OUTPUT_GENERATED, stepId, metadata);
    }

    /**
     * Safety Events
     */
    async emitValidationFailure(
        stepId: string,
        errors: string[],
    ): Promise<void> {
        await this.emit(TelemetryEventType.VALIDATION_FAIL, stepId, {
            errors,
            errorCount: errors.length,
        });
    }

    async emitSecurityViolation(
        stepId: string,
        metadata: {
            violationType: string;
            severity: "low" | "medium" | "high" | "critical";
            details?: string;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.SECURITY_VIOLATION, stepId, metadata);
    }

    async emitPIIDetected(
        stepId: string,
        metadata: {
            piiTypes: string[];
            locations: string[];
            masked: boolean;
        },
    ): Promise<void> {
        await this.emit(TelemetryEventType.PII_DETECTED, stepId, metadata);
    }

    async emitExecutionError(
        stepId: string,
        error: unknown,
    ): Promise<void> {
        const errorData = this.extractErrorData(error);
        await this.emit(TelemetryEventType.ERROR_OCCURRED, stepId, errorData);
    }

    /**
     * Core emission method
     */
    private async emit(
        type: TelemetryEventType,
        stepId: string,
        metadata: Record<string, unknown>,
    ): Promise<void> {
        if (!this.enabled) {
            return;
        }

        const event: TelemetryEvent = {
            type,
            timestamp: new Date(),
            stepId,
            metadata: {
                ...metadata,
                // Add common context
                tier: "tier3",
                component: "unified-executor",
            },
        };

        try {
            // Publish to appropriate channel based on event type
            const channel = this.getChannelForEvent(type);
            await this.eventBus.publish(channel, event);
        } catch (error) {
            // Telemetry should never break execution
            console.error(`[TelemetryShim] Failed to emit event: ${type}`, error);
        }
    }

    /**
     * Helper methods
     */
    private getChannelForEvent(type: TelemetryEventType): string {
        const prefix = type.split(".")[0];
        return `telemetry.${prefix}`;
    }

    private determineLimitType(constraints: ExecutionConstraints): string {
        if (constraints.maxCredits) return "credits";
        if (constraints.maxTime) return "time";
        if (constraints.maxTokens) return "tokens";
        if (constraints.maxCost) return "cost";
        return "unknown";
    }

    private extractErrorData(error: unknown): Record<string, unknown> {
        if (error instanceof Error) {
            return {
                type: error.constructor.name,
                message: error.message,
                stack: error.stack?.split("\n").slice(0, 3), // First 3 lines only
            };
        }

        return {
            type: typeof error,
            value: String(error),
        };
    }

    /**
     * Batch emission for performance
     */
    async emitBatch(events: Array<{
        type: TelemetryEventType;
        stepId: string;
        metadata: Record<string, unknown>;
    }>): Promise<void> {
        if (!this.enabled || events.length === 0) {
            return;
        }

        // Group events by channel
        const eventsByChannel = new Map<string, TelemetryEvent[]>();

        for (const event of events) {
            const channel = this.getChannelForEvent(event.type);
            const telemetryEvent: TelemetryEvent = {
                type: event.type,
                timestamp: new Date(),
                stepId: event.stepId,
                metadata: {
                    ...event.metadata,
                    tier: "tier3",
                    component: "unified-executor",
                },
            };

            if (!eventsByChannel.has(channel)) {
                eventsByChannel.set(channel, []);
            }
            eventsByChannel.get(channel)!.push(telemetryEvent);
        }

        // Emit batches per channel
        const promises: Promise<void>[] = [];
        for (const [channel, channelEvents] of eventsByChannel) {
            promises.push(
                this.eventBus.publish(channel, {
                    batch: true,
                    events: channelEvents,
                }).catch(error => {
                    console.error(`[TelemetryShim] Failed to emit batch to ${channel}`, error);
                }),
            );
        }

        await Promise.all(promises);
    }
}
