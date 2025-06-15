/**
 * Circuit Breaker Manager
 * 
 * Centralized management of multiple circuit breakers across services and operations.
 * Provides unified interface, dynamic configuration, and coordination with error
 * classification system for intelligent circuit breaker orchestration.
 * 
 * Key Features:
 * - Manages multiple circuit breakers across services
 * - Dynamic configuration updates and circuit breaker lifecycle
 * - Integration with error classification for intelligent triggering
 * - Unified metrics and health monitoring
 * - Bulk operations and circuit breaker coordination
 * - Agent-driven pattern optimization
 */

import { AdaptiveCircuitBreaker, CircuitBreakerFactory } from "./circuitBreaker.js";
import { ErrorClassifier } from "./errorClassifier.js";
import { EventBus } from "../events/eventBus.js";
import {
    CircuitState,
    CircuitBreakerConfig,
    CircuitBreakerState,
    ErrorClassification,
    ErrorContext,
    ErrorSeverity,
    ErrorCategory,
    ResilienceEventType,
    ResilienceMetrics,
    MonitoringEventPrefix,
} from "@vrooli/shared";

/**
 * Circuit breaker registry entry
 */
interface CircuitBreakerEntry {
    circuitBreaker: AdaptiveCircuitBreaker;
    config: CircuitBreakerConfig;
    createdAt: Date;
    lastUsed: Date;
    usageCount: number;
    tags: string[];
    metadata: Record<string, unknown>;
}

/**
 * Circuit breaker pattern for intelligent configuration
 */
interface CircuitBreakerPattern {
    id: string;
    name: string;
    description: string;
    serviceMatcher: RegExp;
    operationMatcher: RegExp;
    errorTypes: string[];
    recommendedConfig: CircuitBreakerConfig;
    successRate: number;
    usageCount: number;
    lastUpdated: Date;
    confidence: number;
}

/**
 * Bulk operation result
 */
interface BulkOperationResult {
    successful: string[];
    failed: Array<{ key: string; error: string }>;
    totalProcessed: number;
    duration: number;
}

/**
 * Circuit breaker health summary
 */
interface HealthSummary {
    total: number;
    healthy: number;
    degraded: number;
    unhealthy: number;
    byService: Record<string, {
        healthy: number;
        degraded: number;
        unhealthy: number;
    }>;
    alerting: string[];
    recommendations: string[];
}

/**
 * Circuit Breaker Manager
 * 
 * Centralized management system for circuit breakers with intelligent coordination:
 * - Service discovery and automatic circuit breaker creation
 * - Pattern-based configuration optimization
 * - Error classification integration for smart triggering
 * - Bulk operations and health monitoring
 * - Agent learning integration for pattern evolution
 */
export class CircuitBreakerManager {
    private readonly registry = new Map<string, CircuitBreakerEntry>();
    private readonly factory: CircuitBreakerFactory;
    private readonly eventBus: EventBus;
    private readonly errorClassifier: ErrorClassifier;
    
    // Pattern management
    private patterns: CircuitBreakerPattern[] = [];
    private defaultConfigs = new Map<string, CircuitBreakerConfig>();
    
    // Health monitoring
    private healthCheckTimer?: NodeJS.Timeout;
    private lastHealthCheck = new Date();
    private healthSummary: HealthSummary;
    
    // Metrics
    private metrics: ResilienceMetrics;
    
    constructor(
        eventBus: EventBus,
        errorClassifier: ErrorClassifier,
        private readonly healthCheckInterval = 30000, // 30 seconds
    ) {
        this.eventBus = eventBus;
        this.errorClassifier = errorClassifier;
        this.factory = new CircuitBreakerFactory(eventBus);
        
        this.healthSummary = this.initializeHealthSummary();
        this.metrics = this.initializeMetrics();
        
        this.startHealthMonitoring();
        this.loadDefaultPatterns();
    }

