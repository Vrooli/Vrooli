import { type Logger } from "winston";
import {
    type Navigator,
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
    HOURS_1_S,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { type Redis } from "ioredis";
import { CacheService } from "../../../../../redisConn.js";

/**
 * BaseNavigator - Abstract base class for all navigator implementations
 * 
 * Provides common functionality like caching, validation, and utility methods
 * that all navigators need. Specific navigator implementations extend this
 * class to handle their particular graph format (BPMN, Native, etc.).
 */
export abstract class BaseNavigator implements Navigator {
    abstract readonly type: string;
    abstract readonly version: string;
    
    protected readonly logger: Logger;
    private redis: Redis | null = null;
    private readonly cachePrefix: string;
    
    // Cache TTL constants
    private static readonly HOURS_2_S = 2 * HOURS_1_S; // 2 hours in seconds

    constructor(logger: Logger) {
        this.logger = logger;
        this.cachePrefix = `navigator.${this.type}.configs`;
    }
    
    /**
     * Ensures Redis connection is available
     */
    private async ensureRedis(): Promise<Redis> {
        if (!this.redis) {
            this.redis = await CacheService.get().raw() as Redis;
        }
        return this.redis;
    }

    /**
     * Creates a cache key for the given routine ID
     */
    private createCacheKey(routineId: string): string {
        return `${this.cachePrefix}:${routineId}`;
    }

    /**
     * Validates and caches a routine configuration
     */
    protected validateAndCache(routine: unknown): RoutineVersionConfigObject {
        if (!this.canNavigate(routine)) {
            throw new Error(`Invalid routine configuration for ${this.type} navigator`);
        }

        const routineConfig = routine as RoutineVersionConfigObject;
        
        // Generate routine ID for caching
        const routineId = this.generateRoutineId(routineConfig);
        
        // Fire and forget cache operation
        this.cacheConfig(routineId, routineConfig).catch(err => 
            this.logger.warn("Failed to cache routine config", { 
                routineId, 
                navigatorType: this.type,
                error: err instanceof Error ? err.message : String(err), 
            }),
        );

        return routineConfig;
    }

    /**
     * Caches a routine configuration in Redis
     */
    private async cacheConfig(routineId: string, config: RoutineVersionConfigObject): Promise<void> {
        try {
            const redis = await this.ensureRedis();
            const key = this.createCacheKey(routineId);
            const serialized = JSON.stringify(config);
            
            await redis.setex(key, BaseNavigator.HOURS_2_S, serialized);
        } catch (error) {
            this.logger.debug("Failed to cache routine config", {
                routineId,
                navigatorType: this.type,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Generates a deterministic routine ID for caching
     */
    protected generateRoutineId(config: RoutineVersionConfigObject): string {
        // Create a deterministic ID based on config content
        const contentString = JSON.stringify({
            type: this.type,
            version: config.__version,
            hasGraph: !!config.graph,
            graphType: config.graph?.__type,
            callTypes: {
                action: !!config.callDataAction,
                api: !!config.callDataApi,
                code: !!config.callDataCode,
                generate: !!config.callDataGenerate,
                smartContract: !!config.callDataSmartContract,
                web: !!config.callDataWeb,
            },
        });
        
        // Simple hash function
        let hash = 0;
        const HASH_SHIFT = 5;
        const HEX_BASE = 16;
        
        for (let i = 0; i < contentString.length; i++) {
            const char = contentString.charCodeAt(i);
            hash = ((hash << HASH_SHIFT) - hash) + char;
            hash = hash & hash; // Convert to 32-bit integer
        }
        
        return `${this.type}_config_${Math.abs(hash).toString(HEX_BASE)}`;
    }

    /**
     * Creates a location object
     */
    protected createLocation(nodeId: string, config: RoutineVersionConfigObject): Location {
        const routineId = this.generateRoutineId(config);
        
        return {
            id: `${routineId}-${nodeId}`,
            routineId,
            nodeId,
        };
    }

    /**
     * Retrieves cached routine config
     */
    async getCachedConfig(routineId: string): Promise<RoutineVersionConfigObject> {
        try {
            const redis = await this.ensureRedis();
            const key = this.createCacheKey(routineId);
            const data = await redis.get(key);
            
            if (!data) {
                throw new Error(`Routine config ${routineId} not in cache`);
            }

            const config = JSON.parse(data) as RoutineVersionConfigObject;
            return config;
        } catch (error) {
            this.logger.debug("Failed to retrieve cached routine config", {
                routineId,
                navigatorType: this.type,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error(`Routine config ${routineId} not in cache`);
        }
    }

    /**
     * Creates step info for single-step routines
     */
    protected createSingleStepInfo(config: RoutineVersionConfigObject): StepInfo {
        // Determine the type of single step based on what's configured
        let stepType = "action";
        let stepName = "Single Step";
        
        if (config.callDataAction) {
            stepType = "action";
            stepName = "Action Step";
        } else if (config.callDataApi) {
            stepType = "api";
            stepName = "API Call";
        } else if (config.callDataCode) {
            stepType = "code";
            stepName = "Code Execution";
        } else if (config.callDataGenerate) {
            stepType = "generate";
            stepName = "AI Generation";
        } else if (config.callDataSmartContract) {
            stepType = "contract";
            stepName = "Smart Contract";
        } else if (config.callDataWeb) {
            stepType = "web";
            stepName = "Web Search";
        }

        return {
            id: "single_step",
            name: stepName,
            type: stepType,
            description: "Single step routine execution",
            config,
        };
    }

    // Abstract methods that must be implemented by specific navigators
    abstract canNavigate(routine: unknown): boolean;
    abstract getStartLocation(routine: unknown): Location;
    abstract getAllStartLocations(routine: unknown): Location[];
    abstract getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]>;
    abstract isEndLocation(location: Location): Promise<boolean>;
    abstract getStepInfo(location: Location): Promise<StepInfo>;
    abstract getDependencies(location: Location): Promise<string[]>;
    abstract getParallelBranches(location: Location): Promise<Location[][]>;
    
    // Event-driven navigation methods
    abstract getLocationTriggers(location: Location): Promise<NavigationTrigger[]>;
    abstract getLocationTimeouts(location: Location): Promise<NavigationTimeout[]>;
    abstract canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean>;
}
