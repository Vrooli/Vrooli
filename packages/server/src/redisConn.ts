/* ------------------------------------------------------------------
 * cacheService.ts – Shared Redis-backed cache layer
 * ------------------------------------------------------------------
 *
 *  ▌ Overview
 *  • A **typed** key/value cache that sits in front of Redis Streams,
 *    Postgres, external APIs, … anything slow you want to memoise.
 *  • Exposes an L1 in-process LRU and an L2 Redis store (single node
 *    or Cluster/Sentinel – autodetected from REDIS_URL).
 *  • Designed as a **singleton** so the rest of the codebase can
 *    call `CacheService.get()` without passing handles around.
 *
 *  ▌ Why not reuse RedisStreamBus?
 *  • The event bus needs exactly-once semantics and consumer-group
 *    features; the cache just needs low-latency GET/SET with TTLs.
 *  • Separating concerns keeps both classes smaller & easier to test.
 * ------------------------------------------------------------------ */
import { LRUCache, MINUTES_15_S, SECONDS_1_MS } from "@vrooli/shared";
import IORedis, { type Cluster, type Redis } from "ioredis";
import { logger } from "./events/logger.js";
import { type ServiceHealth } from "./services/health.js";

const DEFAULT_REDIS_PORT = 6379;
const FALLBACK_DEFAULT_TTL_SEC = MINUTES_15_S;

/**
 * Get the Redis URL from the environment variable.
 * This is a function because it allows us to mock the Redis URL for testing.
 * 
 * @returns The Redis URL
 */
export function getRedisUrl() {
    const url = process.env.REDIS_URL;
    if (!url) {
        const message = "REDIS_URL environment variable is not set!";
        logger.error(message);
        throw new Error(message);
    }
    return url;
}

/* ------------------------------------------------------------------ */
/* 1)  Types & defaults                                               */
/* ------------------------------------------------------------------ */

export interface CacheOptions {
    /** Default TTL in seconds if none is supplied to .set() */
    defaultTtl: number;
    /** Max items held in the local L1 cache (0 = disable) */
    localLruSize: number;
    /** Maximum reconnect delay (ms) */
    maxReconnectDelayMs: number;
    /** Multiplier for back-off (ms) */
    reconnectDelayMultiplier: number;
    /** Abort after n attempts (Infinity = never) */
    maxReconnectAttempts: number;
    /** Optional key prefix so multiple services can share a cluster */
    namespace?: string;
}

const DEFAULT_OPTS: CacheOptions = {
    defaultTtl: parseInt(process.env.CACHE_DEFAULT_TTL || "300"), // 5 min
    localLruSize: parseInt(process.env.CACHE_LOCAL_LRU_SIZE || "1000"),
    maxReconnectDelayMs: 5_000,
    reconnectDelayMultiplier: 200,
    maxReconnectAttempts: 25,
    namespace: process.env.CACHE_NAMESPACE || "vrooli",
};

/* ------------------------------------------------------------------ */
/* 2)  Utility helpers                                                */
/* ------------------------------------------------------------------ */

function namespaced(ns: string | undefined, key: string) {
    return ns ? `${ns}:${key}` : key;
}

/* ------------------------------------------------------------------ */
/* 3)  CacheService (singleton)                                       */
/* ------------------------------------------------------------------ */

export class CacheService {
    private static instance: CacheService;
    private opts: CacheOptions;
    private client!: Redis | Cluster;       // instantiated lazily
    private local?: LRUCache<string, unknown>;
    private publishErrorCount = 0;
    private connecting?: Promise<void>;     // ensure single flight
    private inFlightMemoRequests: Map<string, Promise<unknown>> = new Map();

