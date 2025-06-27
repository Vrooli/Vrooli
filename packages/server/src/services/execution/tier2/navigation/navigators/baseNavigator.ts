import { type Logger } from "winston";
import {
    type Navigator,
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { GenericStore } from "../../../shared/GenericStore.js";
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
    private configCache: GenericStore<RoutineVersionConfigObject> | null = null;

    constructor(logger: Logger) {
        this.logger = logger;
    }
    
    /**
     * Ensures the config cache is initialized
     */
    protected async ensureCache(): Promise<GenericStore<RoutineVersionConfigObject>> {
        if (!this.configCache) {
            const redis = await CacheService.get().raw();
            this.configCache = new GenericStore<RoutineVersionConfigObject>(
                this.logger,
                redis as Parameters<typeof GenericStore>[1],
                {
                    keyPrefix: `navigator.${this.type}.configs`,
                    defaultTTL: 7200, // 2 hours
                    publishEvents: false, // Internal cache, no events needed
                },
            );
        }
        return this.configCache;
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
        this.ensureCache().then(cache => cache.set(routineId, routineConfig)).catch(err => 
            this.logger.warn("Failed to cache routine config", { 
                routineId, 
                navigatorType: this.type,
                error: err, 
            }),
        );

        return routineConfig;
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
        const cache = await this.ensureCache();
        const configResult = await cache.get(routineId);
        
        if (!configResult.success || !configResult.data) {
            throw new Error(`Routine config ${routineId} not in cache`);
        }

        return configResult.data;
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
