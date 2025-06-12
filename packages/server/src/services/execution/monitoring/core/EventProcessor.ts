/**
 * EventProcessor handles incoming events from the event bus and converts
 * them into unified metrics for storage.
 */

import { generatePK } from "@vrooli/shared/id/snowflake.js";
import { type BaseEvent } from "../../cross-cutting/events/eventBus.js";
import { MetricsStore } from "./MetricsStore";
import {
    UnifiedMetric,
    MonitoringConfig,
    MetricType,
    SwarmPerformanceSnapshot,
    RunPerformanceMetrics,
    StrategyExecutionMetrics,
    ResourceUsageMetrics,
} from "../types";

/**
 * Event processing statistics
 */
interface ProcessingStats {
    totalEvents: number;
    successfullyProcessed: number;
    errors: number;
    lastProcessedTime: Date;
    processingTimeMs: number[];
}

/**
 * Processes events from the event bus and converts them to unified metrics
 */
export class EventProcessor {
    private readonly stats: ProcessingStats;
    private isInitialized = false;
    
    constructor(
        private readonly store: MetricsStore,
        private readonly config: MonitoringConfig
    ) {
        this.stats = {
            totalEvents: 0,
            successfullyProcessed: 0,
            errors: 0,
            lastProcessedTime: new Date(),
            processingTimeMs: [],
        };
    }
    
    /**
     * Initialize the event processor
     */
    async initialize(): Promise<void> {
        if (this.isInitialized) return;
        
        console.debug("Initializing EventProcessor");
        this.isInitialized = true;
        console.info("EventProcessor initialized successfully");
    }
    
    /**
     * Process a generic monitoring event
     */
    async processEvent(event: BaseEvent): Promise<void> {
        if (!this.isInitialized) {
            throw new Error("EventProcessor not initialized");
        }
        
        const startTime = process.hrtime.bigint();
        this.stats.totalEvents++;
        
        try {
            const metrics = await this.convertEventToMetrics(event);
            
            if (metrics.length > 0) {
                await this.store.storeBatch(metrics);
                this.stats.successfullyProcessed++;
            }
            
        } catch (error) {
            this.stats.errors++;
            console.error("Error processing monitoring event", { 
                error, 
                eventId: event.id,
                eventCategory: event.category,
            });
        } finally {
            const endTime = process.hrtime.bigint();
            const processingTimeMs = Number(endTime - startTime) / 1_000_000;
            
            this.recordProcessingTime(processingTimeMs);
            this.stats.lastProcessedTime = new Date();
        }
    }
    
    /**
     * Process execution-specific events
     */
    async processExecutionEvent(event: BaseEvent): Promise<void> {
        if (event.category !== "execution") return;
        
        const metrics: UnifiedMetric[] = [];
        
        try {
            switch (event.subcategory) {
                case "swarm.started":
                    metrics.push(...this.processSwarmStarted(event));
                    break;
                    
                case "swarm.completed":
                    metrics.push(...this.processSwarmCompleted(event));
                    break;
                    
                case "run.started":
                    metrics.push(...this.processRunStarted(event));
                    break;
                    
                case "run.completed":
                    metrics.push(...this.processRunCompleted(event));
                    break;
                    
                case "step.executed":
                    metrics.push(...this.processStepExecuted(event));
                    break;
                    
                case "strategy.executed":
                    metrics.push(...this.processStrategyExecuted(event));
                    break;
                    
                case "resource.allocated":
                case "resource.released":
                    metrics.push(...this.processResourceEvent(event));
                    break;
                    
                default:
                    // Handle generic execution events
                    metrics.push(...this.processGenericExecutionEvent(event));
                    break;
            }
            
            if (metrics.length > 0) {
                await this.store.storeBatch(metrics);
            }
            
        } catch (error) {
            console.error("Error processing execution event", {
                error,
                eventId: event.id,
                subcategory: event.subcategory,
            });
        }
    }
    
