/**
 * Adaptive Circuit Breaker Implementation
 * 
 * Provides intelligent circuit breaking with adaptive threshold adjustment,
 * monitoring integration, and support for emergent resilience learning.
 * 
 * Key Features:
 * - Three-state machine (CLOSED → OPEN → HALF_OPEN)
 * - Adaptive threshold adjustment based on monitoring data
 * - Integration with telemetry and event systems
 * - Support for fallback strategies and degradation modes
 * - Event emission for agent learning
 * - Performance-aware operation tracking
 */

import { TelemetryShimAdapter as TelemetryShim } from "../../monitoring/adapters/TelemetryShimAdapter.js";
import { EventBus } from "../events/eventBus.js";
import {
    CircuitState,
    CircuitBreakerConfig,
    CircuitBreakerState,
    FallbackStrategyConfig,
    DegradationMode,
    ResilienceEventType,
    ErrorSeverity,
    MonitoringEventPrefix,
    ErrorCategory,
    ErrorRecoverability,
} from "@vrooli/shared";

/**
 * Circuit breaker configuration with defaults
 */
const DEFAULT_CONFIG: CircuitBreakerConfig = {
    failureThreshold: 5,
    timeoutMs: 10000,
    resetTimeoutMs: 30000,
    successThreshold: 2,
    monitoringWindowMs: 60000,
    healthCheckInterval: 5000,
    degradationMode: DegradationMode.FAIL_FAST,
    errorThresholds: [],
};

/**
 * Circuit breaker performance metrics
 */
interface CircuitBreakerMetrics {
    requestCount: number;
    successCount: number;
    failureCount: number;
    rejectionCount: number;
    averageResponseTime: number;
    lastRequestTime: Date;
    lastFailureTime?: Date;
    lastSuccessTime?: Date;
    totalTimeouts: number;
    circuitOpenTime: number;
    fallbackUsageCount: number;
}

/**
 * Adaptive threshold state
 */
interface AdaptiveThresholds {
    baseFailureThreshold: number;
    currentFailureThreshold: number;
    adjustmentFactor: number;
    lastAdjustmentTime: Date;
    learningRate: number;
    stabilityWindow: number;
}

/**
 * Operation execution context
 */
interface ExecutionContext {
    operationId: string;
    timeout: number;
    metadata: Record<string, unknown>;
    startTime: Date;
    retryAttempt?: number;
    parentRequestId?: string;
}

/**
 * Adaptive Circuit Breaker
 * 
 * Implements intelligent circuit breaking with learning capabilities:
 * - Monitors success/failure patterns
 * - Adjusts thresholds based on historical performance
 * - Integrates with monitoring and event systems
 * - Supports multiple fallback strategies
 * - Provides rich telemetry for agent learning
 */
export class AdaptiveCircuitBreaker {
    private readonly service: string;
    private readonly operation: string;
    private readonly config: CircuitBreakerConfig;
    private readonly telemetry: TelemetryShim;
    private readonly eventBus: EventBus;
    
    // State management
    private state: CircuitState = CircuitState.CLOSED;
    private metrics: CircuitBreakerMetrics;
    private adaptiveThresholds: AdaptiveThresholds;
    private stateChangeTime: Date = new Date();
    private nextRetryTime?: Date;
    
    // Internal tracking
    private requestBuffer: Array<{ timestamp: Date; success: boolean; duration: number }> = [];
    private halfOpenRequestCount = 0;
    private stateHistory: Array<{ state: CircuitState; timestamp: Date; reason: string }> = [];
    
    // Timers
    private healthCheckTimer?: NodeJS.Timeout;
    private resetTimer?: NodeJS.Timeout;

    constructor(
        service: string,
        operation: string,
        config: Partial<CircuitBreakerConfig>,
        telemetry: TelemetryShim,
        eventBus: EventBus,
    ) {
        this.service = service;
        this.operation = operation;
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.telemetry = telemetry;
        this.eventBus = eventBus;
        
        this.metrics = this.initializeMetrics();
        this.adaptiveThresholds = this.initializeAdaptiveThresholds();
        
        this.startHealthCheck();
    }

