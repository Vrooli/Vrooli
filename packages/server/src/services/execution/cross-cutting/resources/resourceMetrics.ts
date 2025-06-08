import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import {
    type ResourceType,
    type ExecutionResourceUsage,
    generatePk,
} from "@vrooli/shared";

/**
 * Performance metric
 */
export interface PerformanceMetric {
    name: string;
    value: number;
    unit: string;
    timestamp: Date;
    metadata?: Record<string, unknown>;
}

/**
 * Resource efficiency metrics
 */
export interface EfficiencyMetrics {
    resourceType: ResourceType;
    utilizationRate: number; // 0-1
    wasteRate: number; // 0-1
    costEfficiency: number; // cost per unit of work
    throughput: number; // work units per time
    reliability: number; // success rate
}

/**
 * Resource health metrics
 */
export interface ResourceHealthMetrics {
    availability: number; // 0-1
    responseTime: number; // milliseconds
    errorRate: number; // 0-1
    saturation: number; // 0-1
    queueLength: number;
    activeConnections: number;
}

/**
 * Cost analysis
 */
export interface CostAnalysis {
    totalCost: number;
    costByResource: Map<ResourceType, number>;
    costTrend: Array<{ timestamp: Date; cost: number }>;
    projectedMonthlyCost: number;
    savingsOpportunities: Array<{
        type: string;
        potential: number;
        description: string;
    }>;
}

/**
 * Resource bottleneck analysis
 */
export interface BottleneckAnalysis {
    resourceType: ResourceType;
    severity: "low" | "medium" | "high" | "critical";
    utilizationRate: number;
    queueLength: number;
    averageWaitTime: number;
    recommendations: string[];
}

/**
 * Shared resource metrics collection and analysis
 * 
 * Provides:
 * - Performance monitoring
 * - Efficiency analysis
 * - Cost tracking
 * - Bottleneck detection
 * - Health monitoring
 */
export class ResourceMetrics {
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    private readonly metricsId: string;
    
    // Metrics storage
    private readonly performanceMetrics: Map<string, PerformanceMetric[]> = new Map();
    private readonly efficiencyHistory: EfficiencyMetrics[] = [];
    private readonly healthHistory: ResourceHealthMetrics[] = [];
    private readonly costHistory: Array<{ timestamp: Date; cost: number }> = [];
    
    // Configuration
    private readonly maxHistorySize: number = 1000;
    private readonly metricsInterval: number = 60000; // 1 minute
    private metricsTimer?: NodeJS.Timeout;
    
    constructor(
        metricsId: string,
        logger: Logger,
        eventBus?: EventBus,
    ) {
        this.metricsId = metricsId;
        this.logger = logger;
        this.eventBus = eventBus;
        
        this.startMetricsCollection();
        
        this.logger.debug("[ResourceMetrics] Initialized", {
            metricsId,
        });
    }
    