    /**
     * Convert generic event to metrics
     */
    private async convertEventToMetrics(event: BaseEvent): Promise<UnifiedMetric[]> {
        const metrics: UnifiedMetric[] = [];
        const timestamp = event.timestamp || new Date();
        
        // Extract tier from event source or metadata
        const tier = this.extractTierFromEvent(event);
        
        // Convert based on event category
        switch (event.category) {
            case "telemetry":
                metrics.push(...this.processTelemetryEvent(event, tier, timestamp));
                break;
                
            case "monitoring":
                if (event.subcategory === "metric" && event.data?.metric) {
                    metrics.push(event.data.metric as UnifiedMetric);
                }
                break;
                
            case "health":
                metrics.push(...this.processHealthEvent(event, tier, timestamp));
                break;
                
            case "performance":
                metrics.push(...this.processPerformanceEvent(event, tier, timestamp));
                break;
                
            case "resource":
                metrics.push(...this.processResourceEventGeneric(event, tier, timestamp));
                break;
                
            default:
                // Create a generic metric for unknown events
                metrics.push(this.createGenericMetric(event, tier, timestamp));
                break;
        }
        
        return metrics;
    }
    
    /**
     * Process telemetry events
     */
    private processTelemetryEvent(
        event: BaseEvent,
        tier: 1 | 2 | 3 | "cross-cutting",
        timestamp: Date
    ): UnifiedMetric[] {
        const metrics: UnifiedMetric[] = [];
        
        if (event.data && typeof event.data === "object") {
            for (const [key, value] of Object.entries(event.data)) {
                if (typeof value === "number" || typeof value === "boolean" || typeof value === "string") {
                    metrics.push({
                        id: generatePK().toString(),
                        timestamp,
                        tier,
                        component: event.source?.component || "unknown",
                        type: (event.subcategory as MetricType) || "performance",
                        name: key,
                        value,
                        metadata: {
                            eventId: event.id,
                            originalEvent: event.subcategory,
                        },
                    });
                }
            }
        }
        
        return metrics;
    }
    
