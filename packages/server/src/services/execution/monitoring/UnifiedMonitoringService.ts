/**
 * Unified monitoring service that consolidates all monitoring functionality
 * across the three-tier execution architecture.
 * 
 * This service maintains the event-driven philosophy where intelligence
 * emerges from agents rather than being hard-coded.
 */

import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";
import { MetricsCollector } from "./core/MetricsCollector";
import { MetricsStore } from "./core/MetricsStore";
import { EventProcessor } from "./core/EventProcessor";
import { StatisticalEngine } from "./analytics/StatisticalEngine";
import { PatternDetector } from "./analytics/PatternDetector";
import { QueryInterface } from "./api/QueryInterface";
import { MCPToolAdapter } from "./api/MCPToolAdapter";
import {
    MonitoringConfig,
    UnifiedMetric,
    MetricQuery,
    MetricQueryResult,
    MetricType,
    MonitoringEvent,
    MetricSummary,
} from "./types";

/**
 * Default configuration for the monitoring service
 */
const DEFAULT_CONFIG: MonitoringConfig = {
    maxOverheadMs: 5,
    samplingRates: {
        performance: 1.0,
        resource: 1.0,
        health: 1.0,
        business: 0.1,
        safety: 1.0,
        quality: 0.5,
        efficiency: 0.5,
        intelligence: 0.1,
    },
    bufferSizes: {
        "tier-1": 10000,
        "tier-2": 50000,
        "tier-3": 100000,
        "cross-cutting": 50000,
    },
    retentionPolicies: [
        { tier: 1, metricType: "performance", retentionDays: 7 },
        { tier: 2, metricType: "performance", retentionDays: 3 },
        { tier: 3, metricType: "performance", retentionDays: 1 },
        { tier: "cross-cutting", metricType: "health", retentionDays: 30 },
        { tier: "cross-cutting", metricType: "safety", retentionDays: 90 },
    ],
    enabledMetricTypes: ["performance", "resource", "health", "business", "safety", "quality", "efficiency", "intelligence"],
    enableAnomalyDetection: true,
    enableAutoDownsampling: true,
    eventBusEnabled: true,
    mcpToolsEnabled: true,
};

/**
 * Unified monitoring service that consolidates all monitoring across tiers
 */
export class UnifiedMonitoringService {
    private static instance: UnifiedMonitoringService;
    
    private readonly collector: MetricsCollector;
    private readonly store: MetricsStore;
    private readonly processor: EventProcessor;
    private readonly stats: StatisticalEngine;
    private readonly patterns: PatternDetector;
    private readonly queryInterface: QueryInterface;
    private readonly mcpAdapter: MCPToolAdapter;
    private readonly eventBus?: EventBus;
    private readonly config: MonitoringConfig;
    private readonly logger?: Logger;
    
    private isInitialized = false;
    private initializationPromise?: Promise<void>;
    
    // Timer management for cleanup
    private retentionTimer?: NodeJS.Timeout;
    private downsamplingTimer?: NodeJS.Timeout;
    private anomalyDetectionTimer?: NodeJS.Timeout;
    
    /**
     * Private constructor for singleton pattern
     */
    private constructor(
        config?: Partial<MonitoringConfig>,
        eventBus?: EventBus,
        logger?: Logger
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.logger = logger;
        
        // Initialize core components
        this.collector = new MetricsCollector(this.config, eventBus);
        this.store = new MetricsStore(this.config);
        this.processor = new EventProcessor(this.store, this.config);
        
        // Initialize analytics
        this.stats = new StatisticalEngine();
        this.patterns = new PatternDetector(this.stats);
        
        // Initialize API layer
        this.queryInterface = new QueryInterface(this.store, this.stats);
        this.mcpAdapter = new MCPToolAdapter(this.queryInterface, this.patterns);
        
        // Store event bus instance if provided
        this.eventBus = eventBus;
    }
    
    /**
     * Get singleton instance
     */
    static getInstance(
        config?: Partial<MonitoringConfig>,
        eventBus?: EventBus,
        logger?: Logger
    ): UnifiedMonitoringService {
        if (!UnifiedMonitoringService.instance) {
            UnifiedMonitoringService.instance = new UnifiedMonitoringService(config, eventBus, logger);
        } else if (config || eventBus || logger) {
            // Use logger if available, otherwise console
            const log = logger || console;
            log.warn?.("UnifiedMonitoringService already initialized - ignoring configuration parameters. Use reset() to create new instance.");
        }
        return UnifiedMonitoringService.instance;
    }
    