    /**
     * Get or create circuit breaker for service/operation
     */
    async getCircuitBreaker(
        service: string,
        operation: string,
        config?: Partial<CircuitBreakerConfig>,
    ): Promise<AdaptiveCircuitBreaker> {
        const key = this.createKey(service, operation);
        const existing = this.registry.get(key);
        
        if (existing) {
            existing.lastUsed = new Date();
            existing.usageCount++;
            return existing.circuitBreaker;
        }
        
        // Create new circuit breaker with intelligent configuration
        const effectiveConfig = await this.resolveConfiguration(service, operation, config);
        const circuitBreaker = this.factory.create(service, operation, effectiveConfig);
        
        // Register circuit breaker
        const entry: CircuitBreakerEntry = {
            circuitBreaker,
            config: effectiveConfig as CircuitBreakerConfig,
            createdAt: new Date(),
            lastUsed: new Date(),
            usageCount: 1,
            tags: this.generateTags(service, operation),
            metadata: {
                pattern: this.findMatchingPattern(service, operation)?.id,
                autoCreated: true,
            },
        };
        
        this.registry.set(key, entry);
        
        // All telemetry now handled by event-driven agents
        
        return circuitBreaker;
    }

    /**
     * Execute operation with automatic circuit breaker protection
     */
    async executeWithProtection<T>(
        service: string,
        operation: string,
        operationFn: () => Promise<T>,
        config?: Partial<CircuitBreakerConfig>,
    ): Promise<T> {
        const circuitBreaker = await this.getCircuitBreaker(service, operation, config);
        
        try {
            const result = await circuitBreaker.execute(operationFn);
            
            // Update success metrics
            this.updateMetrics(true, service, operation);
            return result;
            
        } catch (error) {
            // Classify error for intelligent handling
            const classification = await this.classifyError(error, service, operation);
            await this.handleClassifiedError(classification, service, operation, error);
            
            // Update failure metrics
            this.updateMetrics(false, service, operation);
            throw error;
        }
    }