    /* ── ctor is private; use CacheService.get() ───────────────────── */
    private constructor(opts: Partial<CacheOptions> = {}) {
        this.opts = { ...DEFAULT_OPTS, ...opts };

        // Validate and potentially fallback for defaultTtl
        if (isNaN(this.opts.defaultTtl) || this.opts.defaultTtl <= 0) {
            logger.warning(
                `[CacheService] Invalid CACHE_DEFAULT_TTL detected (value: ${this.opts.defaultTtl}). ` +
                `Falling back to default TTL: ${FALLBACK_DEFAULT_TTL_SEC} seconds.`,
            );
            this.opts.defaultTtl = FALLBACK_DEFAULT_TTL_SEC;
        }

        // Validate and potentially adjust for localLruSize
        // If localLruSize is NaN, NaN > 0 is false, so L1 cache is correctly disabled.
        // If it's 0 or negative (but not NaN), this condition also correctly disables L1.
        if (this.opts.localLruSize > 0) {
            // Ensure defaultTTLMs for LRUCache is valid if defaultTtl was corrected
            const lruDefaultTtlMs = this.opts.defaultTtl * SECONDS_1_MS;
            this.local = new LRUCache({
                limit: this.opts.localLruSize,
                defaultTTLMs: lruDefaultTtlMs,
            });
        } else if (this.opts.localLruSize < 0 || isNaN(this.opts.localLruSize)) {
            // Log if CACHE_LOCAL_LRU_SIZE was explicitly invalid and led to disabling L1.
            // Note: parseInt(undefined || "1000") gives 1000. parseInt("abc" || "1000") gives NaN.
            if (process.env.CACHE_LOCAL_LRU_SIZE !== undefined) { // only log if it was explicitly set to something invalid
                logger.warning(
                    `[CacheService] Invalid CACHE_LOCAL_LRU_SIZE detected (value: ${this.opts.localLruSize}). ` +
                    "L1 cache will be disabled.",
                );
            }
            // Ensure localLruSize is set to 0 if it was invalid, to reflect L1 being disabled.
            this.opts.localLruSize = 0;
        }
    }

    /** Idempotent accessor */
    static get(): CacheService {
        if (!CacheService.instance) {
            CacheService.instance = new CacheService();
        }
        return CacheService.instance;
    }

    /* ----------------------------------------------------------------
     * 3.1  Connection management (lazy, with back-off)                */
    /* ---------------------------------------------------------------- */

    private async ensure(): Promise<void> {
        if (this.client && ["ready", "connecting", "reconnecting"].includes(this.client.status)) {
            return;
        }
        // coalesce concurrent connects
        if (this.connecting) {
            return this.connecting;
        }

        let attempts = 0;
        const url = getRedisUrl();

        const connect = async () => {
            attempts += 1;
            try {
                const isCluster = url.includes(",");
                logger.info("[CacheService] connect(): Preparing to instantiate IORedis client.", {
                    url,
                    isCluster,
                    attempts,
                    options: {
                        enableReadyCheck: true,
                        connectTimeout: this.opts.maxReconnectDelayMs,
                        // Note: retryStrategy is not logged here as it's a function
                    },
                });
                const redisInstance = isCluster
                    ? new IORedis.Cluster(
                        url.split(",").map(u => ({ host: new URL(u).hostname, port: +new URL(u).port || DEFAULT_REDIS_PORT })),
                        {
                            enableReadyCheck: true,
                            // It's often good to set retryStrategy for Cluster as well
                            // Example: retryStrategy: times => Math.min(times * 50, 2000)
                        },
                    )
                    : new IORedis(url, {
                        enableReadyCheck: true,
                        connectTimeout: this.opts.maxReconnectDelayMs,
                        // Recommended: Add a retryStrategy for standalone instances too
                        // retryStrategy: times => Math.min(times * 50, 2000),
                        // Keep the client from re-connecting if it's not listening for 'error' events
                        // lazyConnect: true, // Consider this if you want to control connection timing more explicitly
                    });

                // Instead of explicit connect, let ioredis handle it. We can wait for 'ready' or check status.
                // await redis.connect(); 
                this.client = redisInstance;

                // Wait for the client to be ready if enableReadyCheck is true
                // This might involve listening to a 'ready' event or periodically checking status
                // For simplicity in this step, we'll assume ioredis handles this and check status after a brief moment
                // A more robust solution might involve a new Promise that resolves on 'ready' event.
                if (this.client.status !== "ready") {
                    // Give ioredis a moment to connect, especially with enableReadyCheck
                    await new Promise(resolve => setTimeout(resolve, 100)); // Small delay
                }

                // After attempting to connect or giving it a moment, check the status.
                // 'ready' is the desired state when enableReadyCheck is true.
                if (this.client.status === "ready") {
                    logger.info(`[CacheService] connect(): IORedis instance is ready. Status: ${this.client.status}`);
                } else {
                    logger.warning(`[CacheService] connect(): IORedis instance created but not initially ready. Status: ${this.client.status}. Waiting for 'ready' event or timeout.`);

                    // This promise waits for either the 'ready' event or a timeout/error.
                    await new Promise<void>((resolve, reject) => {
                        const errorListener: ((err: Error) => void) | undefined =
                            (err: Error) => {
                                clearTimeout(timeoutId);
                                if (readyListener) this.client.removeListener("ready", readyListener);
                                logger.error("[CacheService] IORedis client emitted error during connection attempt:", {
                                    message: err.message,
                                    name: err.name,
                                    code: (err as any).code, // Using as any for potentially non-standard properties like code
                                    stack: err.stack,
                                });
                                reject(err); // Reject with the actual error from ioredis
                            };

                        const readyListener: (() => void) | undefined = () => {
                            clearTimeout(timeoutId);
                            if (errorListener) this.client.removeListener("error", errorListener);
                            logger.info(`[CacheService] connect(): IORedis client became ready after waiting. Status: ${this.client.status}`);
                            resolve();
                        };

                        const timeoutId = setTimeout(() => {
                            if (errorListener) this.client.removeListener("error", errorListener);
                            if (readyListener) this.client.removeListener("ready", readyListener);
                            reject(new Error(`Redis client failed to become ready within ${this.opts.maxReconnectDelayMs}ms. Final status: ${this.client.status}`));
                        }, this.opts.maxReconnectDelayMs);

                        this.client.once("ready", readyListener);
                        this.client.once("error", errorListener); // Catches errors that might prevent 'ready'
                    });
                    // If the above promise resolved, the client is ready.
                    // If it rejected, the error would have been thrown and caught by the outer catch block.
                }

            } catch (err) {
                const errorDetails: Record<string, any> = { attempts };
                if (err instanceof Error) {
                    errorDetails.message = err.message;
                    errorDetails.name = err.name;
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    if (err.code) {
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        errorDetails.code = err.code;
                    }
                    // Adding stack for more detailed debugging, could be conditional if too verbose
                    errorDetails.stack = err.stack;
                } else {
                    errorDetails.errorObject = String(err);
                }
                logger.error("[CacheService] connect(): redis.connect() failed.", errorDetails);
                throw err;
            }
        };

        this.connecting = connect().finally(() => {
            if (this.client) {
                logger.info(`[CacheService] connect() promise finished. Final client status: ${this.client.status}`);
            } else {
                logger.warning("[CacheService] connect() promise finished, but this.client is not set (connection likely failed definitively).");
            }
            this.connecting = undefined;
        });
        await this.connecting;
        if (this.client) {
            logger.info(`[CacheService] ensure() returning. Client status: ${this.client.status}`);
        } else {
            logger.error("[CacheService] ensure() returning, but this.client is not set. Connection failed.");
        }
    }