    /**
     * Process swarm started events
     */
    private processSwarmStarted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        
        return [{
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: 1,
            component: "SwarmStateMachine",
            type: "business",
            name: "swarm.started",
            value: 1,
            executionId: data?.swarmId,
            metadata: {
                swarmId: data?.swarmId,
                teamId: data?.teamId,
                userId: data?.userId,
            },
        }];
    }
    
    /**
     * Process swarm completed events
     */
    private processSwarmCompleted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        const metrics: UnifiedMetric[] = [];
        
        // Basic completion metric
        metrics.push({
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: 1,
            component: "SwarmStateMachine",
            type: "business",
            name: "swarm.completed",
            value: 1,
            executionId: data?.swarmId,
            metadata: data,
        });
        
        // Performance metrics if available
        if (data?.performance) {
            const perf = data.performance as SwarmPerformanceSnapshot;
            
            metrics.push(
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 1,
                    component: "SwarmStateMachine",
                    type: "performance",
                    name: "swarm.completion_rate",
                    value: perf.completionRate,
                    unit: "percentage",
                    executionId: data?.swarmId,
                },
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 1,
                    component: "SwarmStateMachine",
                    type: "performance",
                    name: "swarm.avg_task_duration",
                    value: perf.avgTaskDuration,
                    unit: "ms",
                    executionId: data?.swarmId,
                },
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 1,
                    component: "SwarmStateMachine",
                    type: "quality",
                    name: "swarm.error_rate",
                    value: perf.errorRate,
                    unit: "percentage",
                    executionId: data?.swarmId,
                }
            );
        }
        
        return metrics;
    }
    
    /**
     * Process run started events
     */
    private processRunStarted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        
        return [{
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: 2,
            component: "RunStateMachine",
            type: "business",
            name: "run.started",
            value: 1,
            executionId: data?.runId,
            metadata: {
                runId: data?.runId,
                routineId: data?.routineId,
                userId: data?.userId,
            },
        }];
    }
    
    /**
     * Process run completed events
     */
    private processRunCompleted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        const metrics: UnifiedMetric[] = [];
        
        metrics.push({
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: 2,
            component: "RunStateMachine",
            type: "business",
            name: "run.completed",
            value: 1,
            executionId: data?.runId,
            metadata: data,
        });
        
        // Add performance metrics if available
        if (data?.metrics) {
            const runMetrics = data.metrics as RunPerformanceMetrics;
            
            if (runMetrics.duration) {
                metrics.push({
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 2,
                    component: "RunStateMachine",
                    type: "performance",
                    name: "run.duration",
                    value: runMetrics.duration,
                    unit: "ms",
                    executionId: data?.runId,
                });
            }
            
            if (runMetrics.successRate) {
                metrics.push({
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 2,
                    component: "RunStateMachine",
                    type: "quality",
                    name: "run.success_rate",
                    value: runMetrics.successRate,
                    unit: "percentage",
                    executionId: data?.runId,
                });
            }
        }
        
        return metrics;
    }
    
    /**
     * Process step executed events
     */
    private processStepExecuted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        const metrics: UnifiedMetric[] = [];
        
        metrics.push({
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: 3,
            component: "UnifiedExecutor",
            type: "performance",
            name: "step.executed",
            value: 1,
            executionId: data?.stepId,
            metadata: {
                stepId: data?.stepId,
                runId: data?.runId,
                strategy: data?.strategy,
                duration: data?.duration,
                success: data?.success,
            },
        });
        
        if (data?.duration) {
            metrics.push({
                id: generatePK().toString(),
                timestamp: event.timestamp || new Date(),
                tier: 3,
                component: "UnifiedExecutor",
                type: "performance",
                name: "step.duration",
                value: data.duration,
                unit: "ms",
                executionId: data?.stepId,
            });
        }
        
        return metrics;
    }
    
    /**
     * Process strategy executed events
     */
    private processStrategyExecuted(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        const metrics: UnifiedMetric[] = [];
        
        if (data?.strategyMetrics) {
            const strategyMetrics = data.strategyMetrics as StrategyExecutionMetrics;
            
            metrics.push(
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 3,
                    component: "StrategyExecutor",
                    type: "performance",
                    name: "strategy.execution_time",
                    value: strategyMetrics.executionTime,
                    unit: "ms",
                    executionId: strategyMetrics.executionId,
                    metadata: { strategy: strategyMetrics.strategy },
                },
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: 3,
                    component: "StrategyExecutor",
                    type: "efficiency",
                    name: "strategy.credits_per_operation",
                    value: strategyMetrics.creditsPerOperation,
                    unit: "credits",
                    executionId: strategyMetrics.executionId,
                    metadata: { strategy: strategyMetrics.strategy },
                }
            );
        }
        
        return metrics;
    }
    
    /**
     * Process resource events
     */
    private processResourceEvent(event: BaseEvent): UnifiedMetric[] {
        const data = event.data as any;
        const metrics: UnifiedMetric[] = [];
        
        if (data?.resourceUsage) {
            const usage = data.resourceUsage as ResourceUsageMetrics;
            
            metrics.push(
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: usage.tier,
                    component: "ResourceManager",
                    type: "resource",
                    name: "resource.credits_used",
                    value: usage.credits.used,
                    unit: "credits",
                    executionId: usage.executionId,
                },
                {
                    id: generatePK().toString(),
                    timestamp: event.timestamp || new Date(),
                    tier: usage.tier,
                    component: "ResourceManager",
                    type: "resource",
                    name: "resource.tokens_total",
                    value: usage.tokens.total,
                    unit: "tokens",
                    executionId: usage.executionId,
                }
            );
        }
        
        return metrics;
    }
    
    /**
     * Process generic execution events
     */
    private processGenericExecutionEvent(event: BaseEvent): UnifiedMetric[] {
        return [{
            id: generatePK().toString(),
            timestamp: event.timestamp || new Date(),
            tier: this.extractTierFromEvent(event),
            component: event.source?.component || "unknown",
            type: "business",
            name: `execution.${event.subcategory}`,
            value: 1,
            metadata: event.data,
        }];
    }
    
    /**
     * Process health events
     */
    private processHealthEvent(
        event: BaseEvent,
        tier: 1 | 2 | 3 | "cross-cutting",
        timestamp: Date
    ): UnifiedMetric[] {
        return [{
            id: generatePK().toString(),
            timestamp,
            tier,
            component: event.source || "unknown",
            type: "health",
            name: `health.${event.subcategory || "status"}`,
            value: event.data?.status === "healthy" ? 1 : 0,
            metadata: event.data,
        }];
    }
    
    /**
     * Process performance events
     */
    private processPerformanceEvent(
        event: BaseEvent,
        tier: 1 | 2 | 3 | "cross-cutting",
        timestamp: Date
    ): UnifiedMetric[] {
        const metrics: UnifiedMetric[] = [];
        
        if (event.data && typeof event.data === "object") {
            for (const [key, value] of Object.entries(event.data)) {
                if (typeof value === "number") {
                    metrics.push({
                        id: generatePK().toString(),
                        timestamp,
                        tier,
                        component: event.source?.component || "unknown",
                        type: "performance",
                        name: `performance.${key}`,
                        value,
                        metadata: { eventId: event.id },
                    });
                }
            }
        }
        
        return metrics;
    }
    
    /**
     * Process generic resource events
     */
    private processResourceEventGeneric(
        event: BaseEvent,
        tier: 1 | 2 | 3 | "cross-cutting",
        timestamp: Date
    ): UnifiedMetric[] {
        return [{
            id: generatePK().toString(),
            timestamp,
            tier,
            component: event.source || "unknown",
            type: "resource",
            name: `resource.${event.subcategory || "usage"}`,
            value: typeof event.data?.value === "number" ? event.data.value : 1,
            metadata: event.data,
        }];
    }
    
    /**
     * Create a generic metric for unknown events
     */
    private createGenericMetric(
        event: BaseEvent,
        tier: 1 | 2 | 3 | "cross-cutting",
        timestamp: Date
    ): UnifiedMetric {
        return {
            id: generatePK().toString(),
            timestamp,
            tier,
            component: event.source || "unknown",
            type: "business",
            name: `${event.category}.${event.subcategory || "event"}`,
            value: 1,
            metadata: {
                eventId: event.id,
                originalCategory: event.category,
                originalSubcategory: event.subcategory,
                data: event.data,
            },
        };
    }
    
    /**
     * Extract tier information from event
     */
    private extractTierFromEvent(event: BaseEvent): 1 | 2 | 3 | "cross-cutting" {
        // Check event source for tier information
        if (event.source?.includes("Tier1") || event.source?.includes("Swarm")) {
            return 1;
        }
        if (event.source?.includes("Tier2") || event.source?.includes("Run")) {
            return 2;
        }
        if (event.source?.includes("Tier3") || event.source?.includes("Strategy") || event.source?.includes("Executor")) {
            return 3;
        }
        
        // Check metadata for tier information
        if (event.metadata?.tier) {
            return event.metadata.tier as 1 | 2 | 3;
        }
        
        // Default to cross-cutting
        return "cross-cutting";
    }
    
    /**
     * Record processing time for performance monitoring
     */
    private recordProcessingTime(timeMs: number): void {
        this.stats.processingTimeMs.push(timeMs);
        
        // Keep only recent processing times
        if (this.stats.processingTimeMs.length > 1000) {
            this.stats.processingTimeMs.shift();
        }
    }
    
    /**
     * Get processing statistics
     */
    getStats(): ProcessingStats & {
        avgProcessingTimeMs: number;
        maxProcessingTimeMs: number;
        errorRate: number;
    } {
        const times = this.stats.processingTimeMs;
        const avgProcessingTimeMs = times.length > 0 
            ? times.reduce((a, b) => a + b, 0) / times.length 
            : 0;
        const maxProcessingTimeMs = times.length > 0 ? Math.max(...times) : 0;
        const errorRate = this.stats.totalEvents > 0 
            ? this.stats.errors / this.stats.totalEvents 
            : 0;
        
        return {
            ...this.stats,
            avgProcessingTimeMs,
            maxProcessingTimeMs,
            errorRate,
        };
    }
}