    /**
     * Bulk force state change across multiple circuit breakers
     */
    async bulkForceState(
        pattern: {
            services?: string[];
            operations?: string[];
            states?: CircuitState[];
            tags?: string[];
        },
        newState: CircuitState,
        reason: string,
    ): Promise<BulkOperationResult> {
        const startTime = Date.now();
        const result: BulkOperationResult = {
            successful: [],
            failed: [],
            totalProcessed: 0,
            duration: 0,
        };
        
        const matchingEntries = this.findMatchingEntries(pattern);
        
        for (const [key, entry] of matchingEntries) {
            result.totalProcessed++;
            
            try {
                await entry.circuitBreaker.forceState(newState, reason, {
                    bulkOperation: true,
                    pattern,
                });
                result.successful.push(key);
                
            } catch (error) {
                result.failed.push({
                    key,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        
        result.duration = Date.now() - startTime;
        
        // All telemetry now handled by event-driven agents
        
        return result;
    }

    /**
     * Update configuration for multiple circuit breakers
     */
    async bulkUpdateConfig(
        pattern: {
            services?: string[];
            operations?: string[];
            tags?: string[];
        },
        configUpdates: Partial<CircuitBreakerConfig>,
    ): Promise<BulkOperationResult> {
        const startTime = Date.now();
        const result: BulkOperationResult = {
            successful: [],
            failed: [],
            totalProcessed: 0,
            duration: 0,
        };
        
        const matchingEntries = this.findMatchingEntries(pattern);
        
        for (const [key, entry] of matchingEntries) {
            result.totalProcessed++;
            
            try {
                entry.circuitBreaker.updateConfig(configUpdates);
                Object.assign(entry.config, configUpdates);
                result.successful.push(key);
                
            } catch (error) {
                result.failed.push({
                    key,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        
        result.duration = Date.now() - startTime;
        return result;
    }

    /**
     * Get comprehensive circuit breaker states
     */
    getAllStates(): Map<string, CircuitBreakerState> {
        const states = new Map<string, CircuitBreakerState>();
        
        for (const [key, entry] of this.registry) {
            states.set(key, entry.circuitBreaker.getState());
        }
        
        return states;
    }

    /**
     * Get circuit breakers by service
     */
    getByService(service: string): Map<string, AdaptiveCircuitBreaker> {
        const result = new Map<string, AdaptiveCircuitBreaker>();
        
        for (const [key, entry] of this.registry) {
            if (key.startsWith(`${service}:`)) {
                result.set(key, entry.circuitBreaker);
            }
        }
        
        return result;
    }

    /**
     * Get health summary across all circuit breakers
     */
    getHealthSummary(): HealthSummary {
        const summary: HealthSummary = {
            total: this.registry.size,
            healthy: 0,
            degraded: 0,
            unhealthy: 0,
            byService: {},
            alerting: [],
            recommendations: [],
        };
        
        for (const [key, entry] of this.registry) {
            const state = entry.circuitBreaker.getState();
            const [service] = key.split(":");
            
            if (!summary.byService[service]) {
                summary.byService[service] = { healthy: 0, degraded: 0, unhealthy: 0 };
            }
            
            switch (state.state) {
                case CircuitState.CLOSED:
                    summary.healthy++;
                    summary.byService[service].healthy++;
                    break;
                case CircuitState.HALF_OPEN:
                    summary.degraded++;
                    summary.byService[service].degraded++;
                    break;
                case CircuitState.OPEN:
                    summary.unhealthy++;
                    summary.byService[service].unhealthy++;
                    summary.alerting.push(key);
                    break;
            }
        }
        
        // Generate recommendations
        this.generateHealthRecommendations(summary);
        
        return summary;
    }

    /**
     * Add circuit breaker pattern for intelligent configuration
     */
    addPattern(pattern: Omit<CircuitBreakerPattern, "lastUpdated">): void {
        const fullPattern: CircuitBreakerPattern = {
            ...pattern,
            lastUpdated: new Date(),
        };
        
        // Remove existing pattern with same ID
        this.patterns = this.patterns.filter(p => p.id !== pattern.id);
        this.patterns.push(fullPattern);
        
        // Sort by confidence descending
        this.patterns.sort((a, b) => b.confidence - a.confidence);
    }

    /**
     * Remove circuit breaker (for cleanup)
     */
    async removeCircuitBreaker(service: string, operation: string): Promise<boolean> {
        const key = this.createKey(service, operation);
        const entry = this.registry.get(key);
        
        if (!entry) {
            return false;
        }
        
        await entry.circuitBreaker.shutdown();
        this.registry.delete(key);
        
        // All telemetry now handled by event-driven agents
        
        return true;
    }

    /**
     * Cleanup unused circuit breakers
     */
    async cleanup(maxIdleTime = 3600000): Promise<BulkOperationResult> { // 1 hour default
        const cutoff = Date.now() - maxIdleTime;
        const result: BulkOperationResult = {
            successful: [],
            failed: [],
            totalProcessed: 0,
            duration: 0,
        };
        
        const startTime = Date.now();
        
        for (const [key, entry] of this.registry) {
            if (entry.lastUsed.getTime() < cutoff) {
                result.totalProcessed++;
                
                try {
                    await entry.circuitBreaker.shutdown();
                    this.registry.delete(key);
                    result.successful.push(key);
                    
                } catch (error) {
                    result.failed.push({
                        key,
                        error: error instanceof Error ? error.message : String(error),
                    });
                }
            }
        }
        
        result.duration = Date.now() - startTime;
        return result;
    }

    /**
     * Get comprehensive metrics
     */
    getMetrics(): ResilienceMetrics & {
        circuitBreakerCount: number;
        patternCount: number;
        healthSummary: HealthSummary;
        topFailingServices: Array<{ service: string; failureRate: number }>;
    } {
        return {
            ...this.metrics,
            circuitBreakerCount: this.registry.size,
            patternCount: this.patterns.length,
            healthSummary: this.getHealthSummary(),
            topFailingServices: this.getTopFailingServices(),
        };
    }

    /**
     * Shutdown all circuit breakers
     */
    async shutdown(): Promise<void> {
        if (this.healthCheckTimer) {
            clearInterval(this.healthCheckTimer);
        }
        
        const shutdownPromises: Promise<void>[] = [];
        for (const entry of this.registry.values()) {
            shutdownPromises.push(entry.circuitBreaker.shutdown());
        }
        
        await Promise.allSettled(shutdownPromises);
        this.registry.clear();
    }

    /**
     * Resolve configuration using patterns and defaults
     */
    private async resolveConfiguration(
        service: string,
        operation: string,
        userConfig?: Partial<CircuitBreakerConfig>,
    ): Promise<Partial<CircuitBreakerConfig>> {
        // Start with base defaults
        let config: Partial<CircuitBreakerConfig> = {
            failureThreshold: 5,
            timeoutMs: 10000,
            resetTimeoutMs: 30000,
            successThreshold: 2,
            monitoringWindowMs: 60000,
            healthCheckInterval: 5000,
        };
        
        // Apply service-specific defaults
        const serviceConfig = this.defaultConfigs.get(service);
        if (serviceConfig) {
            config = { ...config, ...serviceConfig };
        }
        
        // Apply pattern-based configuration
        const pattern = this.findMatchingPattern(service, operation);
        if (pattern) {
            config = { ...config, ...pattern.recommendedConfig };
        }
        
        // Apply user overrides
        if (userConfig) {
            config = { ...config, ...userConfig };
        }
        
        return config;
    }

    /**
     * Find matching pattern for service/operation
     */
    private findMatchingPattern(service: string, operation: string): CircuitBreakerPattern | undefined {
        return this.patterns.find(pattern => 
            pattern.serviceMatcher.test(service) && 
            pattern.operationMatcher.test(operation),
        );
    }

    /**
     * Classify error using error classifier
     */
    private async classifyError(
        error: unknown,
        service: string,
        operation: string,
    ): Promise<ErrorClassification> {
        const context: ErrorContext = {
            tier: 3,
            component: service,
            operation,
            attemptCount: 1,
            previousStrategies: [],
            systemState: {},
            resourceState: {},
            performanceMetrics: {},
            userContext: {
                requestId: `cb-${Date.now()}`,
            },
        };
        
        return this.errorClassifier.classify(error instanceof Error ? error : new Error(String(error)), context);
    }

    /**
     * Handle classified error with intelligent actions
     */
    private async handleClassifiedError(
        classification: ErrorClassification,
        service: string,
        operation: string,
        error: unknown,
    ): Promise<void> {
        // Critical errors might warrant immediate circuit opening
        if (classification.severity === ErrorSeverity.CRITICAL || 
            classification.severity === ErrorSeverity.FATAL) {
            
            const circuitBreaker = await this.getCircuitBreaker(service, operation);
            await circuitBreaker.forceState(
                CircuitState.OPEN,
                `Critical error: ${classification.severity}`,
                { classification, error },
            );
        }
        
        // Emit error classification event
        await this.eventBus.publish("resilience.error_classified", {
            service,
            operation,
            classification,
            error: error instanceof Error ? {
                type: error.constructor.name,
                message: error.message,
            } : { type: typeof error, message: String(error) },
            timestamp: new Date(),
            source: "circuit-breaker-manager",
        });
    }

    /**
     * Find entries matching pattern
     */
    private findMatchingEntries(pattern: {
        services?: string[];
        operations?: string[];
        states?: CircuitState[];
        tags?: string[];
    }): Map<string, CircuitBreakerEntry> {
        const matches = new Map<string, CircuitBreakerEntry>();
        
        for (const [key, entry] of this.registry) {
            const [service, operation] = key.split(":");
            const state = entry.circuitBreaker.getState();
            
            // Check service filter
            if (pattern.services && !pattern.services.includes(service)) {
                continue;
            }
            
            // Check operation filter
            if (pattern.operations && !pattern.operations.includes(operation)) {
                continue;
            }
            
            // Check state filter
            if (pattern.states && !pattern.states.includes(state.state)) {
                continue;
            }
            
            // Check tags filter
            if (pattern.tags && !pattern.tags.some(tag => entry.tags.includes(tag))) {
                continue;
            }
            
            matches.set(key, entry);
        }
        
        return matches;
    }

    /**
     * Generate tags for circuit breaker categorization
     */
    private generateTags(service: string, operation: string): string[] {
        const tags = ["circuit-breaker", `service:${service}`, `operation:${operation}`];
        
        // Add technology-specific tags
        if (service.includes("llm") || service.includes("ai")) {
            tags.push("ai-service");
        }
        if (service.includes("db") || service.includes("database")) {
            tags.push("database");
        }
        if (service.includes("api") || service.includes("http")) {
            tags.push("api");
        }
        
        return tags;
    }

    /**
     * Create consistent key for service/operation pair
     */
    private createKey(service: string, operation: string): string {
        return `${service}:${operation}`;
    }

    /**
     * Start health monitoring
     */
    private startHealthMonitoring(): void {
        this.healthCheckTimer = setInterval(() => {
            this.performHealthCheck().catch(error => {
                console.error("[CircuitBreakerManager] Health check failed:", error);
            });
        }, this.healthCheckInterval);
    }

    /**
     * Perform comprehensive health check
     */
    private async performHealthCheck(): Promise<void> {
        this.lastHealthCheck = new Date();
        this.healthSummary = this.getHealthSummary();
        
        // All telemetry now handled by event-driven agents
        // Health data available through getHealthSummary() and getMetrics()
    }

    /**
     * Update metrics based on operation result
     */
    private updateMetrics(success: boolean, service: string, operation: string): void {
        if (success) {
            this.metrics.recoveryRate++;
        } else {
            this.metrics.errorRate++;
        }
        
        this.metrics.timestamp = new Date();
        
        // Update service-specific breakdown
        const key = `${service}:${operation}`;
        if (!this.metrics.breakdown[key]) {
            this.metrics.breakdown[key] = 0;
        }
        this.metrics.breakdown[key]++;
    }

    /**
     * Generate health recommendations
     */
    private generateHealthRecommendations(summary: HealthSummary): void {
        summary.recommendations = [];
        
        if (summary.unhealthy > 0) {
            summary.recommendations.push(
                `${summary.unhealthy} circuit breakers are open - investigate failing services`,
            );
        }
        
        if (summary.degraded > summary.total * 0.2) {
            summary.recommendations.push(
                "High number of degraded circuit breakers - consider system health review",
            );
        }
        
        // Service-specific recommendations
        for (const [service, stats] of Object.entries(summary.byService)) {
            if (stats.unhealthy > stats.healthy) {
                summary.recommendations.push(
                    `Service ${service} has more failing than healthy operations`,
                );
            }
        }
    }

    /**
     * Get top failing services
     */
    private getTopFailingServices(): Array<{ service: string; failureRate: number }> {
        const serviceCounts = new Map<string, { total: number; failures: number }>();
        
        for (const [key, entry] of this.registry) {
            const [service] = key.split(":");
            const state = entry.circuitBreaker.getState();
            
            if (!serviceCounts.has(service)) {
                serviceCounts.set(service, { total: 0, failures: 0 });
            }
            
            const counts = serviceCounts.get(service)!;
            counts.total++;
            
            if (state.state === CircuitState.OPEN) {
                counts.failures++;
            }
        }
        
        return Array.from(serviceCounts.entries())
            .map(([service, counts]) => ({
                service,
                failureRate: counts.total > 0 ? counts.failures / counts.total : 0,
            }))
            .filter(item => item.failureRate > 0)
            .sort((a, b) => b.failureRate - a.failureRate)
            .slice(0, 10); // Top 10
    }

    /**
     * Load default patterns for common scenarios
     */
    private loadDefaultPatterns(): void {
        this.addPattern({
            id: "llm-api-pattern",
            name: "LLM API Circuit Breaker",
            description: "Circuit breaker for LLM API calls with rate limiting awareness",
            serviceMatcher: /.*llm.*|.*ai.*|.*openai.*|.*anthropic.*/i,
            operationMatcher: /.*/,
            errorTypes: ["rate_limited", "timeout", "service_unavailable"],
            recommendedConfig: {
                failureThreshold: 3,
                timeoutMs: 30000,
                resetTimeoutMs: 60000,
                successThreshold: 2,
                monitoringWindowMs: 120000,
                healthCheckInterval: 10000,
                degradationMode: "USE_FALLBACK" as any,
                errorThresholds: [],
            },
            successRate: 0.85,
            usageCount: 0,
            confidence: 0.9,
        });

        this.addPattern({
            id: "database-pattern",
            name: "Database Circuit Breaker",
            description: "Circuit breaker for database operations",
            serviceMatcher: /.*db.*|.*database.*|.*postgres.*|.*redis.*/i,
            operationMatcher: /.*/,
            errorTypes: ["connection_error", "timeout", "deadlock"],
            recommendedConfig: {
                failureThreshold: 5,
                timeoutMs: 5000,
                resetTimeoutMs: 15000,
                successThreshold: 3,
                monitoringWindowMs: 30000,
                healthCheckInterval: 5000,
                degradationMode: "FAIL_FAST" as any,
                errorThresholds: [],
            },
            successRate: 0.95,
            usageCount: 0,
            confidence: 0.95,
        });
    }

    /**
     * Initialize health summary
     */
    private initializeHealthSummary(): HealthSummary {
        return {
            total: 0,
            healthy: 0,
            degraded: 0,
            unhealthy: 0,
            byService: {},
            alerting: [],
            recommendations: [],
        };
    }

    /**
     * Initialize metrics
     */
    private initializeMetrics(): ResilienceMetrics {
        return {
            errorRate: 0,
            recoveryRate: 0,
            meanTimeToRecovery: 0,
            circuitBreakerTrips: 0,
            fallbackUsage: 0,
            userImpactScore: 0,
            systemReliability: 1.0,
            adaptationEffectiveness: 0,
            timestamp: new Date(),
            breakdown: {},
        };
    }
}