/**
 * Unified Event System Service
 * 
 * Provides singleton initialization and lifecycle management for the unified event system.
 * This service ensures the unified event system is properly initialized and integrated
 * with the execution architecture through the compatibility adapter.
 */

import { type Logger } from "winston";
import { CacheService } from "../../../redisConn.js";
import { createUnifiedEventSystem, type UnifiedEventSystemConfig } from "../index.js";
import { type IEventBus } from "../types.js";

/**
 * Global state for the unified event system
 * 
 * NOTE: Adapters removed in Phase 4 of event adapter migration.
 * SwarmContextManager now emits events directly.
 */
let unifiedEventSystem: IEventBus | null = null;
let initialized = false;

/**
 * Unified Event System Service
 * 
 * Manages the lifecycle of the unified event system and provides
 * integration with the existing execution architecture.
 */
export class UnifiedEventSystemService {
    private static instance: UnifiedEventSystemService | null = null;
    private readonly logger: Logger;

    private constructor(logger: Logger = logger) {
        this.logger = logger;
    }

    /**
     * Get singleton instance
     */
    static getInstance(logger?: Logger): UnifiedEventSystemService {
        if (!UnifiedEventSystemService.instance) {
            UnifiedEventSystemService.instance = new UnifiedEventSystemService(logger);
        }
        return UnifiedEventSystemService.instance;
    }

    /**
     * Initialize the unified event system
     * 
     * Phase 4: Simplified initialization without adapters.
     * SwarmContextManager now handles direct event emission.
     */
    async initialize(): Promise<void> {
        if (initialized) {
            this.logger.debug("[UnifiedEventSystemService] Already initialized");
            return;
        }

        try {
            this.logger.info("[UnifiedEventSystemService] Initializing unified event system...");

            // Determine configuration based on environment
            const config = this.createConfiguration();

            // Create the unified event system
            unifiedEventSystem = await createUnifiedEventSystem(config);

            // Start the unified event system
            await unifiedEventSystem.start();

            initialized = true;

            this.logger.info("[UnifiedEventSystemService] Unified event system initialized successfully (direct integration)", {
                backend: config.backend,
                enableMetrics: config.enableMetrics,
                enableReplay: config.enableReplay,
            });

        } catch (error) {
            this.logger.error("[UnifiedEventSystemService] Failed to initialize unified event system", {
                error: error instanceof Error ? error.message : String(error),
                stack: error instanceof Error ? error.stack : undefined,
            });
            throw error;
        }
    }

    /**
     * Shutdown the unified event system
     */
    async shutdown(): Promise<void> {
        if (!initialized) {
            return;
        }

        try {
            this.logger.info("[UnifiedEventSystemService] Shutting down unified event system...");

            if (unifiedEventSystem) {
                await unifiedEventSystem.stop();
            }

            initialized = false;
            unifiedEventSystem = null;

            this.logger.info("[UnifiedEventSystemService] Unified event system shutdown complete");

        } catch (error) {
            this.logger.error("[UnifiedEventSystemService] Error during shutdown", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Get the unified event system instance (for direct use)
     */
    getUnifiedEventSystem(): IEventBus | null {
        return unifiedEventSystem;
    }


    /**
     * Check if the system is initialized
     */
    isInitialized(): boolean {
        return initialized;
    }

    /**
     * Create configuration for the unified event system
     */
    private createConfiguration(): UnifiedEventSystemConfig {
        const isProduction = process.env.NODE_ENV === "production";
        const redisUrl = CacheService.getRedisUrl();

        const config: UnifiedEventSystemConfig = {
            // Backend selection
            backend: isProduction && redisUrl ? "redis" : "memory",

            // Redis configuration (if using Redis)
            redis: redisUrl ? {
                url: redisUrl,
                streamName: "vrooli:unified:events",
                maxHistorySize: parseInt(process.env.EVENT_HISTORY_SIZE || "50000", 10),
                eventTtl: parseInt(process.env.EVENT_TTL_DAYS || "7", 10) * 24 * 60 * 60, // Convert days to seconds
            } : undefined,

            // Performance settings
            batchSize: parseInt(process.env.EVENT_BATCH_SIZE || "100", 10),
            maxEventHistory: parseInt(process.env.MAX_EVENT_HISTORY || "10000", 10),

            // Integration settings
            logger: this.logger,
            enableMetrics: process.env.ENABLE_EVENT_METRICS !== "false",
            enableReplay: process.env.ENABLE_EVENT_REPLAY !== "false",
        };

        this.logger.debug("[UnifiedEventSystemService] Created configuration", {
            backend: config.backend,
            redisConfigured: !!config.redis,
            enableMetrics: config.enableMetrics,
            enableReplay: config.enableReplay,
        });

        return config;
    }
}

/**
 * Global initialization function for use in singletons.ts
 */
export async function initUnifiedEventSystem(): Promise<void> {
    const service = UnifiedEventSystemService.getInstance();
    await service.initialize();
}

/**
 * Global shutdown function for use in process cleanup
 */
export async function shutdownUnifiedEventSystem(): Promise<void> {
    const service = UnifiedEventSystemService.getInstance();
    await service.shutdown();
}

/**
 * Get the unified event system instance (convenience function)
 */
export function getUnifiedEventSystem(): IEventBus | null {
    const service = UnifiedEventSystemService.getInstance();
    return service.getUnifiedEventSystem();
}