    /**
     * Execute operation with circuit breaker protection
     */
    async execute<T>(
        operation: () => Promise<T>,
        context?: Partial<ExecutionContext>,
    ): Promise<T> {
        const executionContext: ExecutionContext = {
            operationId: `${this.service}:${this.operation}:${Date.now()}`,
            timeout: this.config.timeoutMs,
            metadata: {},
            startTime: new Date(),
            ...context,
        };

        const canExecute = await this.canExecute(executionContext);
        
        if (!canExecute) {
            await this.handleRejection(executionContext);
            throw new CircuitBreakerOpenError(
                `Circuit breaker OPEN for ${this.service}:${this.operation}`,
                this.state,
                this.getStateInfo(),
            );
        }

        return this.executeWithMonitoring(operation, executionContext);
    }

    /**
     * Force circuit state change
     */
    async forceState(
        newState: CircuitState,
        reason: string,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        const oldState = this.state;
        await this.changeState(newState, reason);
        
        await this.telemetry.emitError(
            new Error(`Circuit breaker force state: ${oldState} → ${newState}`),
            `${this.service}:${this.operation}`,
            "medium",
            { reason, forced: true, ...metadata },
        );
    }

    /**
     * Get current circuit breaker state
     */
    getState(): CircuitBreakerState {
        return {
            state: this.state,
            failureCount: this.metrics.failureCount,
            successCount: this.metrics.successCount,
            lastFailureTime: this.metrics.lastFailureTime,
            lastSuccessTime: this.metrics.lastSuccessTime,
            stateChangeTime: this.stateChangeTime,
            nextRetryTime: this.nextRetryTime,
            metadata: {
                service: this.service,
                operation: this.operation,
                adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
                averageResponseTime: this.metrics.averageResponseTime,
                requestCount: this.metrics.requestCount,
                rejectionCount: this.metrics.rejectionCount,
            },
        };
    }

    /**
     * Get performance metrics
     */
    getMetrics(): CircuitBreakerMetrics & {
        adaptiveThresholds: AdaptiveThresholds;
        stateHistory: typeof this.stateHistory;
    } {
        return {
            ...this.metrics,
            adaptiveThresholds: { ...this.adaptiveThresholds },
            stateHistory: [...this.stateHistory],
        };
    }

    /**
     * Update configuration at runtime
     */
    updateConfig(updates: Partial<CircuitBreakerConfig>): void {
        Object.assign(this.config, updates);
        
        // Adjust adaptive thresholds if base threshold changed
        if (updates.failureThreshold) {
            this.adaptiveThresholds.baseFailureThreshold = updates.failureThreshold;
            this.recalculateThreshold();
        }
    }

    /**
     * Shutdown circuit breaker
     */
    async shutdown(): Promise<void> {
        if (this.healthCheckTimer) {
            clearTimeout(this.healthCheckTimer);
        }
        if (this.resetTimer) {
            clearTimeout(this.resetTimer);
        }
        
        await this.emitCircuitBreakerEvent(ResilienceEventType.CIRCUIT_BREAKER_CLOSED, {
            reason: "shutdown",
            finalMetrics: this.metrics,
        });
    }

    /**
     * Check if operation can execute based on current state
     */
    private async canExecute(context: ExecutionContext): Promise<boolean> {
        this.cleanupRequestBuffer();
        
        switch (this.state) {
            case CircuitState.CLOSED:
                return true;
                
            case CircuitState.OPEN:
                if (this.shouldAttemptReset()) {
                    await this.changeState(CircuitState.HALF_OPEN, "reset timeout elapsed");
                    return true;
                }
                return this.shouldUseFallback();
                
            case CircuitState.HALF_OPEN:
                if (this.halfOpenRequestCount < this.config.successThreshold) {
                    return true;
                }
                return false;
                
            default:
                return false;
        }
    }