    /* ----------------------------------------------------------------
     * 3.2  Public API                                                 
     * ----------------------------------------------------------------
     * All methods auto-serialise JSON so callers can store objects
     * without manually calling JSON.stringify / parse.
     * ---------------------------------------------------------------- */

    /** GET – returns parsed value or null */
    async get<T>(key: string): Promise<T | null> {
        const nsKey = namespaced(this.opts.namespace, key);

        // L1
        if (this.local?.has(nsKey)) return this.local.get(nsKey) as T;

        // L2
        await this.ensure();
        const txt = await this.client.get(nsKey);
        if (txt == null) return null;
        try {
            const val = JSON.parse(txt) as T;
            // When populating L1 from L2, L1 will use its default TTL policy,
            // as the original TTL from L2 isn't typically available with the GET response.
            this.local?.set(nsKey, val);
            return val;
        } catch (parseError) {
            logger.error(`[CacheService] Failed to parse JSON for key ${nsKey}`, {
                key: nsKey,
                error: parseError instanceof Error ? parseError.message : String(parseError),
            });
            this.local?.delete(nsKey); // Remove from L1 to prevent serving corrupted data
            try {
                // Attempt to delete from L2, but don't let failure here mask the parseError
                await this.client.del(nsKey);
            } catch (delError) {
                logger.error(`[CacheService] Failed to delete malformed key ${nsKey} from L2 after parse error`, {
                    key: nsKey,
                    originalError: parseError instanceof Error ? parseError.message : String(parseError),
                    deleteError: delError instanceof Error ? delError.message : String(delError),
                });
            }
            return null;
        }
    }

    /**
     * SET with optional TTL (seconds).  If ttl is 0 or undefined,
     * CacheService.defaultTtl is used.
     */
    async set<T>(key: string, value: T, ttlSec?: number): Promise<void> {
        const nsKey = namespaced(this.opts.namespace, key);
        const ttl = ttlSec ?? this.opts.defaultTtl;
        let stringifiedValue: string;

        try {
            stringifiedValue = JSON.stringify(value);
        } catch (stringifyError) {
            logger.error(`[CacheService] Failed to stringify value for key ${nsKey} prior to Redis SET`, {
                key: nsKey,
                valueType: typeof value,
                valuePreview: "<value_not_logged_due_to_stringify_error>",
                error: stringifyError instanceof Error ? stringifyError.message : String(stringifyError),
            });
            throw stringifyError; // Re-throw to ensure the operation fails as expected
        }

        // L2
        await this.ensure();
        try {
            await this.client.set(nsKey, stringifiedValue, "EX", ttl);
        } catch (err) {
            this.publishErrorCount += 1;
            logger.error("[CacheService] Redis SET failed", {
                key: nsKey,
                err: err instanceof Error ? err.message : String(err),
            });
            throw err;
        }

        // L1 - only if L2 set was successful
        if (this.local) {
            const ttlMs = ttl * SECONDS_1_MS;
            this.local.set(nsKey, value, ttlMs);
        }
    }

