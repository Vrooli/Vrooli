import { EventEmitter } from "events";
import { logger } from "../../events/logger.js";
import { HTTPClient, type AuthConfig } from "../http/httpClient.js";
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

        const previousStatus = this._status;
        this._status = DiscoveryStatus.Discovering;

        try {
            const found = await this.performDiscovery();

            const wasAvailable = previousStatus === DiscoveryStatus.Available;
            this._status = found ? DiscoveryStatus.Available : DiscoveryStatus.NotFound;

            // Emit events based on status change
            if (!wasAvailable && found) {
                this.emit(ResourceEvent.Discovered, {
                    resourceId: this.id,
                    category: this.category,
                    timestamp: new Date(),
                });
            } else if (wasAvailable && !found) {
                this.emit(ResourceEvent.Lost, {
                    resourceId: this.id,
                    category: this.category,
                    timestamp: new Date(),
                });
            }

            return found;
        } catch (error) {
            logger.error(`[${this.id}] Discovery failed`, error);
            this._status = DiscoveryStatus.NotFound;
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
                message: "Resource not available",
                timestamp: new Date(),
            };
        }

        const previousHealth = this._health;
        this._health = ResourceHealth.Checking;

        try {
            const result = await this.performHealthCheck();

            this._health = result.healthy ? ResourceHealth.Healthy : ResourceHealth.Unhealthy;
            this._lastHealthCheck = new Date();

            // Emit event if health changed
            if (previousHealth !== this._health) {
                this.emit(ResourceEvent.HealthChanged, {
                    resourceId: this.id,
                    category: this.category,
                    previousHealth,
                    currentHealth: this._health,
                    timestamp: new Date(),
                });
            }

            return result;
        } catch (error) {
            logger.error(`[${this.id}] Health check failed`, error);
            this._health = ResourceHealth.Unhealthy;

            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
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

        // Start new interval
        this.healthCheckInterval = setInterval(async () => {
            try {
                await this.healthCheck();
            } catch (error) {
                logger.error(`[${this.id}] Periodic health check failed`, error);
            }
        }, this.config.healthCheck.intervalMs);

        logger.debug(`[${this.id}] Started health monitoring, interval: ${this.config.healthCheck.intervalMs}ms`);
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

        // Reset state
        this._status = DiscoveryStatus.NotFound;
        this._health = ResourceHealth.Unknown;
        this.isInitialized = false;

        // Remove all event listeners
        this.removeAllListeners();
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

