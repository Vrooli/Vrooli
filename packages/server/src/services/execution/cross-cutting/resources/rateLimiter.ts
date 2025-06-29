/**
 * Rate Limiter Interface for Resource Management
 * 
 * Provides basic rate limiting functionality for resource allocation.
 * Actual implementation can be plugged in as needed.
 */

export interface RateLimiter {
    /**
     * Check if an operation is allowed under rate limits
     * @param entityId - The entity to check (user, swarm, etc.)
     * @param operation - The operation type
     * @param count - Number of operations requested
     * @returns true if allowed, false if rate limited
     */
    checkLimit(entityId: string, operation: string, count: number): Promise<boolean>;
    
    /**
     * Reset rate limit for an entity
     * @param entityId - The entity to reset
     * @param operation - The operation type to reset
     */
    resetLimit(entityId: string, operation?: string): Promise<void>;
}

// Constants for rate limiting
const ONE_MINUTE_MS = 60_000;

/**
 * Simple in-memory rate limiter implementation
 * This is a placeholder implementation - production should use Redis-based rate limiting
 */
export class InMemoryRateLimiter implements RateLimiter {
    private static readonly DEFAULT_WINDOW_MS = ONE_MINUTE_MS;
    private limits = new Map<string, Map<string, { count: number; resetAt: number }>>();
    
    constructor(
        private defaultLimit: number = 100,
        private windowMs: number = InMemoryRateLimiter.DEFAULT_WINDOW_MS,
    ) {}
    
    async checkLimit(entityId: string, operation: string, count: number): Promise<boolean> {
        const now = Date.now();
        
        let entityLimits = this.limits.get(entityId);
        if (!entityLimits) {
            entityLimits = new Map();
            this.limits.set(entityId, entityLimits);
        }
        
        let operationLimit = entityLimits.get(operation);
        if (!operationLimit || operationLimit.resetAt < now) {
            operationLimit = {
                count: 0,
                resetAt: now + this.windowMs,
            };
            entityLimits.set(operation, operationLimit);
        }
        
        if (operationLimit.count + count > this.defaultLimit) {
            return false;
        }
        
        operationLimit.count += count;
        return true;
    }
    
    async resetLimit(entityId: string, operation?: string): Promise<void> {
        if (operation) {
            const entityLimits = this.limits.get(entityId);
            if (entityLimits) {
                entityLimits.delete(operation);
            }
        } else {
            this.limits.delete(entityId);
        }
    }
}