    /**
     * Execute operation with comprehensive monitoring
     */
    private async executeWithMonitoring<T>(
        operation: () => Promise<T>,
        context: ExecutionContext,
    ): Promise<T> {
        const startTime = Date.now();
        let success = false;
        let error: Error | undefined;
        let result: T;

        try {
            // Apply timeout
            const timeoutPromise = new Promise<never>((_, reject) => {
                setTimeout(() => {
                    reject(new Error(`Operation timeout after ${context.timeout}ms`));
                }, context.timeout);
            });

            result = await Promise.race([operation(), timeoutPromise]);
            success = true;
            
            await this.recordSuccess(context, Date.now() - startTime);
            return result;
            
        } catch (err) {
            error = err instanceof Error ? err : new Error(String(err));
            await this.recordFailure(context, error, Date.now() - startTime);
            throw error;
            
        } finally {
            // Update metrics
            this.updateMetrics(success, Date.now() - startTime);
            
            // Emit telemetry
            await this.telemetry.emitExecutionTiming(
                `${this.service}:${this.operation}`,
                context.operationId,
                context.startTime,
                new Date(),
                success,
                {
                    circuitState: this.state,
                    adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
                    error: error?.message,
                    retryAttempt: context.retryAttempt,
                },
            );
        }
    }

    /**
     * Record successful operation
     */
    private async recordSuccess(context: ExecutionContext, duration: number): Promise<void> {
        this.requestBuffer.push({ timestamp: new Date(), success: true, duration });
        this.metrics.successCount++;
        this.metrics.lastSuccessTime = new Date();
        
        if (this.state === CircuitState.HALF_OPEN) {
            this.halfOpenRequestCount++;
            
            if (this.halfOpenRequestCount >= this.config.successThreshold) {
                await this.changeState(CircuitState.CLOSED, "success threshold reached");
            }
        }
        
        // Adaptive learning: success might allow threshold relaxation
        this.adaptThresholdOnSuccess();
        
        await this.emitCircuitBreakerEvent(ResilienceEventType.RECOVERY_COMPLETED, {
            context,
            duration,
            adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
        });
    }

    /**
     * Record failed operation
     */
    private async recordFailure(
        context: ExecutionContext,
        error: Error,
        duration: number,
    ): Promise<void> {
        this.requestBuffer.push({ timestamp: new Date(), success: false, duration });
        this.metrics.failureCount++;
        this.metrics.lastFailureTime = new Date();
        
        // Check if we should open the circuit
        const shouldOpen = this.shouldOpenCircuit();
        
        if (this.state === CircuitState.HALF_OPEN || shouldOpen) {
            await this.changeState(CircuitState.OPEN, `failure threshold exceeded: ${error.message}`);
        }
        
        // Adaptive learning: failure might require threshold tightening
        this.adaptThresholdOnFailure();
        
        await this.emitCircuitBreakerEvent(ResilienceEventType.RECOVERY_FAILED, {
            context,
            error: {
                type: error.constructor.name,
                message: error.message,
                stack: error.stack,
            },
            duration,
            adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
        });
    }

    /**
     * Handle request rejection
     */
    private async handleRejection(context: ExecutionContext): Promise<void> {
        this.metrics.rejectionCount++;
        
        // Try fallback if configured
        if (this.config.fallbackStrategy && this.config.degradationMode === DegradationMode.USE_FALLBACK) {
            await this.emitCircuitBreakerEvent(ResilienceEventType.FALLBACK_TRIGGERED, {
                context,
                fallbackStrategy: this.config.fallbackStrategy,
                degradationMode: this.config.degradationMode,
            });
        }
    }

    /**
     * Change circuit breaker state
     */
    private async changeState(newState: CircuitState, reason: string): Promise<void> {
        const oldState = this.state;
        this.state = newState;
        this.stateChangeTime = new Date();
        
        // Record state change in history
        this.stateHistory.push({
            state: newState,
            timestamp: this.stateChangeTime,
            reason,
        });
        
        // Keep history size manageable
        if (this.stateHistory.length > 100) {
            this.stateHistory.splice(0, 10);
        }
        
        // Reset counters for new state
        if (newState === CircuitState.HALF_OPEN) {
            this.halfOpenRequestCount = 0;
        }
        
        // Set up reset timer for OPEN state
        if (newState === CircuitState.OPEN) {
            this.nextRetryTime = new Date(Date.now() + this.config.resetTimeoutMs);
            this.metrics.circuitOpenTime += Date.now() - this.stateChangeTime.getTime();
        }
        
        // Emit state change event
        const eventType = newState === CircuitState.OPEN 
            ? ResilienceEventType.CIRCUIT_BREAKER_OPENED
            : ResilienceEventType.CIRCUIT_BREAKER_CLOSED;
            
        await this.emitCircuitBreakerEvent(eventType, {
            oldState,
            newState,
            reason,
            metrics: this.metrics,
        });
        
        // Emit telemetry
        await this.telemetry.emitComponentHealth(
            `${this.service}:${this.operation}`,
            newState === CircuitState.OPEN ? "unhealthy" : 
            newState === CircuitState.HALF_OPEN ? "degraded" : "healthy",
            [{
                name: "circuit_breaker",
                status: newState === CircuitState.OPEN ? "fail" : 
                       newState === CircuitState.HALF_OPEN ? "warn" : "pass",
                message: reason,
                duration: Date.now() - this.stateChangeTime.getTime(),
            }],
        );
    }

