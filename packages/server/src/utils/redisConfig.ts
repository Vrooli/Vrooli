/**
 * Centralized Redis connection configuration utility
 * 
 * Ensures all Redis connections across the application use consistent,
 * robust settings to prevent connection drops and retry issues.
 */

import { SECONDS_30_MS, SECONDS_5_MS } from "@vrooli/shared";
import { type RedisOptions } from "ioredis";
import { logger } from "../events/logger.js";

export type StandardRedisOptions = RedisOptions

/**
 * Standard Redis connection configuration used across all services
 * to ensure consistent behavior and prevent connection issues.
 */
export function getStandardRedisConfig(component: string): StandardRedisOptions {
    const baseConfig: StandardRedisOptions = {
        enableReadyCheck: true,
        maxRetriesPerRequest: null,
        enableOfflineQueue: true,
        lazyConnect: true,

        // Timeouts optimized for production stability
        connectTimeout: process.env.NODE_ENV === "test" ? SECONDS_5_MS : SECONDS_30_MS, // 30s for production startup
        commandTimeout: process.env.NODE_ENV === "test" ? SECONDS_5_MS : 15000, // 15s for production operations

        // Ultra-aggressive TCP keepalive to prevent WSL2/Docker NAT timeout issues  
        keepAlive: process.env.NODE_ENV === "test" ? 0 : SECONDS_5_MS, // 5s keepalive (ultra-aggressive for WSL2/Docker)

        // Standardized retry strategy across all components with enhanced debugging
        retryStrategy: (times: number) => {
            if (process.env.NODE_ENV === "test") {
                if (times > 2) return null;
                return times * 100;
            }

            // Fail fast if we're clearly connecting to the wrong host (native-linux issue)
            if (times === 1) {
                const redisUrl = process.env.REDIS_URL || "not_set";
                if (redisUrl.includes("redis:6379") && process.env.NODE_ENV === "development") {
                    logger.error("🚨 NATIVE-LINUX REDIS CONFIG ERROR: Using container hostname 'redis:6379' instead of native port mapping!", {
                        currentRedisUrl: redisUrl,
                        component,
                        suggestion: "Check if REDIS_URL is set correctly for native-linux mode (should be 127.0.0.1:PORT)",
                        attempt: times,
                    });
                }
            }

            // Production retry strategy with exponential backoff and jitter
            const baseDelay = Math.min(times * 100, 3000);
            const jitter = Math.random() * 100;
            const delay = baseDelay + jitter;

            // Enhanced logging for debugging connection issues - capture full context
            const stack = new Error().stack;
            const callingFunction = stack?.split("\n")[2]?.trim().replace(/^at\s+/, "") || "unknown";
            const fullStack = stack?.split("\n").slice(1, 6).map(line => line.trim()) || [];

            // Only log final failure, not intermediate retries

            if (times > 10) {
                logger.error(`${component} Redis connection failed after 10 attempts`, {
                    operation: "redis_connection",
                    component,
                });
                return null;
            }

            return delay;
        },

        // Standardized reconnection behavior
        reconnectOnError: (err) => {
            if (process.env.NODE_ENV === "test") return false;

            logger.error(`${component} Redis reconnect error`, {
                error: err.message,
                component,
            });
            return true; // Always try to reconnect in production
        },
    };

    return baseConfig;
}

/**
 * Apply standard Redis configuration to URL-based connections
 */
export function createRedisUrlConfig(url: string, component: string): StandardRedisOptions {
    const standardConfig = getStandardRedisConfig(component);

    // Parse URL to extract connection details
    const urlObj = new URL(url);

    return {
        ...standardConfig,
        host: urlObj.hostname,
        port: parseInt(urlObj.port) || 6379,
        password: urlObj.password || undefined,
        username: urlObj.username || undefined,
        db: urlObj.pathname && urlObj.pathname.length > 1 ? parseInt(urlObj.pathname.slice(1)) || 0 : 0,
    };
}

/**
 * Apply standard Redis configuration for environment-based connections
 */
export function createRedisEnvConfig(component: string): StandardRedisOptions {
    return {
        ...getStandardRedisConfig(component),
        host: process.env.REDIS_HOST || "localhost",
        port: parseInt(process.env.REDIS_PORT || "6379"),
        password: process.env.REDIS_PASSWORD,
        db: parseInt(process.env.REDIS_DB || "0"),
    };
}
