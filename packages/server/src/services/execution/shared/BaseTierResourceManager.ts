/**
 * Base Tier Resource Manager
 * 
 * Abstract base class that eliminates duplication across all tier resource managers.
 * Provides common initialization, error handling, and lifecycle management.
 * 
 * Each tier extends this with their specific adapter and method implementations.
 */

import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { ResourceManager as UnifiedResourceManager } from "../cross-cutting/resources/resourceManager.js";
import { ResourcePoolManager } from "../cross-cutting/resources/resourcePool.js";
import { UsageTracker } from "../cross-cutting/resources/usageTracker.js";
import { RateLimiter } from "../cross-cutting/resources/rateLimiter.js";
import { 
    type TierResourceConfig, 
    type TierResourceAdapter,
    TierResourceConfigFactory 
} from "./ResourceManagerConfig.js";

/**
 * Abstract base class for all tier resource managers
 */
export abstract class BaseTierResourceManager<TAdapter extends TierResourceAdapter = TierResourceAdapter> {
    protected readonly logger: Logger;
    protected readonly eventBus: EventBus;
    protected readonly unifiedManager: UnifiedResourceManager;
    protected readonly adapter: TAdapter;
    
    // Shared components
    protected readonly poolManager: ResourcePoolManager;
    protected readonly usageTracker: UsageTracker;
    protected readonly rateLimiter: RateLimiter;
    
    // Configuration
    protected readonly config: TierResourceConfig;

    constructor(
        logger: Logger, 
        eventBus: EventBus, 
        tier: 1 | 2 | 3,
        customConfig?: Partial<TierResourceConfig>
    ) {
        this.logger = logger;
        this.eventBus = eventBus;
        
        // Get tier-specific configuration
        this.config = customConfig 
            ? TierResourceConfigFactory.mergeConfig(tier, customConfig)
            : TierResourceConfigFactory.getConfig(tier);
        
        // Initialize shared components
        this.initializeSharedComponents();
        
        // Create unified manager
        this.unifiedManager = this.createUnifiedManager();
        
        // Create tier-specific adapter
        this.adapter = this.createAdapter();
        
        // Setup event handlers
        this.setupEventHandlers();
        
        this.logger.info(`[BaseTierResourceManager] Initialized Tier ${tier} resource manager`, {
            tier: this.config.tier,
            eventTopic: this.config.eventTopic,
        });
    }

    /**
     * Initialize shared components used by all tiers
     */
    private initializeSharedComponents(): void {
        this.poolManager = new ResourcePoolManager(this.logger);
        this.usageTracker = new UsageTracker(this.logger);
        this.rateLimiter = new RateLimiter(this.logger, this.eventBus);
        
        // Configure rate limiter with tier-specific settings
        this.rateLimiter.configure([{
            resource: `tier${this.config.tier}`,
            limit: this.config.rateLimiterConfig.defaultLimit,
            window: this.config.rateLimiterConfig.windowMs,
            burstLimit: this.config.rateLimiterConfig.burstMultiplier 
                ? this.config.rateLimiterConfig.defaultLimit * this.config.rateLimiterConfig.burstMultiplier
                : undefined,
        }]);
    }

    /**
     * Create the unified resource manager with tier-specific configuration
     */
    private createUnifiedManager(): UnifiedResourceManager {
        return new UnifiedResourceManager(
            this.eventBus,
            {
                tier: this.config.tier,
                defaultLimits: this.config.defaultLimits,
                cleanupInterval: this.config.cleanupInterval,
                eventTopic: this.config.eventTopic,
            },
            {
                poolManager: this.poolManager,
                usageTracker: this.usageTracker,
                rateLimiter: this.rateLimiter,
            },
        );
    }

    /**
     * Setup common event handlers
     */
    protected setupEventHandlers(): void {
        // Listen for resource events
        this.eventBus.on(`${this.config.eventTopic}.allocated`, (event) => {
            this.onResourceAllocated(event.data);
        });
        
        this.eventBus.on(`${this.config.eventTopic}.released`, (event) => {
            this.onResourceReleased(event.data);
        });
        
        this.eventBus.on(`${this.config.eventTopic}.exhausted`, (event) => {
            this.onResourceExhausted(event.data);
        });
    }

    /**
     * Handle resource allocation events
     */
    protected onResourceAllocated(data: any): void {
        this.logger.debug(`[BaseTierResourceManager] Resource allocated`, {
            tier: this.config.tier,
            allocation: data,
        });
    }

    /**
     * Handle resource release events
     */
    protected onResourceReleased(data: any): void {
        this.logger.debug(`[BaseTierResourceManager] Resource released`, {
            tier: this.config.tier,
            release: data,
        });
    }

    /**
     * Handle resource exhaustion events
     */
    protected onResourceExhausted(data: any): void {
        this.logger.warn(`[BaseTierResourceManager] Resource exhausted`, {
            tier: this.config.tier,
            exhaustion: data,
        });
    }

    /**
     * Common error handling wrapper
     */
    protected async withErrorHandling<T>(
        operation: string,
        fn: () => Promise<T>,
        fallback?: () => T
    ): Promise<T> {
        try {
            return await fn();
        } catch (error) {
            this.logger.error(`[BaseTierResourceManager] ${operation} failed`, {
                tier: this.config.tier,
                error: error instanceof Error ? error.message : String(error),
            });
            
            if (fallback) {
                return fallback();
            }
            
            throw error;
        }
    }

    /**
     * Get the unified resource manager
     */
    public getUnifiedManager(): UnifiedResourceManager {
        return this.unifiedManager;
    }

    /**
     * Get the tier-specific adapter
     */
    public getAdapter(): TAdapter {
        return this.adapter;
    }

    /**
     * Get configuration
     */
    public getConfig(): TierResourceConfig {
        return { ...this.config };
    }

    /**
     * Cleanup resources
     */
    public async cleanup(): Promise<void> {
        this.logger.info(`[BaseTierResourceManager] Cleaning up Tier ${this.config.tier} resource manager`);
        
        try {
            // Stop unified manager (which handles most cleanup)
            if (typeof this.unifiedManager.stop === 'function') {
                await this.unifiedManager.stop();
            }
        } catch (error) {
            this.logger.error(`[BaseTierResourceManager] Cleanup failed`, {
                tier: this.config.tier,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    // Abstract methods that each tier must implement
    protected abstract createAdapter(): TAdapter;
}