    /**
     * Reset singleton instance (useful for testing and environment changes)
     */
    static async reset(): Promise<void> {
        const instance = UnifiedMonitoringService.instance;
        if (instance) {
            // Clear reference first to prevent race conditions
            UnifiedMonitoringService.instance = null as any;
            // Then shutdown the instance
            await instance.shutdown();
        }
    }
    
    /**
     * Initialize the service and start listening to events
     */
    async initialize(): Promise<void> {
        if (this.isInitialized) {
            return;
        }
        
        if (this.initializationPromise) {
            return this.initializationPromise;
        }
        
        this.initializationPromise = this._initialize();
        await this.initializationPromise;
    }
    
    private async _initialize(): Promise<void> {
        try {
            console.debug("Initializing UnifiedMonitoringService");
            
            // Initialize components
            await this.store.initialize();
            await this.processor.initialize();
            
            // Subscribe to events if enabled
            if (this.config.eventBusEnabled && this.eventBus) {
                await this.subscribeToEvents();
            }
            
            // Start background tasks
            this.startBackgroundTasks();
            
            this.isInitialized = true;
            console.info("UnifiedMonitoringService initialized successfully");
        } catch (error) {
            console.error("Failed to initialize UnifiedMonitoringService", { error });
            throw error;
        }
    }
    
    /**
     * Record a metric with minimal overhead
     */
    async record(metric: Omit<UnifiedMetric, "id" | "timestamp">): Promise<void> {
        return this.collector.record(metric);
    }
    
    /**
     * Record multiple metrics in batch
     */
    async recordBatch(metrics: Array<Omit<UnifiedMetric, "id" | "timestamp">>): Promise<void> {
        return this.collector.recordBatch(metrics);
    }
    
    /**
     * Query metrics with filtering and aggregation
     */
    async query(query: MetricQuery): Promise<MetricQueryResult> {
        return this.queryInterface.execute(query);
    }
    
    /**
     * Get statistical summary for a metric
     */
    async getSummary(
        metricName: string,
        startTime: Date,
        endTime: Date,
        groupBy?: string[]
    ): Promise<MetricSummary[]> {
        const metrics = await this.queryInterface.execute({
            name: metricName,
            startTime,
            endTime,
            groupBy: groupBy as any,
        });
        
        return this.stats.generateSummaries(metrics.metrics);
    }
    
    /**
     * Detect anomalies in recent metrics
     */
    async detectAnomalies(
        metricName: string,
        lookbackMinutes: number = 60
    ): Promise<UnifiedMetric[]> {
        if (!this.config.enableAnomalyDetection) {
            return [];
        }
        
        const endTime = new Date();
        const startTime = new Date(endTime.getTime() - lookbackMinutes * 60 * 1000);
        
        const metrics = await this.queryInterface.execute({
            name: metricName,
            startTime,
            endTime,
            orderBy: "timestamp",
            order: "asc",
        });
        
        return this.patterns.detectAnomalies(metrics.metrics);
    }
    
    /**
     * Get MCP tool adapter for monitoring agents
     */
    getMCPTools(): MCPToolAdapter {
        if (!this.config.mcpToolsEnabled) {
            throw new Error("MCP tools are disabled in monitoring configuration");
        }
        return this.mcpAdapter;
    }
    
    /**
     * Subscribe to relevant events from the event bus
     */
    private async subscribeToEvents(): Promise<void> {
        if (!this.eventBus) return;
        
        // Subscribe to telemetry events
        await this.eventBus.subscribe({
            id: "monitoring-telemetry",
            pattern: "telemetry.*",
            handler: async (event) => {
                await this.processor.processEvent(event);
            },
        });
        
        // Subscribe to execution events
        await this.eventBus.subscribe({
            id: "monitoring-execution",
            pattern: "execution.*",
            handler: async (event) => {
                await this.processor.processExecutionEvent(event);
            },
        });
        
        // Subscribe to health events
        await this.eventBus.subscribe({
            id: "monitoring-health",
            pattern: "health.*",
            handler: async (event) => {
                await this.processor.processEvent(event);
            },
        });
    }
    
