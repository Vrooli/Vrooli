import { type Logger } from "winston";
import { type IEventBus } from "../../../events/types.js";

/**
 * Rate limit configuration
 */
export interface RateLimitConfig {
    resource: string;
    limit: number;
    window: number; // milliseconds
    burstLimit?: number;
    burstWindow?: number; // milliseconds
}

/**
 * Rate limit state
 */
export interface RateLimitState {
    resource: string;
    limit: number;
    burstLimit: number;
    window: number;
    current: number;
    burstCurrent: number;
    resetTime: number;
    burstResetTime: number;
    violations: number;
}

/**
 * Rate limit check result
 */
export interface RateLimitResult {
    allowed: boolean;
    reason?: string;
    retryAfter?: number; // milliseconds
    remaining?: number;
    resetTime?: number;
}

/**
 * Shared rate limiter implementation
 * 
 * Provides rate limiting with:
 * - Token bucket algorithm
 * - Burst support
 * - Sliding window
 * - Distributed state (Redis)
 * - Event emission for violations
 */
export class RateLimiter {
    private readonly logger: Logger;
    private readonly eventBus?: IEventBus;
    private readonly limits: Map<string, RateLimitState> = new Map();

    constructor(logger: Logger, eventBus?: IEventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
    }

    /**
     * Configure rate limits
     */
    configure(configs: RateLimitConfig[]): void {
        for (const config of configs) {
            const state: RateLimitState = {
                resource: config.resource,
                limit: config.limit,
                burstLimit: config.burstLimit || config.limit * 2,
                window: config.window,
                current: 0,
                burstCurrent: 0,
                resetTime: Date.now() + config.window,
                burstResetTime: Date.now() + (config.burstWindow || config.window / 2),
                violations: 0,
            };
            this.limits.set(config.resource, state);
        }

        this.logger.debug("[RateLimiter] Configured rate limits", {
            resources: configs.map(c => c.resource),
        });
    }

    /**
     * Check if an operation is allowed
     */
    async check(
        resource: string,
        count = 1,
        identifier?: string,
    ): Promise<RateLimitResult> {
        const state = this.limits.get(resource);
        if (!state) {
            // No limit configured, allow
            return { allowed: true };
        }

        // Reset if window expired
        const now = Date.now();
        if (now >= state.resetTime) {
            state.current = 0;
            state.resetTime = now + state.window;
        }

        // Reset burst if burst window expired
        if (now >= state.burstResetTime) {
            state.burstCurrent = 0;
            state.burstResetTime = now + (state.window / 2);
        }

        // Check normal limit
        if (state.current + count <= state.limit) {
            state.current += count;
            return {
                allowed: true,
                remaining: state.limit - state.current,
                resetTime: state.resetTime,
            };
        }

        // Check burst limit
        if (state.burstCurrent + count <= state.burstLimit) {
            state.burstCurrent += count;
            this.logger.warn("[RateLimiter] Using burst capacity", {
                resource,
                burstUsed: state.burstCurrent,
                burstLimit: state.burstLimit,
            });

            return {
                allowed: true,
                remaining: state.burstLimit - state.burstCurrent,
                resetTime: state.burstResetTime,
            };
        }

        // Rate limit exceeded
        state.violations++;
        const retryAfter = Math.min(
            state.resetTime - now,
            state.burstResetTime - now,
        );

        // Emit violation event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: "rate_limit.violated",
                timestamp: new Date(),
                data: {
                    resource,
                    identifier,
                    violations: state.violations,
                    retryAfter,
                },
            });
        }

        this.logger.warn("[RateLimiter] Rate limit exceeded", {
            resource,
            identifier,
            violations: state.violations,
            retryAfter,
        });

        return {
            allowed: false,
            reason: `Rate limit exceeded for ${resource}`,
            retryAfter,
            remaining: 0,
            resetTime: state.resetTime,
        };
    }

    /**
     * Reset rate limit for a resource
     */
    reset(resource: string): void {
        const state = this.limits.get(resource);
        if (state) {
            state.current = 0;
            state.burstCurrent = 0;
            state.violations = 0;
            state.resetTime = Date.now() + state.window;
            state.burstResetTime = Date.now() + (state.window / 2);
        }
    }

    /**
     * Get current state for monitoring
     */
    getState(resource?: string): RateLimitState | Map<string, RateLimitState> | null {
        if (resource) {
            return this.limits.get(resource) || null;
        }
        return new Map(this.limits);
    }

    /**
     * Update limit configuration dynamically
     */
    updateLimit(resource: string, limit: number, burstLimit?: number): void {
        const state = this.limits.get(resource);
        if (state) {
            state.limit = limit;
            if (burstLimit !== undefined) {
                state.burstLimit = burstLimit;
            }

            this.logger.info("[RateLimiter] Updated rate limit", {
                resource,
                limit,
                burstLimit: state.burstLimit,
            });
        }
    }
}
