/**
 * Distributed Lock Service
 * 
 * Redis-based distributed locking for event handling race condition prevention.
 * Uses Redis SET with NX and EX options for atomic lock acquisition.
 */

import { type Redis } from "ioredis";
import { logger } from "../../events/logger.js";
import { type ILockService, type Lock, type LockOptions } from "./types.js";

const LONG_LIVED_LOCK_THRESHOLD_MS = 10_000;

/**
 * Redis-based distributed lock implementation
 */
export class RedisLockService implements ILockService {
    private redis: Redis;

    constructor(redis: Redis) {
        this.redis = redis;
    }

    async acquire(key: string, options: LockOptions): Promise<Lock> {
        const lockKey = `lock:${key}`;
        const lockValue = `${Date.now()}-${Math.random()}`;
        const maxRetries = options.retries ?? 0;
        const retryDelay = 100; // ms

        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            try {
                // Try to acquire lock atomically
                const result = await this.redis.set(
                    lockKey,
                    lockValue,
                    "PX", // milliseconds
                    options.ttl,
                    "NX", // only if not exists
                );

                if (result === "OK") {
                    logger.debug("Lock acquired", { key, lockValue, attempt });

                    return new RedisLock(
                        this.redis,
                        lockKey,
                        lockValue,
                        options.ttl,
                    );
                }

                // Lock already exists, wait and retry
                if (attempt < maxRetries) {
                    await new Promise(resolve => setTimeout(resolve, retryDelay));
                }
            } catch (error) {
                logger.error("Lock acquisition error", { key, attempt, error });

                if (attempt === maxRetries) {
                    throw error;
                }

                // Wait before retry
                await new Promise(resolve => setTimeout(resolve, retryDelay));
            }
        }

        throw new Error(`Failed to acquire lock after ${maxRetries + 1} attempts: ${key}`);
    }
}

/**
 * Redis lock implementation
 */
class RedisLock implements Lock {
    private released = false;
    private renewalInterval?: NodeJS.Timer;

    constructor(
        private redis: Redis,
        private key: string,
        private value: string,
        private ttl: number,
    ) {
        // Auto-renew lock if it's long-lived
        if (ttl > LONG_LIVED_LOCK_THRESHOLD_MS) {
            this.startRenewal();
        }
    }

    async release(): Promise<void> {
        if (this.released) {
            return;
        }

        this.released = true;

        // Stop renewal
        if (this.renewalInterval) {
            clearInterval(this.renewalInterval);
            this.renewalInterval = undefined;
        }

        try {
            // Use Lua script to ensure we only release our own lock
            const script = `
                if redis.call("get", KEYS[1]) == ARGV[1] then
                    return redis.call("del", KEYS[1])
                else
                    return 0
                end
            `;

            const result = await this.redis.eval(script, 1, this.key, this.value);

            if (result === 1) {
                logger.debug("Lock released", { key: this.key, value: this.value });
            } else {
                logger.warn("Lock was already released or expired", {
                    key: this.key,
                    value: this.value,
                });
            }
        } catch (error) {
            logger.error("Lock release error", { key: this.key, error });
            throw error;
        }
    }

    /**
     * Start automatic renewal of the lock
     */
    private startRenewal(): void {
        const renewalInterval = Math.floor(this.ttl / 3); // Renew at 1/3 of TTL

        this.renewalInterval = setInterval(async () => {
            if (this.released) {
                return;
            }

            try {
                // Renew lock if it's still ours
                const script = `
                    if redis.call("get", KEYS[1]) == ARGV[1] then
                        return redis.call("pexpire", KEYS[1], ARGV[2])
                    else
                        return 0
                    end
                `;

                const result = await this.redis.eval(
                    script,
                    1,
                    this.key,
                    this.value,
                    this.ttl.toString(),
                );

                if (result === 1) {
                    logger.debug("Lock renewed", { key: this.key });
                } else {
                    logger.warn("Lock renewal failed - lock lost", { key: this.key });
                    this.released = true;
                    clearInterval(this.renewalInterval!);
                }
            } catch (error) {
                logger.error("Lock renewal error", { key: this.key, error });
            }
        }, renewalInterval);
    }
}

/**
 * In-memory lock service for testing
 */
export class InMemoryLockService implements ILockService {
    private locks = new Map<string, { value: string; expiresAt: number }>();

    async acquire(key: string, options: LockOptions): Promise<Lock> {
        const lockKey = `lock:${key}`;
        const lockValue = `${Date.now()}-${Math.random()}`;
        const expiresAt = Date.now() + options.ttl;
        const maxRetries = options.retries ?? 0;
        const retryDelay = 100;

        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            // Clean up expired locks
            this.cleanupExpiredLocks();

            // Try to acquire lock
            if (!this.locks.has(lockKey)) {
                this.locks.set(lockKey, { value: lockValue, expiresAt });
                return new InMemoryLock(this.locks, lockKey, lockValue);
            }

            // Lock exists, wait and retry
            if (attempt < maxRetries) {
                await new Promise(resolve => setTimeout(resolve, retryDelay));
            }
        }

        throw new Error(`Failed to acquire lock after ${maxRetries + 1} attempts: ${key}`);
    }

    private cleanupExpiredLocks(): void {
        const now = Date.now();
        for (const [key, lock] of this.locks.entries()) {
            if (now > lock.expiresAt) {
                this.locks.delete(key);
            }
        }
    }
}

/**
 * In-memory lock implementation
 */
class InMemoryLock implements Lock {
    private released = false;

    constructor(
        private locks: Map<string, { value: string; expiresAt: number }>,
        private key: string,
        private value: string,
    ) { }

    async release(): Promise<void> {
        if (this.released) {
            return;
        }

        this.released = true;

        const lock = this.locks.get(this.key);
        if (lock && lock.value === this.value) {
            this.locks.delete(this.key);
        }
    }
}