    /**
     * Determine if circuit should open based on current failure rate
     */
    private shouldOpenCircuit(): boolean {
        const windowStart = Date.now() - this.config.monitoringWindowMs;
        const recentRequests = this.requestBuffer.filter(
            req => req.timestamp.getTime() >= windowStart,
        );
        
        if (recentRequests.length === 0) {
            return false;
        }
        
        const failures = recentRequests.filter(req => !req.success).length;
        return failures >= this.adaptiveThresholds.currentFailureThreshold;
    }

    /**
     * Check if circuit should attempt reset from OPEN to HALF_OPEN
     */
    private shouldAttemptReset(): boolean {
        if (!this.nextRetryTime) {
            return true;
        }
        return Date.now() >= this.nextRetryTime.getTime();
    }

    /**
     * Check if fallback should be used in OPEN state
     */
    private shouldUseFallback(): boolean {
        return this.config.degradationMode === DegradationMode.USE_FALLBACK ||
               this.config.degradationMode === DegradationMode.PARTIAL_SERVICE;
    }

    /**
     * Adapt threshold based on successful operations
     */
    private adaptThresholdOnSuccess(): void {
        const now = Date.now();
        const timeSinceLastAdjustment = now - this.adaptiveThresholds.lastAdjustmentTime.getTime();
        
        // Only adjust if stability window has passed
        if (timeSinceLastAdjustment < this.adaptiveThresholds.stabilityWindow) {
            return;
        }
        
        // Gradually increase threshold if we're consistently successful
        const successRate = this.calculateRecentSuccessRate();
        if (successRate > 0.9) { // 90% success rate
            const increase = Math.ceil(this.adaptiveThresholds.currentFailureThreshold * 0.1);
            this.adaptiveThresholds.currentFailureThreshold = Math.min(
                this.adaptiveThresholds.currentFailureThreshold + increase,
                this.adaptiveThresholds.baseFailureThreshold * 2, // Cap at 2x base
            );
            
            this.adaptiveThresholds.lastAdjustmentTime = new Date();
        }
    }

    /**
     * Adapt threshold based on failed operations
     */
    private adaptThresholdOnFailure(): void {
        // Tighten threshold more aggressively on failures
        const decrease = Math.ceil(this.adaptiveThresholds.currentFailureThreshold * 0.2);
        this.adaptiveThresholds.currentFailureThreshold = Math.max(
            this.adaptiveThresholds.currentFailureThreshold - decrease,
            Math.ceil(this.adaptiveThresholds.baseFailureThreshold * 0.5), // Min at 50% of base
        );
        
        this.adaptiveThresholds.lastAdjustmentTime = new Date();
    }

    /**
     * Recalculate threshold based on recent performance
     */
    private recalculateThreshold(): void {
        const successRate = this.calculateRecentSuccessRate();
        
        if (successRate > 0.95) {
            // Very high success - can be more lenient
            this.adaptiveThresholds.currentFailureThreshold = 
                Math.ceil(this.adaptiveThresholds.baseFailureThreshold * 1.5);
        } else if (successRate < 0.8) {
            // Low success - be more strict
            this.adaptiveThresholds.currentFailureThreshold = 
                Math.ceil(this.adaptiveThresholds.baseFailureThreshold * 0.7);
        } else {
            // Normal range - use base threshold
            this.adaptiveThresholds.currentFailureThreshold = 
                this.adaptiveThresholds.baseFailureThreshold;
        }
    }

