/**
 * Resource Manager Configuration
 * 
 * Shared configuration types and defaults for all tier resource managers.
 * Eliminates duplication while allowing tier-specific customization.
 */

import { type ResourceAmount } from "../cross-cutting/resources/resourceManager.js";

/**
 * Tier-specific resource configuration
 */
export interface TierResourceConfig {
    tier: 1 | 2 | 3;
    defaultLimits: ResourceAmount;
    cleanupInterval: number;
    eventTopic: string;
    rateLimiterConfig: {
        defaultLimit: number;
        windowMs: number;
        burstMultiplier?: number;
    };
}

/**
 * Base configuration class with tier-specific defaults
 */
export class TierResourceConfigFactory {
    /**
     * Get tier-specific configuration
     */
    static getConfig(tier: 1 | 2 | 3): TierResourceConfig {
        switch (tier) {
            case 1:
                return {
                    tier: 1,
                    defaultLimits: {
                        credits: 100000,
                        time: 3600000, // 1 hour
                        memory: 1024 * 1024 * 1024, // 1GB
                        tokens: 1000000,
                        apiCalls: 10000,
                    },
                    cleanupInterval: 60000,
                    eventTopic: "swarm.resources",
                    rateLimiterConfig: {
                        defaultLimit: 1000,
                        windowMs: 60000,
                    },
                };
            
            case 2:
                return {
                    tier: 2,
                    defaultLimits: {
                        credits: 10000,
                        time: 300000, // 5 minutes
                        memory: 512 * 1024 * 1024, // 512MB
                        tokens: 100000,
                        apiCalls: 1000,
                    },
                    cleanupInterval: 30000,
                    eventTopic: "run.resources",
                    rateLimiterConfig: {
                        defaultLimit: 100,
                        windowMs: 60000,
                    },
                };
            
            case 3:
                return {
                    tier: 3,
                    defaultLimits: {
                        credits: 1000,
                        time: 60000, // 1 minute
                        memory: 256 * 1024 * 1024, // 256MB
                        tokens: 10000,
                        apiCalls: 100,
                    },
                    cleanupInterval: 60000,
                    eventTopic: "step.resources",
                    rateLimiterConfig: {
                        defaultLimit: 10,
                        windowMs: 60000,
                        burstMultiplier: 1.5,
                    },
                };
            
            default:
                throw new Error(`Unsupported tier: ${tier}`);
        }
    }
    
    /**
     * Merge custom configuration with defaults
     */
    static mergeConfig(tier: 1 | 2 | 3, customConfig: Partial<TierResourceConfig>): TierResourceConfig {
        const defaultConfig = this.getConfig(tier);
        return {
            ...defaultConfig,
            ...customConfig,
            defaultLimits: {
                ...defaultConfig.defaultLimits,
                ...customConfig.defaultLimits,
            },
            rateLimiterConfig: {
                ...defaultConfig.rateLimiterConfig,
                ...customConfig.rateLimiterConfig,
            },
        };
    }
}

/**
 * Resource adapter types for each tier
 */
export type TierResourceAdapter = 
    | import("../cross-cutting/resources/adapters.js").SwarmResourceAdapter
    | import("../cross-cutting/resources/adapters.js").RunResourceAdapter  
    | import("../cross-cutting/resources/adapters.js").StepResourceAdapter;