    /** DEL – removes from both tiers */
    async del(key: string): Promise<void> {
        const nsKey = namespaced(this.opts.namespace, key);
        this.local?.delete(nsKey);
        await this.ensure();
        await this.client.del(nsKey);
    }

    /** Convenience helper for caches that wrap expensive fns */
    async memo<T>(key: string, ttlSec: number, fn: () => Promise<T>): Promise<T> {
        // `key` here is the original key, `get` and `set` will handle namespacing for cache operations.
        const cached = await this.get<T>(key);
        if (cached != null) return cached;

        // For the in-flight requests map, we should use a key that uniquely identifies
        // the operation, consistent with how cache keys are derived if namespacing is used.
        const mapKey = namespaced(this.opts.namespace, key);

        if (this.inFlightMemoRequests.has(mapKey)) {
            return this.inFlightMemoRequests.get(mapKey) as Promise<T>;
        }

        const executionPromise = (async () => {
            try {
                const freshValue = await fn();
                // `this.set` will handle namespacing for `key` internally.
                await this.set(key, freshValue, ttlSec);
                return freshValue;
            } finally {
                this.inFlightMemoRequests.delete(mapKey);
            }
        })();

        this.inFlightMemoRequests.set(mapKey, executionPromise);
        return executionPromise;
    }

    /* 3.3  Lua script support --------------------------------------- */

    /**
     * Registers a Lua script once and returns its SHA1 hash.
     * You can then call `evalsha()` directly on the underlying client.
     */
    async scriptLoad(name: string, lua: string): Promise<string> {
        await this.ensure();
        const sha = await this.client.script("LOAD", lua);
        logger.info(`[CacheService] Lua script "${name}" loaded (${sha})`);
        return sha as string;
    }

    /** Raw access to the underlying IORedis client for edge cases */
    async raw(): Promise<Redis | Cluster> {
        await this.ensure();
        // If ensure() completes without throwing, this.client is guaranteed to be initialized and connected.
        return this.client;
    }

    /* ----------------------------------------------------------------
     * 3.4  Health & shutdown                                          
     * ---------------------------------------------------------------- */

    async metrics(): Promise<ServiceHealth> {
        const now = Date.now();
        try {
            await this.ensure();
            const info = await this.client.info();
            const connected = true;

            // More robust parsing of used_memory & keyspace stats
            const lines = info.split(/\r?\n/);
            let usedMem = "0";

            for (const line of lines) {
                if (line.startsWith("used_memory:")) {
                    usedMem = line.split(":")[1] || "0";
                    break; // Found used_memory, no need to iterate further for this specific stat
                }
            }

            let numKeys: number;
            if (this.client instanceof IORedis.Cluster) {
                // For a Redis Cluster, client.info() typically returns info from a single node.
                // An aggregate key count for the entire cluster is not reliably available via a simple INFO command.
                // A more robust approach would involve querying DBSIZE on all master nodes and summing the results,
                // e.g., by iterating `this.client.nodes('master')`.
                // For simplicity here, we report 0, acknowledging this limitation.
                numKeys = 0;
                logger.debug("[CacheService] metrics: Reporting 0 keys for Redis Cluster. For an accurate count, a different strategy is needed.");
            } else if (this.client instanceof IORedis) { // Handles Redis standalone instances
                numKeys = await (this.client as Redis).dbsize();
            } else {
                // This case should ideally not be reached if ensure() correctly initializes the client
                logger.warning("[CacheService] metrics: Client type unknown, cannot determine key count accurately.");
                numKeys = 0;
            }

            return {
                healthy: true,
                status: "Operational",
                lastChecked: now,
                details: {
                    connected,
                    keys: numKeys,
                    usedMemory: Number(usedMem),
                    publishErrorCount: this.publishErrorCount,
                },
            };
        } catch (err) {
            return {
                healthy: false,
                status: "Down",
                lastChecked: now,
                details: {
                    connected: false,
                    keys: 0, // Report 0 keys on error
                    usedMemory: 0,
                    publishErrorCount: this.publishErrorCount,
                },
            };
        }
    }

    async close(): Promise<void> {
        this.local?.clear();
        if (!this.client) return;
        await this.client.quit();
    }
}