    /**
     * Calculate recent success rate
     */
    private calculateRecentSuccessRate(): number {
        const windowStart = Date.now() - this.config.monitoringWindowMs;
        const recentRequests = this.requestBuffer.filter(
            req => req.timestamp.getTime() >= windowStart,
        );
        
        if (recentRequests.length === 0) {
            return 1.0; // No recent requests, assume healthy
        }
        
        const successes = recentRequests.filter(req => req.success).length;
        return successes / recentRequests.length;
    }

    /**
     * Clean up old entries from request buffer
     */
    private cleanupRequestBuffer(): void {
        const cutoff = Date.now() - this.config.monitoringWindowMs;
        this.requestBuffer = this.requestBuffer.filter(
            req => req.timestamp.getTime() >= cutoff,
        );
    }

    /**
     * Update internal metrics
     */
    private updateMetrics(success: boolean, duration: number): void {
        this.metrics.requestCount++;
        this.metrics.lastRequestTime = new Date();
        
        // Update average response time with exponential moving average
        const alpha = 0.1; // Smoothing factor
        this.metrics.averageResponseTime = 
            this.metrics.averageResponseTime * (1 - alpha) + duration * alpha;
    }

    /**
     * Start health check timer
     */
    private startHealthCheck(): void {
        if (this.config.healthCheckInterval > 0) {
            this.healthCheckTimer = setInterval(() => {
                this.performHealthCheck().catch(error => {
                    console.error(`[CircuitBreaker] Health check failed: ${error.message}`);
                });
            }, this.config.healthCheckInterval);
        }
    }

    /**
     * Perform health check and emit metrics
     */
    private async performHealthCheck(): Promise<void> {
        const healthStatus = this.state === CircuitState.OPEN ? "unhealthy" :
                           this.state === CircuitState.HALF_OPEN ? "degraded" : "healthy";
        
        await this.telemetry.emitComponentHealth(
            `${this.service}:${this.operation}`,
            healthStatus,
            [{
                name: "circuit_breaker_health",
                status: healthStatus === "healthy" ? "pass" : 
                       healthStatus === "degraded" ? "warn" : "fail",
                message: `State: ${this.state}, Failures: ${this.metrics.failureCount}`,
            }],
        );
        
        // Emit resource utilization
        await this.telemetry.emitResourceUtilization(
            `${this.service}:${this.operation}`,
            {
                requests: this.metrics.requestCount,
                failures: this.metrics.failureCount,
                rejections: this.metrics.rejectionCount,
                avgResponseTime: this.metrics.averageResponseTime,
            } as any,
        );
    }