    /**
     * Start background tasks for maintenance
     */
    private startBackgroundTasks(): void {
        // Retention policy enforcement
        this.retentionTimer = setInterval(async () => {
            try {
                await this.store.enforceRetentionPolicies();
            } catch (error) {
                console.error("Error enforcing retention policies", { error });
            }
        }, 60 * 60 * 1000); // Every hour
        
        // Downsampling
        if (this.config.enableAutoDownsampling) {
            this.downsamplingTimer = setInterval(async () => {
                try {
                    await this.store.downsampleOldMetrics();
                } catch (error) {
                    console.error("Error downsampling metrics", { error });
                }
            }, 4 * 60 * 60 * 1000); // Every 4 hours
        }
        
        // Anomaly detection for critical metrics
        if (this.config.enableAnomalyDetection) {
            this.anomalyDetectionTimer = setInterval(async () => {
                try {
                    await this.runAnomalyDetection();
                } catch (error) {
                    console.error("Error running anomaly detection", { error });
                }
            }, 5 * 60 * 1000); // Every 5 minutes
        }
    }
    
    /**
     * Run anomaly detection on critical metrics
     */
    private async runAnomalyDetection(): Promise<void> {
        const criticalMetrics = [
            "swarm.task.error_rate",
            "run.step.failure_rate",
            "resource.credits.utilization",
            "execution.latency.p99",
        ];
        
        for (const metric of criticalMetrics) {
            const anomalies = await this.detectAnomalies(metric, 30);
            
            if (anomalies.length > 0) {
                // Emit anomaly events for monitoring agents
                const event: MonitoringEvent = {
                    id: `anomaly-${Date.now()}`,
                    type: "monitoring.anomaly",
                    timestamp: new Date(),
                    source: {
                        tier: "cross-cutting",
                        component: "UnifiedMonitoringService",
                        instanceId: "singleton",
                    },
                    data: {
                        anomaly: {
                            timestamp: new Date(),
                            metric: anomalies[0],
                            score: anomalies.length / 10, // Simple score based on count
                            type: "pattern",
                            confidence: 0.8,
                            context: { detectedAnomalies: anomalies.length },
                        },
                    },
                    metadata: {
                        version: "1.0.0",
                        tags: ["monitoring", "anomaly"],
                        priority: "HIGH",
                    },
                };
                
                if (this.eventBus) {
                    await this.eventBus.publish(event);
                }
            }
        }
    }
    
    /**
     * Shutdown the service gracefully
     */
    async shutdown(): Promise<void> {
        console.info("Shutting down UnifiedMonitoringService");
        
        // Clear background timers
        if (this.retentionTimer) {
            clearInterval(this.retentionTimer);
            this.retentionTimer = undefined;
        }
        if (this.downsamplingTimer) {
            clearInterval(this.downsamplingTimer);
            this.downsamplingTimer = undefined;
        }
        if (this.anomalyDetectionTimer) {
            clearInterval(this.anomalyDetectionTimer);
            this.anomalyDetectionTimer = undefined;
        }
        
        // Flush any pending metrics
        await this.collector.flush();
        
        // Close store connections
        await this.store.close();
        
        // Unsubscribe from events
        if (this.eventBus) {
            await this.eventBus.unsubscribe("monitoring-telemetry");
            await this.eventBus.unsubscribe("monitoring-execution");
            await this.eventBus.unsubscribe("monitoring-health");
        }
        
        this.isInitialized = false;
        
        console.info("UnifiedMonitoringService shutdown complete");
    }
    
    /**
     * Get service health status
     */
    async getHealth(): Promise<{
        status: "healthy" | "degraded" | "unhealthy";
        details: Record<string, any>;
    }> {
        const collectorHealth = await this.collector.getHealth();
        const storeHealth = await this.store.getHealth();
        
        const overallHealth = 
            collectorHealth.status === "healthy" && storeHealth.status === "healthy"
                ? "healthy"
                : collectorHealth.status === "unhealthy" || storeHealth.status === "unhealthy"
                ? "unhealthy"
                : "degraded";
        
        return {
            status: overallHealth,
            details: {
                collector: collectorHealth,
                store: storeHealth,
                initialized: this.isInitialized,
                config: {
                    eventBusEnabled: this.config.eventBusEnabled,
                    mcpToolsEnabled: this.config.mcpToolsEnabled,
                    anomalyDetectionEnabled: this.config.enableAnomalyDetection,
                },
            },
        };
    }
}