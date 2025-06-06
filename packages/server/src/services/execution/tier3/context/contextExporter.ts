import { type Logger } from "winston";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { type ResourceUsage, type StrategyType } from "@vrooli/shared";

/**
 * Context data exported for cross-tier synchronization
 */
export interface ExportedContext {
    stepId: string;
    timestamp: Date;
    strategy: StrategyType;
    resourceUsage: ResourceUsage;
    outputs: Record<string, unknown>;
    metadata?: Record<string, unknown>;
}

/**
 * Context synchronization event
 */
interface ContextSyncEvent {
    type: "context.exported";
    source: "tier3";
    data: ExportedContext;
}

/**
 * ContextExporter - Cross-tier context synchronization
 * 
 * This component exports execution context from Tier 3 to other tiers,
 * enabling coordinated decision-making and resource management across
 * the execution hierarchy.
 * 
 * Key responsibilities:
 * - Export step execution results to Tier 2
 * - Share resource usage with Tier 1
 * - Propagate strategy decisions
 * - Enable learning and optimization
 * 
 * The exporter uses event-driven communication to maintain loose coupling
 * between tiers while ensuring consistent state propagation.
 */
export class ContextExporter {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly exportChannel = "execution.context.sync";

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
    }

    /**
     * Exports context after step execution
     * 
     * This method packages execution results and broadcasts them
     * to other tiers for coordination and learning.
     */
    async exportContext(
        stepId: string,
        context: Omit<ExportedContext, "stepId" | "timestamp">,
    ): Promise<void> {
        const exportedContext: ExportedContext = {
            stepId,
            timestamp: new Date(),
            ...context,
        };

        this.logger.debug("[ContextExporter] Exporting context", {
            stepId,
            strategy: context.strategy,
            outputKeys: Object.keys(context.outputs),
        });

        try {
            // Create sync event
            const event: ContextSyncEvent = {
                type: "context.exported",
                source: "tier3",
                data: exportedContext,
            };

            // Publish to sync channel
            await this.eventBus.publish(this.exportChannel, event);

            // Also publish specific events for different consumers
            await this.publishSpecificEvents(exportedContext);

            this.logger.debug("[ContextExporter] Context exported successfully", { stepId });

        } catch (error) {
            this.logger.error("[ContextExporter] Failed to export context", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't throw - context export should not break execution
        }
    }

    /**
     * Exports batch of contexts for efficiency
     */
    async exportBatch(contexts: ExportedContext[]): Promise<void> {
        if (contexts.length === 0) return;

        this.logger.debug("[ContextExporter] Exporting context batch", {
            count: contexts.length,
            stepIds: contexts.map(c => c.stepId),
        });

        try {
            // Create batch sync event
            const event = {
                type: "context.batch_exported",
                source: "tier3",
                data: contexts,
                timestamp: new Date(),
            };

            await this.eventBus.publish(this.exportChannel, event);

        } catch (error) {
            this.logger.error("[ContextExporter] Failed to export context batch", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Exports error context for failure analysis
     */
    async exportErrorContext(
        stepId: string,
        error: Error,
        partialContext?: Partial<ExportedContext>,
    ): Promise<void> {
        const errorContext = {
            stepId,
            timestamp: new Date(),
            error: {
                type: error.constructor.name,
                message: error.message,
                stack: error.stack,
            },
            ...partialContext,
        };

        this.logger.debug("[ContextExporter] Exporting error context", {
            stepId,
            errorType: error.constructor.name,
        });

        try {
            const event = {
                type: "context.error_exported",
                source: "tier3",
                data: errorContext,
            };

            await this.eventBus.publish(this.exportChannel, event);

        } catch (publishError) {
            this.logger.error("[ContextExporter] Failed to export error context", {
                stepId,
                error: publishError instanceof Error ? publishError.message : String(publishError),
            });
        }
    }

    /**
     * Publishes specific events for different tier consumers
     */
    private async publishSpecificEvents(context: ExportedContext): Promise<void> {
        const promises: Promise<void>[] = [];

        // Resource usage event for Tier 1 (Coordination)
        if (context.resourceUsage && Object.keys(context.resourceUsage).length > 0) {
            promises.push(
                this.publishResourceUsageEvent(context.stepId, context.resourceUsage)
            );
        }

        // Strategy decision event for learning systems
        promises.push(
            this.publishStrategyDecisionEvent(context.stepId, context.strategy, context.outputs)
        );

        // Output generation event for Tier 2 (Process)
        promises.push(
            this.publishOutputEvent(context.stepId, context.outputs)
        );

        await Promise.all(promises).catch(error => {
            this.logger.warn("[ContextExporter] Some specific events failed to publish", {
                error: error instanceof Error ? error.message : String(error),
            });
        });
    }

    /**
     * Publishes resource usage event
     */
    private async publishResourceUsageEvent(
        stepId: string,
        usage: ResourceUsage,
    ): Promise<void> {
        const event = {
            type: "resource.usage_reported",
            source: "tier3",
            data: {
                stepId,
                usage,
                timestamp: new Date(),
            },
        };

        await this.eventBus.publish("resource.usage", event);
    }

    /**
     * Publishes strategy decision event
     */
    private async publishStrategyDecisionEvent(
        stepId: string,
        strategy: StrategyType,
        outputs: Record<string, unknown>,
    ): Promise<void> {
        const event = {
            type: "strategy.decision_made",
            source: "tier3",
            data: {
                stepId,
                strategy,
                outputComplexity: this.calculateOutputComplexity(outputs),
                timestamp: new Date(),
            },
        };

        await this.eventBus.publish("strategy.decisions", event);
    }

    /**
     * Publishes output generation event
     */
    private async publishOutputEvent(
        stepId: string,
        outputs: Record<string, unknown>,
    ): Promise<void> {
        const event = {
            type: "output.generated",
            source: "tier3",
            data: {
                stepId,
                outputKeys: Object.keys(outputs),
                outputSizes: this.calculateOutputSizes(outputs),
                timestamp: new Date(),
            },
        };

        await this.eventBus.publish("execution.outputs", event);
    }

    /**
     * Helper methods
     */
    private calculateOutputComplexity(outputs: Record<string, unknown>): number {
        let complexity = 0;

        for (const value of Object.values(outputs)) {
            if (typeof value === "object" && value !== null) {
                complexity += this.getObjectComplexity(value);
            } else {
                complexity += 1;
            }
        }

        return complexity;
    }

    private getObjectComplexity(obj: unknown, depth: number = 0, maxDepth: number = 5): number {
        if (depth > maxDepth) return 1;

        let complexity = 1;

        if (Array.isArray(obj)) {
            complexity += obj.length;
            for (const item of obj.slice(0, 10)) { // Sample first 10
                complexity += this.getObjectComplexity(item, depth + 1, maxDepth);
            }
        } else if (typeof obj === "object" && obj !== null) {
            const entries = Object.entries(obj);
            complexity += entries.length;
            for (const [, value] of entries.slice(0, 10)) { // Sample first 10
                complexity += this.getObjectComplexity(value, depth + 1, maxDepth);
            }
        }

        return complexity;
    }

    private calculateOutputSizes(outputs: Record<string, unknown>): Record<string, number> {
        const sizes: Record<string, number> = {};

        for (const [key, value] of Object.entries(outputs)) {
            sizes[key] = this.estimateSize(value);
        }

        return sizes;
    }

    private estimateSize(value: unknown): number {
        try {
            return JSON.stringify(value).length;
        } catch {
            return -1; // Indicates serialization error
        }
    }
}