    /**
     * Emit circuit breaker event for agent learning
     */
    private async emitCircuitBreakerEvent(
        type: ResilienceEventType,
        metadata: Record<string, unknown>,
    ): Promise<void> {
        const event = {
            id: `cb-${this.service}-${this.operation}-${Date.now()}`,
            timestamp: new Date(),
            type,
            severity: this.state === CircuitState.OPEN ? ErrorSeverity.ERROR : ErrorSeverity.INFO,
            source: {
                tier: 3 as const,
                component: "circuit-breaker",
                operation: `${this.service}:${this.operation}`,
                requestId: `cb-${Date.now()}`,
            },
            classification: {
                severity: this.state === CircuitState.OPEN ? ErrorSeverity.ERROR : ErrorSeverity.INFO,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                systemFunctional: this.state !== CircuitState.OPEN,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 0.9,
                timestamp: new Date(),
                metadata: {
                    circuitState: this.state,
                    adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
                },
            },
            context: {
                tier: 3 as const,
                component: "circuit-breaker",
                operation: `${this.service}:${this.operation}`,
                attemptCount: 1,
                previousStrategies: [],
                systemState: { circuitState: this.state },
                resourceState: { metrics: this.metrics },
                performanceMetrics: {
                    averageResponseTime: this.metrics.averageResponseTime,
                    successRate: this.metrics.successCount / (this.metrics.requestCount || 1),
                },
            },
            strategy: {
                strategyType: "CIRCUIT_BREAK" as any,
                maxAttempts: 1,
                backoffStrategy: {
                    type: "FIXED" as any,
                    initialDelayMs: this.config.resetTimeoutMs,
                    maxDelayMs: this.config.resetTimeoutMs,
                    multiplier: 1,
                    jitterPercent: 0,
                    adaptiveAdjustment: true,
                },
                fallbackActions: [],
                priority: 1,
                timeoutMs: this.config.timeoutMs,
                conditions: [],
                estimatedSuccessRate: this.calculateRecentSuccessRate(),
                resourceRequirements: {},
            },
            outcome: {
                success: this.state !== CircuitState.OPEN,
                duration: Date.now() - this.stateChangeTime.getTime(),
                attemptCount: 1,
                strategiesUsed: ["CIRCUIT_BREAK" as any],
                qualityImpact: this.state === CircuitState.OPEN ? 1.0 : 0.0,
                resourceUsage: { requests: this.metrics.requestCount },
                userImpact: "MINIMAL" as any,
                lessons: [],
            },
            learningData: {
                similarity: 0.8,
                contextFeatures: [
                    `service:${this.service}`,
                    `operation:${this.operation}`,
                    `state:${this.state}`,
                ],
                effectiveStrategies: this.state === CircuitState.CLOSED ? ["CIRCUIT_BREAK" as any] : [],
                ineffectiveStrategies: this.state === CircuitState.OPEN ? ["CIRCUIT_BREAK" as any] : [],
                environmentalFactors: {
                    adaptiveThreshold: this.adaptiveThresholds.currentFailureThreshold,
                    successRate: this.calculateRecentSuccessRate(),
                },
                recommendations: [
                    this.state === CircuitState.OPEN 
                        ? "Consider fallback strategy or service health check"
                        : "Monitor success rate for threshold adjustment",
                ],
                confidence: 0.8,
            },
            ...metadata,
        };

        await this.eventBus.publish("resilience.circuit_breaker", event);
    }

    /**
     * Get comprehensive state information
     */
    private getStateInfo(): Record<string, unknown> {
        return {
            state: this.state,
            service: this.service,
            operation: this.operation,
            metrics: this.metrics,
            adaptiveThresholds: this.adaptiveThresholds,
            config: this.config,
            stateHistory: this.stateHistory.slice(-5), // Last 5 state changes
            nextRetryTime: this.nextRetryTime,
        };
    }

    /**
     * Initialize metrics with default values
     */
    private initializeMetrics(): CircuitBreakerMetrics {
        return {
            requestCount: 0,
            successCount: 0,
            failureCount: 0,
            rejectionCount: 0,
            averageResponseTime: 0,
            lastRequestTime: new Date(),
            totalTimeouts: 0,
            circuitOpenTime: 0,
            fallbackUsageCount: 0,
        };
    }

    /**
     * Initialize adaptive thresholds
     */
    private initializeAdaptiveThresholds(): AdaptiveThresholds {
        return {
            baseFailureThreshold: this.config.failureThreshold,
            currentFailureThreshold: this.config.failureThreshold,
            adjustmentFactor: 0.1,
            lastAdjustmentTime: new Date(),
            learningRate: 0.01,
            stabilityWindow: 30000, // 30 seconds
        };
    }
}

/**
 * Circuit breaker open error
 */
export class CircuitBreakerOpenError extends Error {
    constructor(
        message: string,
        public readonly state: CircuitState,
        public readonly stateInfo: Record<string, unknown>,
    ) {
        super(message);
        this.name = "CircuitBreakerOpenError";
    }
}

/**
 * Circuit breaker factory for creating instances with shared dependencies
 */
export class CircuitBreakerFactory {
    constructor(
        private readonly telemetry: TelemetryShim,
        private readonly eventBus: EventBus,
    ) {}

    create(
        service: string,
        operation: string,
        config: Partial<CircuitBreakerConfig> = {},
    ): AdaptiveCircuitBreaker {
        return new AdaptiveCircuitBreaker(
            service,
            operation,
            config,
            this.telemetry,
            this.eventBus,
        );
    }
}