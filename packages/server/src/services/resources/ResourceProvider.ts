import { EventEmitter } from "events";
import { logger } from "../../events/logger.js";
import { HTTPClient, type AuthConfig } from "../http/httpClient.js";
import { CircuitBreaker, CircuitBreakerFactory } from "./circuitBreaker.js";
import type { GetConfig, GetResourceMetadata, ResourceId } from "./typeRegistry.js";
import type {
    HealthCheckResult,
    IResource,
    ResourceCategory,
    ResourceInitOptions,
    TypedInternalResourceInfo,
    TypedPublicResourceInfo
} from "./types.js";
import {
    DiscoveryStatus,
    ResourceEvent,
    ResourceHealth,
    type DeploymentType
} from "./types.js";

// Constants
const DEFAULT_FETCH_TIMEOUT_MS = 5000;

/**
 * Abstract base class for all resource providers with full type safety
 * Handles common functionality like health checks, discovery, and lifecycle
 */
export abstract class ResourceProvider<
    TId extends ResourceId = ResourceId,
    TConfig extends GetConfig<TId> = GetConfig<TId>
> extends EventEmitter
    implements IResource {

    protected config?: TConfig;
    protected _health: ResourceHealth = ResourceHealth.Unknown;
    protected _status: DiscoveryStatus = DiscoveryStatus.NotFound;
    protected _lastHealthCheck?: Date;
    protected healthCheckInterval?: NodeJS.Timeout;
    protected isInitialized = false;
    protected httpClient?: HTTPClient;
    
    // Circuit breakers for resilience
    protected discoveryCircuitBreaker?: CircuitBreaker;
    protected healthCheckCircuitBreaker?: CircuitBreaker;
    
    // Error tracking for health monitoring
    private consecutiveHealthFailures = 0;
    private maxConsecutiveFailures = 5;

    constructor() {
        super();
    }

    // Abstract properties that must be implemented
    abstract readonly id: string;
    abstract readonly category: ResourceCategory;
    abstract readonly displayName: string;
    abstract readonly description: string;
    abstract readonly isSupported: boolean;
    abstract readonly deploymentType: DeploymentType;

    // Abstract methods that must be implemented
    protected abstract performHealthCheck(): Promise<HealthCheckResult>;
    protected abstract performDiscovery(): Promise<boolean>;

    /**
     * Initialize the resource with configuration
     */
    async initialize(options: ResourceInitOptions<TId>): Promise<void> {
        if (this.isInitialized) {
            logger.warn(`[${this.id}] Already initialized, skipping`);
            return;
        }

        this.config = options.config as TConfig;

        // Initialize HTTP client with resource profile
        this.httpClient = HTTPClient.forResources();
        
        // Initialize circuit breakers for resilience
        this.discoveryCircuitBreaker = CircuitBreakerFactory.forResourceDiscovery(this.id);
        this.healthCheckCircuitBreaker = CircuitBreakerFactory.forHealthCheck(this.id);

        // Set initial status based on enabled state
        if (!this.config?.enabled) {
            this._status = DiscoveryStatus.NotFound;
            logger.info(`[${this.id}] Resource disabled in configuration`);
            return;
        }

        try {
            // Perform initial discovery
            const discovered = await this.discover();

            if (discovered) {
                // Start health monitoring if configured
                this.startHealthMonitoring();

                // Perform initial health check
                await this.healthCheck();
            }

            this.isInitialized = true;
            logger.info(`[${this.id}] Initialized successfully, status: ${this._status}`);
        } catch (error) {
            logger.error(`[${this.id}] Failed to initialize`, error);
            this._status = DiscoveryStatus.NotFound;
            this._health = ResourceHealth.Unhealthy;
            throw error;
        }
    }

    /**
     * Discover if this resource is running locally
     */
    async discover(): Promise<boolean> {
        if (!this.isSupported) {
            this._status = DiscoveryStatus.NotSupported;
            return false;
        }

        if (!this.discoveryCircuitBreaker?.isCallAllowed()) {
            logger.debug(`[${this.id}] Discovery blocked by circuit breaker`);
            // Don't change status if circuit breaker is blocking
            return this._status === DiscoveryStatus.Available;
        }

        const previousStatus = this._status;
        this._status = DiscoveryStatus.Discovering;

        try {
            // Use circuit breaker to protect against repeated failures
            const found = await this.discoveryCircuitBreaker!.execute(async () => {
                return await this.performDiscovery();
            });

            const wasAvailable = previousStatus === DiscoveryStatus.Available;
            this._status = found ? DiscoveryStatus.Available : DiscoveryStatus.NotFound;

            // Emit events based on status change
            if (!wasAvailable && found) {
                this.emit(ResourceEvent.Discovered, {
                    resourceId: this.id,
                    category: this.category,
                    timestamp: new Date(),
                });
                logger.info(`[${this.id}] Resource discovered and available`);
            } else if (wasAvailable && !found) {
                this.emit(ResourceEvent.Lost, {
                    resourceId: this.id,
                    category: this.category,
                    timestamp: new Date(),
                });
                logger.warn(`[${this.id}] Resource lost and no longer available`);
            }

            return found;
        } catch (error) {
            logger.error(`[${this.id}] Discovery failed`, error);
            this._status = DiscoveryStatus.NotFound;
            
            // If this was a circuit breaker error, don't mark as permanently failed
            if (error instanceof Error && error.message.includes("Circuit breaker is OPEN")) {
                logger.debug(`[${this.id}] Discovery temporarily blocked by circuit breaker`);
            }
            
            return false;
        }
    }

    /**
     * Check if the resource is healthy and available
     */
    async healthCheck(): Promise<HealthCheckResult> {
        if (this._status !== DiscoveryStatus.Available) {
            return {
                healthy: false,
                message: "Resource not available for health check",
                timestamp: new Date(),
            };
        }

        if (!this.healthCheckCircuitBreaker?.isCallAllowed()) {
            logger.debug(`[${this.id}] Health check blocked by circuit breaker`);
            return {
                healthy: false,
                message: "Health check temporarily blocked by circuit breaker",
                timestamp: new Date(),
            };
        }

        const previousHealth = this._health;
        this._health = ResourceHealth.Checking;

        try {
            // Use circuit breaker to protect against repeated health check failures
            const result = await this.healthCheckCircuitBreaker!.execute(async () => {
                return await this.performHealthCheck();
            });

            this._health = result.healthy ? ResourceHealth.Healthy : ResourceHealth.Unhealthy;
            this._lastHealthCheck = new Date();

            // Track consecutive failures
            if (result.healthy) {
                this.consecutiveHealthFailures = 0;
            } else {
                this.consecutiveHealthFailures++;
                
                // Stop health monitoring if too many consecutive failures
                if (this.consecutiveHealthFailures >= this.maxConsecutiveFailures) {
                    logger.warn(`[${this.id}] Too many consecutive health failures (${this.consecutiveHealthFailures}), stopping health monitoring temporarily`);
                    this.stopHealthMonitoring();
                    
                    // Mark resource as lost if it was previously available
                    if (this._status === DiscoveryStatus.Available) {
                        this._status = DiscoveryStatus.NotFound;
                        this.emit(ResourceEvent.Lost, {
                            resourceId: this.id,
                            category: this.category,
                            timestamp: new Date(),
                            details: { reason: "Too many consecutive health failures" },
                        });
                    }
                }
            }

            // Emit event if health changed
            if (previousHealth !== this._health) {
                this.emit(ResourceEvent.HealthChanged, {
                    resourceId: this.id,
                    category: this.category,
                    previousHealth,
                    currentHealth: this._health,
                    timestamp: new Date(),
                    details: { 
                        consecutiveFailures: this.consecutiveHealthFailures,
                        circuitBreakerState: this.healthCheckCircuitBreaker?.getStats().state,
                    },
                });
            }

            return result;
        } catch (error) {
            this.consecutiveHealthFailures++;
            logger.error(`[${this.id}] Health check failed (${this.consecutiveHealthFailures} consecutive failures)`, error);
            this._health = ResourceHealth.Unhealthy;

            // Handle circuit breaker errors more gracefully
            const isCircuitBreakerError = error instanceof Error && error.message.includes("Circuit breaker is OPEN");
            
            return {
                healthy: false,
                message: isCircuitBreakerError 
                    ? "Health check blocked by circuit breaker due to repeated failures"
                    : error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
                details: {
                    consecutiveFailures: this.consecutiveHealthFailures,
                    circuitBreakerBlocked: isCircuitBreakerError,
                },
            };
        }
    }

    /**
     * Get public resource information (safe to expose externally)
     */
    getPublicInfo(): TypedPublicResourceInfo<TId> {
        return {
            id: this.id,
            category: this.category,
            displayName: this.displayName,
            description: this.description,
            status: this._status,
            health: this._health,
            lastHealthCheck: this._lastHealthCheck,
            metadata: this.getMetadata(),
        };
    }

    /**
     * Get internal resource information including sensitive configuration
     * @internal WARNING: Contains sensitive data including API keys, passwords, and tokens.
     * Only use this method internally for resource operations that require configuration access.
     */
    getInternalInfo(): TypedInternalResourceInfo<TId> {
        return {
            ...this.getPublicInfo(),
            config: this.config,
        };
    }

    /**
     * Override to provide resource-specific metadata
     * Subclasses should return properly typed metadata for their category
     */
    protected getMetadata(): GetResourceMetadata<TId> {
        // Force subclasses to implement properly typed metadata
        throw new Error(`getMetadata must be implemented by ${this.constructor.name}`);
    }

    /**
     * Start periodic health monitoring
     */
    protected startHealthMonitoring(): void {
        if (!this.config?.healthCheck?.intervalMs) {
            return;
        }

        // Clear any existing interval
        this.stopHealthMonitoring();

        // Start new interval with improved error handling
        this.healthCheckInterval = setInterval(async () => {
            try {
                await this.healthCheck();
            } catch (error) {
                logger.error(`[${this.id}] Periodic health check failed`, error);
                
                // If health monitoring should be stopped due to consecutive failures,
                // the healthCheck method itself will handle that
            }
        }, this.config.healthCheck.intervalMs);

        logger.debug(`[${this.id}] Started health monitoring, interval: ${this.config.healthCheck.intervalMs}ms`);
    }

    /**
     * Restart health monitoring after a failure period
     * This method can be called externally to resume monitoring
     */
    public restartHealthMonitoring(): void {
        if (this._status !== DiscoveryStatus.Available) {
            logger.debug(`[${this.id}] Cannot restart health monitoring - resource not available`);
            return;
        }

        logger.info(`[${this.id}] Restarting health monitoring after failure period`);
        this.consecutiveHealthFailures = 0;
        
        // Reset circuit breakers
        this.healthCheckCircuitBreaker?.forceReset();
        this.discoveryCircuitBreaker?.forceReset();
        
        this.startHealthMonitoring();
    }

    /**
     * Stop health monitoring
     */
    protected stopHealthMonitoring(): void {
        if (this.healthCheckInterval) {
            clearInterval(this.healthCheckInterval);
            this.healthCheckInterval = undefined;
            logger.debug(`[${this.id}] Stopped health monitoring`);
        }
    }

    /**
     * Cleanup and shutdown the resource
     */
    async shutdown(): Promise<void> {
        logger.info(`[${this.id}] Shutting down resource`);

        // Stop health monitoring
        this.stopHealthMonitoring();

        // Reset circuit breakers to prevent any pending operations
        this.discoveryCircuitBreaker?.forceReset();
        this.healthCheckCircuitBreaker?.forceReset();

        // Clean up HTTP client resources (though HTTPClient doesn't need explicit cleanup,
        // this is good practice for future extensions)
        this.httpClient = undefined;

        // Reset error tracking
        this.consecutiveHealthFailures = 0;

        // Reset state
        this._status = DiscoveryStatus.NotFound;
        this._health = ResourceHealth.Unknown;
        this.isInitialized = false;

        // Remove all event listeners
        this.removeAllListeners();

        logger.debug(`[${this.id}] Resource shutdown completed`);
    }


    /**
     * Get authentication configuration from resource config
     * Override this method in specific resources for custom auth handling
     */
    protected getAuthConfig(): AuthConfig | undefined {
        if (!this.config) return undefined;

        const anyConfig = this.config as any;

        // Check for API key
        if (anyConfig.apiKey) {
            return {
                type: "apikey",
                token: anyConfig.apiKey,
                headerName: anyConfig.apiKeyHeader || "X-API-Key",
            };
        }

        // Check for bearer token
        if (anyConfig.bearerToken) {
            return {
                type: "bearer",
                token: anyConfig.bearerToken,
            };
        }

        // Check for basic auth
        if (anyConfig.username && anyConfig.password) {
            return {
                type: "basic",
                username: anyConfig.username,
                password: anyConfig.password,
            };
        }

        return undefined;
    }
}