    /**
     * Record performance metric
     */
    async recordMetric(
        name: string,
        value: number,
        unit: string,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        const metric: PerformanceMetric = {
            name,
            value,
            unit,
            timestamp: new Date(),
            metadata,
        };
        
        if (!this.performanceMetrics.has(name)) {
            this.performanceMetrics.set(name, []);
        }
        
        const metrics = this.performanceMetrics.get(name)!;
        metrics.push(metric);
        
        // Limit history size
        if (metrics.length > this.maxHistorySize) {
            metrics.shift();
        }
        
        // Emit metric event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'metrics.recorded',
                timestamp: new Date(),
                data: {
                    metricsId: this.metricsId,
                    metric,
                },
            });
        }
        
        this.logger.debug("[ResourceMetrics] Recorded metric", {
            name,
            value,
            unit,
        });
    }
    
    /**
     * Record efficiency metrics
     */
    async recordEfficiency(efficiency: EfficiencyMetrics): Promise<void> {
        this.efficiencyHistory.push(efficiency);
        
        // Limit history size
        if (this.efficiencyHistory.length > this.maxHistorySize) {
            this.efficiencyHistory.shift();
        }
        
        // Emit efficiency event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'metrics.efficiency',
                timestamp: new Date(),
                data: {
                    metricsId: this.metricsId,
                    efficiency,
                },
            });
        }
    }
    
    /**
     * Record health metrics
     */
    async recordHealth(health: ResourceHealthMetrics): Promise<void> {
        this.healthHistory.push(health);
        
        // Limit history size
        if (this.healthHistory.length > this.maxHistorySize) {
            this.healthHistory.shift();
        }
        
        // Check for alerts
        this.checkHealthAlerts(health);
        
        // Emit health event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'metrics.health',
                timestamp: new Date(),
                data: {
                    metricsId: this.metricsId,
                    health,
                },
            });
        }
    }
    
    /**
     * Record execution usage and derive metrics
     */
    async recordExecution(
        usage: ExecutionResourceUsage,
        duration: number,
        success: boolean,
    ): Promise<void> {
        const cost = usage.cost || Number(usage.creditsUsed) || 0;
        
        // Record cost
        this.costHistory.push({
            timestamp: new Date(),
            cost,
        });
        
        // Record performance metrics
        await this.recordMetric("execution_duration", duration, "ms");
        await this.recordMetric("execution_cost", cost, "credits");
        await this.recordMetric("memory_peak", usage.memoryUsedMB, "MB");
        
        if (usage.tokens) {
            await this.recordMetric("tokens_used", usage.tokens, "count");
        }
        
        if (usage.apiCalls) {
            await this.recordMetric("api_calls", usage.apiCalls, "count");
        }
        
        // Calculate and record efficiency
        const efficiency = this.calculateExecutionEfficiency(usage, duration, success);
        await this.recordEfficiency(efficiency);
        
        // Calculate throughput
        const throughput = success ? (1000 / duration) : 0; // executions per second
        await this.recordMetric("throughput", throughput, "executions/s");
    }
    
    /**
     * Get current performance metrics
     */
    getPerformanceMetrics(metricName?: string): Map<string, PerformanceMetric[]> {
        if (metricName) {
            const metrics = this.performanceMetrics.get(metricName);
            return metrics ? new Map([[metricName, metrics]]) : new Map();
        }
        return new Map(this.performanceMetrics);
    }
    
    /**
     * Get efficiency trend
     */
    getEfficiencyTrend(resourceType?: ResourceType): EfficiencyMetrics[] {
        if (resourceType) {
            return this.efficiencyHistory.filter(e => e.resourceType === resourceType);
        }
        return [...this.efficiencyHistory];
    }
    
    /**
     * Get health trend
     */
    getHealthTrend(): ResourceHealthMetrics[] {
        return [...this.healthHistory];
    }
    
    /**
     * Analyze costs
     */
    getCostAnalysis(): CostAnalysis {
        const totalCost = this.costHistory.reduce((sum, entry) => sum + entry.cost, 0);
        
        // Calculate cost by resource (simplified)
        const costByResource = new Map<ResourceType, number>();
        costByResource.set(ResourceType.CREDITS, totalCost);
        
        // Get trend (last 30 data points)
        const costTrend = this.costHistory.slice(-30);
        
        // Project monthly cost
        const recentCosts = this.costHistory.slice(-100); // Last 100 executions
        const avgCost = recentCosts.length > 0 
            ? recentCosts.reduce((sum, entry) => sum + entry.cost, 0) / recentCosts.length
            : 0;
        const projectedMonthlyCost = avgCost * 30 * 24 * 60; // Rough estimate
        
        // Savings opportunities
        const savingsOpportunities = this.calculateSavingsOpportunities();
        
        return {
            totalCost,
            costByResource,
            costTrend,
            projectedMonthlyCost,
            savingsOpportunities,
        };
    }
    
    /**
     * Detect bottlenecks
     */
    detectBottlenecks(): BottleneckAnalysis[] {
        const bottlenecks: BottleneckAnalysis[] = [];
        
        // Analyze each resource type
        for (const resourceType of Object.values(ResourceType)) {
            const analysis = this.analyzeResourceBottleneck(resourceType as ResourceType);
            if (analysis.severity !== "low") {
                bottlenecks.push(analysis);
            }
        }
        
        return bottlenecks;
    }
    
    /**
     * Get resource utilization
     */
    getResourceUtilization(resourceType: ResourceType): number {
        const efficiencyMetrics = this.efficiencyHistory
            .filter(e => e.resourceType === resourceType)
            .slice(-10); // Last 10 measurements
        
        if (efficiencyMetrics.length === 0) {
            return 0;
        }
        
        const avgUtilization = efficiencyMetrics
            .reduce((sum, e) => sum + e.utilizationRate, 0) / efficiencyMetrics.length;
        
        return avgUtilization;
    }
    
    /**
     * Shutdown metrics collection
     */
    shutdown(): void {
        if (this.metricsTimer) {
            clearInterval(this.metricsTimer);
            this.metricsTimer = undefined;
        }
        
        this.logger.info("[ResourceMetrics] Shutdown", {
            metricsId: this.metricsId,
            metricsCount: this.performanceMetrics.size,
        });
    }
    
    /**
     * Calculate execution efficiency
     */
    private calculateExecutionEfficiency(
        usage: ExecutionResourceUsage,
        duration: number,
        success: boolean,
    ): EfficiencyMetrics {
        const credits = Number(usage.creditsUsed) || 0;
        const memory = usage.memoryUsedMB || 0;
        
        // Calculate utilization (simplified)
        const utilizationRate = success ? 0.8 : 0.2; // Success = high utilization
        
        // Calculate waste rate
        const wasteRate = success ? 0.1 : 0.8; // Failure = high waste
        
        // Calculate cost efficiency (credits per successful execution)
        const costEfficiency = success && credits > 0 ? credits : Number.MAX_SAFE_INTEGER;
        
        // Calculate throughput (work per time)
        const throughput = success ? (1000 / duration) : 0;
        
        // Calculate reliability (success rate - simplified)
        const reliability = success ? 1.0 : 0.0;
        
        return {
            resourceType: ResourceType.CREDITS,
            utilizationRate,
            wasteRate,
            costEfficiency,
            throughput,
            reliability,
        };
    }
    
    /**
     * Analyze resource bottleneck
     */
    private analyzeResourceBottleneck(resourceType: ResourceType): BottleneckAnalysis {
        const utilization = this.getResourceUtilization(resourceType);
        
        let severity: "low" | "medium" | "high" | "critical";
        if (utilization < 0.5) severity = "low";
        else if (utilization < 0.7) severity = "medium";
        else if (utilization < 0.9) severity = "high";
        else severity = "critical";
        
        const recommendations: string[] = [];
        if (utilization > 0.8) {
            recommendations.push("Consider increasing resource capacity");
            recommendations.push("Optimize resource usage patterns");
        }
        
        return {
            resourceType,
            severity,
            utilizationRate: utilization,
            queueLength: 0, // Would need actual queue data
            averageWaitTime: 0, // Would need actual timing data
            recommendations,
        };
    }
    
    /**
     * Calculate savings opportunities
     */
    private calculateSavingsOpportunities(): Array<{
        type: string;
        potential: number;
        description: string;
    }> {
        const opportunities: Array<{
            type: string;
            potential: number;
            description: string;
        }> = [];
        
        // Analyze efficiency for savings
        const avgEfficiency = this.efficiencyHistory.length > 0
            ? this.efficiencyHistory.reduce((sum, e) => sum + e.utilizationRate, 0) / this.efficiencyHistory.length
            : 0;
        
        if (avgEfficiency < 0.7) {
            opportunities.push({
                type: "efficiency",
                potential: (0.7 - avgEfficiency) * 100,
                description: "Improve resource utilization efficiency",
            });
        }
        
        // Check for waste
        const avgWaste = this.efficiencyHistory.length > 0
            ? this.efficiencyHistory.reduce((sum, e) => sum + e.wasteRate, 0) / this.efficiencyHistory.length
            : 0;
        
        if (avgWaste > 0.2) {
            opportunities.push({
                type: "waste_reduction",
                potential: avgWaste * 100,
                description: "Reduce resource waste through better planning",
            });
        }
        
        return opportunities;
    }
    
    /**
     * Check for health alerts
     */
    private checkHealthAlerts(health: ResourceHealthMetrics): void {
        const alerts: string[] = [];
        
        if (health.availability < 0.9) {
            alerts.push("Low availability");
        }
        
        if (health.errorRate > 0.05) {
            alerts.push("High error rate");
        }
        
        if (health.saturation > 0.8) {
            alerts.push("High resource saturation");
        }
        
        if (alerts.length > 0 && this.eventBus) {
            this.eventBus.emit({
                type: 'metrics.alert',
                timestamp: new Date(),
                data: {
                    metricsId: this.metricsId,
                    alerts,
                    health,
                },
            }).catch(err => {
                this.logger.error("[ResourceMetrics] Failed to emit alert", { error: err });
            });
        }
    }
    
    /**
     * Start metrics collection timer
     */
    private startMetricsCollection(): void {
        this.metricsTimer = setInterval(() => {
            this.collectSystemMetrics();
        }, this.metricsInterval);
    }
    
    /**
     * Collect system metrics
     */
    private async collectSystemMetrics(): Promise<void> {
        // Collect basic system metrics
        const memoryUsage = process.memoryUsage();
        
        await this.recordMetric("heap_used", memoryUsage.heapUsed / 1024 / 1024, "MB");
        await this.recordMetric("heap_total", memoryUsage.heapTotal / 1024 / 1024, "MB");
        await this.recordMetric("external", memoryUsage.external / 1024 / 1024, "MB");
        
        // Record health metrics
        const health: ResourceHealthMetrics = {
            availability: 1.0, // Always available for in-memory metrics
            responseTime: 0, // Immediate for in-memory
            errorRate: 0, // No errors in basic collection
            saturation: memoryUsage.heapUsed / memoryUsage.heapTotal,
            queueLength: 0,
            activeConnections: 1,
        };
        
        await this.recordHealth(health);
    }
}