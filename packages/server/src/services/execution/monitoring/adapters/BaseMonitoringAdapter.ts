/**
 * BaseMonitoringAdapter - Base class for all monitoring component adapters
 * 
 * Provides common functionality for adapters that migrate legacy monitoring
 * components to the UnifiedMonitoringService architecture.
 */

import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { UnifiedMonitoringService } from "../UnifiedMonitoringService.js";
import { type UnifiedMetric, type MetricType } from "../types.js";

/**
 * Base configuration for monitoring adapters
 */
export interface BaseMonitoringConfig {
    componentName: string;
    tier: 1 | 2 | 3 | "cross-cutting";
    eventChannels?: string[];
    enableEventBus?: boolean;
}

/**
 * Base class for monitoring adapters
 */
export abstract class BaseMonitoringAdapter {
    protected readonly unifiedService: UnifiedMonitoringService;
    protected readonly logger?: Logger;
    protected readonly eventBus?: EventBus;
    protected readonly config: BaseMonitoringConfig;
    protected subscriptionIds: string[] = [];

    constructor(
        config: BaseMonitoringConfig,
        logger?: Logger,
        eventBus?: EventBus
    ) {
        this.config = config;
        this.logger = logger;
        this.eventBus = eventBus;

        // Get unified monitoring service instance
        this.unifiedService = UnifiedMonitoringService.getInstance(
            {
                maxOverheadMs: 5,
                eventBusEnabled: config.enableEventBus ?? true,
                mcpToolsEnabled: true,
            },
            this.eventBus,
            this.logger
        );

        // Subscribe to events if configured
        if (config.eventChannels && eventBus) {
            this.subscribeToEvents(config.eventChannels);
        }
    }

    /**
     * Record a metric through the unified service
     */
    protected async recordMetric(
        name: string,
        value: number | string | boolean,
        type: MetricType,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        try {
            await this.unifiedService.record({
                tier: this.config.tier,
                component: this.config.componentName,
                type,
                name,
                value,
                metadata: {
                    ...metadata,
                    source: `${this.config.componentName}_adapter`,
                },
            });
        } catch (error) {
            this.logger?.error(`[${this.config.componentName}Adapter] Failed to record metric:`, error);
        }
    }

    /**
     * Query metrics from the unified service
     */
    protected async queryMetrics(query: {
        name?: string | string[];
        startTime?: Date;
        endTime?: Date;
        limit?: number;
        metadata?: Record<string, unknown>;
    }): Promise<UnifiedMetric[]> {
        try {
            const result = await this.unifiedService.query({
                tier: [this.config.tier],
                component: [this.config.componentName],
                name: query.name,
                startTime: query.startTime,
                endTime: query.endTime,
                limit: query.limit,
                orderBy: "timestamp",
                order: "desc",
            });

            return result.metrics;
        } catch (error) {
            this.logger?.error(`[${this.config.componentName}Adapter] Failed to query metrics:`, error);
            return [];
        }
    }

    /**
     * Emit an event through the event bus with type safety
     */
    protected async emitEvent(
        channel: string,
        event: any
    ): Promise<void> {
        if (!this.eventBus || !event) {
            return;
        }

        try {
            // Create a safe event object with required fields
            const safeEvent = {
                id: event.id || `${this.config.componentName}_${Date.now()}`,
                type: channel,
                source: {
                    tier: this.config.tier,
                    component: this.config.componentName,
                    instanceId: `${this.config.componentName}_adapter`,
                },
                timestamp: event.timestamp || new Date(),
                data: event.data || event,
                metadata: {
                    ...event.metadata,
                    adapterVersion: '1.0.0',
                    originalType: event.type,
                },
            };
            
            await this.eventBus.publish(safeEvent);
        } catch (error) {
            this.logger?.error(`[${this.config.componentName}Adapter] Failed to emit event:`, error);
        }
    }

    /**
     * Subscribe to event channels
     */
    private subscribeToEvents(channels: string[]): void {
        if (!this.eventBus) {
            return;
        }

        for (const channel of channels) {
            const subscriptionId = `${this.config.componentName}_adapter_${channel}_${Date.now()}`;
            this.subscriptionIds.push(subscriptionId);
            
            this.eventBus.subscribe({
                id: subscriptionId,
                pattern: channel,
                handler: async (event: any) => {
                    await this.handleEvent(channel, event);
                },
            }).catch(error => {
                this.logger?.error(
                    `[${this.config.componentName}Adapter] Failed to subscribe to ${channel}:`,
                    error
                );
            });
        }
    }

    /**
     * Handle incoming events - to be implemented by subclasses
     */
    protected abstract handleEvent(channel: string, event: any): Promise<void>;

    /**
     * Determine metric type based on metric name or context
     */
    protected getMetricType(metricName: string, context?: any): MetricType {
        const name = metricName.toLowerCase();
        
        if (name.includes('error') || name.includes('failure') || name.includes('health')) {
            return 'health';
        }
        if (name.includes('security') || name.includes('validation') || name.includes('safety')) {
            return 'safety';
        }
        if (name.includes('perf') || name.includes('duration') || name.includes('latency') || 
            name.includes('throughput') || name.includes('resource')) {
            return 'performance';
        }
        
        return 'business';
    }

    /**
     * Extract numeric value from various formats
     */
    protected extractNumericValue(value: any): number {
        if (typeof value === 'number') {
            return value;
        }
        if (typeof value === 'boolean') {
            return value ? 1 : 0;
        }
        if (typeof value === 'string') {
            const parsed = parseFloat(value);
            return isNaN(parsed) ? 0 : parsed;
        }
        if (value && typeof value === 'object') {
            // Try common value fields
            if ('value' in value) return this.extractNumericValue(value.value);
            if ('count' in value) return this.extractNumericValue(value.count);
            if ('total' in value) return this.extractNumericValue(value.total);
            if ('duration' in value) return this.extractNumericValue(value.duration);
        }
        return 0;
    }

    /**
     * Convert legacy format to unified metric metadata
     */
    protected buildMetadata(
        originalData: any,
        additionalMetadata?: Record<string, unknown>
    ): Record<string, unknown> {
        return {
            timestamp: new Date(),
            ...additionalMetadata,
            originalData,
            adapterVersion: '1.0.0',
        };
    }

    /**
     * Cleanup subscriptions and resources
     */
    async cleanup(): Promise<void> {
        // Unsubscribe from all events
        for (const id of this.subscriptionIds) {
            try {
                await this.eventBus?.unsubscribe(id);
            } catch (error) {
                this.logger?.error(`[${this.config.componentName}Adapter] Failed to unsubscribe ${id}:`, error);
            }
        }
        this.subscriptionIds = [];
        
        // Call any subclass-specific cleanup
        await this.onCleanup();
    }
    
    /**
     * Hook for subclass-specific cleanup
     */
    protected async onCleanup(): Promise<void> {
        // Default empty implementation - subclasses can override
    }

    /**
     * Shutdown the adapter and clean up resources
     */
    async shutdown(): Promise<void> {
        this.logger?.info(`[${this.config.componentName}Adapter] Shutting down`);
        await this.cleanup();
    }
}