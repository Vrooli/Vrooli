// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-06-17
import { type CreditEntryType, type CreditSourceSystem } from "@prisma/client";
import { MINUTES_1_MS, nanoid, SECONDS_1_MS, SECONDS_5_MS } from "@vrooli/shared";
import IORedis, { type Redis } from "ioredis";
import { EventEmitter } from "node:events";
import { logger } from "../events/logger.js";
import { getRedisUrl } from "../redisConn.js";
import { type ServiceHealth } from "./health.js";

/*──────────────────────── base event ───────────────────*/
/**
 * Base interface for all events, requiring only a type discriminator.
 */
export interface BaseEvent {
    type: string;
}

/*──────────────────────── financial events ───────────────────*/
export interface BillingEvent extends BaseEvent {
    type: "billing:event";
    /** idempotency key (UUID v4 from emitter) */
    id: string;
    /** credit_account PK (user *or* team), NOT the teamId or userId */
    accountId: string;          // bigint → string for JSON safety
    /** signed amount.  + = top-up, – = spend. Represented as string for JSON safety. */
    delta: string;
    /** typed reason */
    entryType: CreditEntryType;
    /** system that emitted it (Stripe, Scheduler, LLM, …) */
    source: CreditSourceSystem;
    /** any additional data you want to inspect later */
    meta?: Record<string, unknown>;
}

export interface BillingNegativeBalanceEvent extends BaseEvent {
    type: "billing.negative_balance";
    payload: {
        accountId: string;
        currentBalance: string; // bigint -> string for JSON safety
    };
}

/*──────────────────────── tool execution ───────────────────*/

/*──────────────────────── routine execution ───────────────────*/


/*──────────────────────── conversation execution ───────────────────*/
/**
 * Events related to conversations, extending BaseEvent with conversation-specific fields.
 */
export interface ConversationBaseEvent extends BaseEvent {
    conversationId: string;
    turnId?: number;
}

export interface ToolResultEvent extends ConversationBaseEvent {
    type: "tool.result";
    toolCallId: string;
    result: unknown;
}

export interface ScheduledTickEvent extends ConversationBaseEvent {
    type: "scheduled.tick";
}

export type ConversationEvent =
    | ToolResultEvent
    | ScheduledTickEvent;

/**
 * Union type representing all possible application events.
 * Add new event types here as needed.
 */
export type AppEvent =
    | ConversationEvent
    | BillingEvent
    | BillingNegativeBalanceEvent;

/**
 * Callback signature for event subscribers.
 */
type EventCallback = (evt: AppEvent) => void | Promise<void>;

/**
 * Redis XINFO STREAM response structure
 * The response is an array of alternating field names and values
 */
type XInfoStreamResponse = Array<string | number | null | Array<string | Array<[string, string]>>>;

/** Wait helper – avoids re-implementing a back-off timer each time. */
function sleep(ms: number) {
    return new Promise(r => setTimeout(r, ms));
}

/* ------------------------------------------------------------------
 * eventBus.ts – Unified Publish/Subscribe Spine for Internal Server Events
 * ------------------------------------------------------------------
 *
 * ### Overview
 * The Event Bus is a core infrastructure component designed for **internal server-to-server communication**. It routes events like chat messages and credit ledger updates to a single server for processing, ensuring consistency and preventing race conditions. This system decouples backend services, allowing them to communicate efficiently without direct dependencies.
 *
 * ### Purpose
 * - **Internal Event Handling:** Manages events within the backend, such as database updates or workflow triggers.
 * - **Consistency:** Ensures events are processed by a single server, avoiding duplication or conflicts.
 * - **Modularity:** Uses a publish-subscribe model to enhance backend service independence.
 *
 * ### Key Differences from SocketService
 * - **Event Bus:**
 *   - **Scope:** Internal, server-to-server communication.
 *   - **Purpose:** Processes backend events (e.g., chat message logging, credit updates).
 *   - **Processing:** Guarantees exactly-once processing per consumer group (Redis implementation).
 * - **SocketService:**
 *   - **Scope:** External, server-to-client communication.
 *   - **Purpose:** Sends real-time updates to users via websockets (e.g., notifications).
 *   - **Processing:** Focuses on delivery, not internal consistency.
 *
 * ### When to Use
 * - **Use Event Bus:** For internal backend events requiring processing, such as updating a credit ledger or logging a chat message.
 * - **Use SocketService:** For sending data directly to clients, like notifying a user of a new message.
 *
 * ### Event Examples
 * - `message.created`: A new chat message is created (processed internally, e.g., for logging).
 * - `credit.update`: A user's credit balance changes (updates the ledger).
 *
 * ### Technical Notes
 * - **Publish-Subscribe:** Services publish events and subscribe to receive them.
 * - **Backends:** InMemoryEventBus (testing), RedisStreamBus (production with exactly-once processing).
 * - **Singleton Rule:** Create one instance per Node.js process to manage resources efficiently.
 *
 * ### Best Practices
 * - Ensure event data is JSON-serializable.
 * - Implement error handling in subscribers to maintain stability.
 * 
 * ### Future Work
 * - Implement instance-affinity for RedisStreamBus, so events are processed by the same worker that published them 
 * under normal circumstances, and work is "stolen" by other workers when experiencing high load. This would 
 * improve cache efficiency during multi-turn conversations by reducing the number of servers that respond to the same conversation.
 * ------------------------------------------------------------------ */
export abstract class EventBus {
    private readonly lifecycleEmitter = new EventEmitter();  // internal for life‑cycle

    /**
     * Initialize the event bus (e.g. connect to Redis).
     */
    abstract init(): Promise<void>;

    /**
     * Subscribe to *all* events.  Called on startup by ConversationLoop
     * and any other interested service.
     * 
     * @param callback - The callback to handle each event
     */
    abstract subscribe(callback: EventCallback): void;

    /** 
     * Publish a single event. 
     * 
     * The call must resolve when the message is *accepted* 
     * by the back‑end (not necessarily delivered to all consumers).
     * 
     * @param event - The event to publish
     */
    abstract publish(event: AppEvent): Promise<void>;

    /** 
     * Flush buffers and close connections.  
     * 
     * Idempotent.
     */
    abstract close(): Promise<void>;

    /** 
     * Let workers hook into shutdown 
     * 
     * @param callback - The callback to handle the shutdown
     */
    onClose(callback: () => void) {
        this.lifecycleEmitter.once("close", callback);
    }

    /** 
     * Public method to trigger shutdown from outside the class 
     */
    triggerShutdown() {
        this.emitClose();
    }

    /** 
     * Call inside subclasses **before** network resources are closed 
     */
    protected emitClose() {
        this.lifecycleEmitter.emit("close");
    }
}

/* ------------------------------------------------------------------
 * 1) In‑memory, single‑process implementation ---------------------- */

/**
 * In‑memory, single‑process implementation.
 * 
 * WARNING: This implementation is not suitable for production use.
 * It is only intended for testing purposes and quick prototyping.
 */
export class InMemoryEventBus extends EventBus {
    private readonly emitter = new EventEmitter();

    async init(): Promise<void> {
        // No-op for in-memory bus
        return Promise.resolve();
    }

    /** Register a listener. Duplicate registrations are allowed. */
    subscribe(cb: EventCallback) {
        this.emitter.on("evt", cb);
    }

    async publish(evt: AppEvent) {
        this.emitter.emit("evt", evt);
    }

    async close() {
        this.emitClose();
        this.emitter.removeAllListeners();
    }
}

/* ------------------------------------------------------------------
 * 2) Redis Streams implementation ----------------------------------- */

/**
 * Interface defining comprehensive metrics for the RedisStreamBus.
 */
export interface RedisBusMetrics {
    /**
     * Indicates whether the bus is currently connected to Redis.
     * - true: Connected and operational.
     * - false: Disconnected, indicating potential issues with Redis connectivity.
     * Look for: Persistent false values may suggest network or Redis server problems.
     */
    connected: boolean;
    /**
     * Information about the stream's current state.
     */
    stream: {
        /**
         * The total number of messages currently in the stream.
         * - High values may indicate that messages are not being processed quickly enough.
         * Look for: Rapid increases could signal consumer slowdown or failure.
         */
        length: number;
        /**
         * The ID of the first message in the stream.
         * - Useful for debugging and understanding the stream's age.
         * Look for: Very old IDs might indicate unprocessed messages piling up.
         */
        firstId: string;
        /**
         * The ID of the last message in the stream.
         * - Helps track the most recent activity in the stream.
         * Look for: Compare with consumer group last delivered IDs to assess lag.
         */
        lastId: string;
    };
    /**
     * Detailed information about each consumer group associated with the stream.
     */
    consumerGroups: Record<string, {
        /**
         * The number of consumers in this group.
         * - Indicates the processing capacity of the group.
         * Look for: Low numbers with high pending messages may suggest under-resourcing.
         */
        consumersCount: number;
        /**
         * The number of messages pending delivery in this group.
         * - High values may indicate that the group is falling behind.
         * Look for: Persistent high counts could mean consumers are overwhelmed or crashed.
         */
        pending: number;
        /**
         * The lag of the consumer group, i.e., the number of messages in the stream after the last delivered ID.
         * - High lag indicates that the group is not keeping up with the stream.
         * Look for: Increasing lag suggests processing bottlenecks.
         */
        lag: number;
        /**
         * Detailed information about each consumer within the group.
         */
        consumers: Record<string, {
            /**
             * The number of pending messages assigned to this consumer.
             * - High values may indicate that this consumer is struggling or dead.
             * Look for: Uneven distribution could pinpoint a specific failing consumer.
             */
            pending: number;
            /**
             * The idle time of the consumer in milliseconds since its last interaction.
             * - High idle time may indicate that the consumer is not processing messages.
             * Look for: Values exceeding expected processing intervals suggest crashes or stalls.
             */
            idle: number;
        }>;
    }>;
    /**
     * The number of messages in the dead-letter queue.
     * - A growing number indicates that some messages are failing to process after retries.
     * Look for: Steady growth could signal systemic processing errors needing investigation.
     */
    deadLetterQueueSize: number;
    /**
     * The total number of errors encountered during message publishing since the bus started.
     * - High counts may indicate issues with Redis or message serialization.
     * Look for: Spikes could correlate with network issues or Redis downtime.
     */
    publishErrorCount: number;
    /**
     * The total number of errors encountered during subscribing or message processing since the bus started.
     * - High counts may suggest problems in consumer logic or Redis interactions.
     * Look for: Frequent increases might point to buggy handlers or resource exhaustion.
     */
    subscribeErrorCount: number;
}

/**
 * Stream bus options
 */
export type StreamBusOptions = {
    /**
     * The number of milliseconds to wait before running XAUTOCLAIM, which 
     * is used to rescue orphaned messages from the consumer group.
     */
    autoClaimEveryMs: number;
    /**
     * The number of messages to fetch from the stream per batch.
     */
    batchSize: number;
    /**
     * The number of milliseconds to block the consumer group when no messages are available.
     * This is useful to prevent the consumer group from polling the stream too frequently.
     */
    blockMs: number;
    /**
     * How long to wait for existing process loops to complete in close() before quitting.
     */
    closeTimeoutMs: number;
    /**
     * The consumer group name.
     * This is used to identify the consumer group in Redis (read: queue), which
     * allows us to process events in a single stream.
     */
    groupName: string;
    /**
     * The maximum reconnection delay in milliseconds.
     */
    maxReconnectDelayMs: number;
    /**
     * The maximum number of times reconnection will be attempted 
     * before giving up.
     */
    maxReconnectAttempts: number;
    /**
     * The maximum number of messages to keep in the stream.
     * This is used to prevent the stream from growing too large.
     */
    maxStreamSize: number;
    /**
     * How quickly the reconnection delay increases.
     */
    reconnectDelayMultiplier: number;
    /**
     * How many times to retry a message before sending it to a dead-letter stream.
     */
    retryCount: number;
    /** 
     * The stream name. 
     * This is used to identify the stream (read: topic) in Redis, which 
     * allows us to publish and subscribe to events on a single stream.
     */
    streamName: string;
}

const DEFAULT_STREAM_OPTIONS: StreamBusOptions = {
    autoClaimEveryMs: parseInt(process.env.REDIS_AUTO_CLAIM_EVERY_MS || MINUTES_1_MS.toString()),
    batchSize: parseInt(process.env.REDIS_BATCH_SIZE || "30"),
    blockMs: parseInt(process.env.REDIS_BLOCK_MS || "5000"),
    closeTimeoutMs: parseInt(process.env.REDIS_CLOSE_TIMEOUT_MS || SECONDS_1_MS.toString()),
    groupName: process.env.REDIS_GROUP_NAME || "vrooli-workers",
    maxReconnectDelayMs: parseInt(process.env.REDIS_MAX_RECONNECT_DELAY_MS || SECONDS_5_MS.toString()),
    maxReconnectAttempts: parseInt(process.env.REDIS_MAX_RECONNECT_ATTEMPTS || "10"),
    maxStreamSize: parseInt(process.env.REDIS_MAX_STREAM_SIZE || "100000"),
    reconnectDelayMultiplier: parseInt(process.env.REDIS_RECONNECT_DELAY_MULTIPLIER || "100"),
    retryCount: parseInt(process.env.REDIS_RETRY_COUNT || "3"),
    streamName: process.env.REDIS_STREAM_NAME || "vrooli.events",
};

/**
 * Determines reconnection delay based on the number of retries. 
 * The delay increases up to a maximum of 5 seconds.
 * 
 * @param retries - The number of retries
 * @param opts - The stream bus options
 * @returns The reconnection delay in milliseconds
 */
function getReconnectDelay(retries: number, opts: StreamBusOptions) {
    return Math.min(retries * opts.reconnectDelayMultiplier, opts.maxReconnectDelayMs);
}

export class RedisStreamBus extends EventBus {
    private options: StreamBusOptions;
    private client: Redis;
    private subRunning = false;
    private publishErrorCount = 0;
    private subscribeErrorCount = 0;
    private initialized = false; // Added to track initialization

    // Constructor
    constructor(options: Partial<StreamBusOptions> = {}) {
        super();
        this.options = { ...DEFAULT_STREAM_OPTIONS, ...options };
        this.client = new IORedis(getRedisUrl(), {
            retryStrategy: (retries) => {
                if (retries > this.options.maxReconnectAttempts) {
                    logger.error("[RedisStreamBus] Max retries reached");
                    this.emitClose();
                    return null;          // stop retrying
                }
                return getReconnectDelay(retries, this.options);
            },
        });
    }

    async init(): Promise<void> {
        if (this.initialized) return;
        await this.ensure();
        this.initialized = true;
    }

    private async ensure() {
        // Step 1: Ensure the client is connected or attempting to connect.
        if (this.client.status !== "ready" &&
            this.client.status !== "connect" && // 'connect' is when client is connected, before 'ready'
            this.client.status !== "connecting" &&
            this.client.status !== "reconnecting") {
            try {
                logger.info(`[RedisStreamBus] Attempting client.connect() for stream ${this.options.streamName}. Current status: ${this.client.status}`);
                await this.client.connect();
                logger.info(`[RedisStreamBus] Client.connect() successful for stream ${this.options.streamName}. Status: ${this.client.status}`);
            } catch (connectError) {
                logger.error(`[RedisStreamBus] Client.connect() failed for stream ${this.options.streamName}.`, {
                    errorName: (connectError as Error)?.name,
                    errorMessage: (connectError as Error)?.message,
                    errorStack: (connectError as Error)?.stack,
                    errorObj: JSON.stringify(connectError, Object.getOwnPropertyNames(connectError)),
                });
                throw connectError;
            }
        }

        // Step 2: Always attempt to create stream & consumer-group idempotently,
        // as client 'ready' status doesn't guarantee these structures exist.
        try {
            await this.client.xgroup(
                "CREATE",
                this.options.streamName,
                this.options.groupName,
                "$",
                "MKSTREAM",
            );
        } catch (xgroupError) {
            const err = xgroupError as Error;
            if (err.message && err.message.includes("BUSYGROUP")) {
                // Consumer group already exists. This is good! Can log if desired, but we'll skip it.
            } else {
                logger.error(`[RedisStreamBus] Failed to create consumer group ${this.options.groupName} for stream ${this.options.streamName}. Client status: ${this.client.status}`, {
                    errorName: err?.name,
                    errorMessage: err?.message,
                    errorStack: err?.stack,
                    errorObj: JSON.stringify(xgroupError, Object.getOwnPropertyNames(xgroupError)),
                });
                // If group creation fails for reasons other than BUSYGROUP, this is critical.
                throw xgroupError;
            }
        }
    }

    /** fire-and-forget publish */
    async publish(event: AppEvent) {
        await this.ensure();
        let serializedEvent: string;
        try {
            serializedEvent = JSON.stringify(event);
        } catch (error) {
            this.publishErrorCount++;
            logger.error("[RedisStreamBus] Serialization failed", {
                event,
                trace: "REDIS-STREAM-BUS-PUBLISH-SERIALIZATION-ERROR",
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
            });
            throw new Error("Event serialization failed");
        }
        try {
            await this.client.xadd(
                this.options.streamName, // stream name
                "*", // ID
                "j", serializedEvent, // message
                "MAXLEN", "~", this.options.maxStreamSize.toString(), // trim threshold
            );
        } catch (error) {
            this.publishErrorCount++;
            logger.error("[RedisStreamBus] Failed to publish event", {
                trace: "REDIS-STREAM-BUS-PUBLISH-ERROR",
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Subscribes to events, processing each exactly once per consumer group.
     * @param callback Callback to handle each event
     */
    subscribe(callback: EventCallback) {
        (async () => {
            await this.ensure();
            // Unique and descriptive consumer name
            const consumer = `${process.env.HOSTNAME || "unknown-pod"}-${nanoid()}`;
            this.subRunning = true;

            /* ── 1️⃣ normal consumption loop ────────────────────────── */
            const consumeLoop = async () => {
                while (this.subRunning) {
                    const res = await this.client.xreadgroup(
                        "GROUP", this.options.groupName, consumer,
                        "COUNT", this.options.batchSize.toString(),
                        "BLOCK", this.options.blockMs.toString(),
                        "STREAMS", this.options.streamName, ">",
                    ) as [string, [string, Record<string, string>][]][] | null;

                    if (!res) continue;

                    for (const [, messages] of res) {
                        for (const [id, message] of messages) {
                            try {
                                await callback(JSON.parse(message.j));
                                await this.client.xack(this.options.streamName, this.options.groupName, id);
                            } catch (e) {
                                this.subscribeErrorCount++;
                                logger.error("[RedisStreamBus] handler error", {
                                    trace: "REDIS-STREAM-BUS-CONSUME-LOOP",
                                    messageId: id,
                                    message: e instanceof Error ? e.message : String(e),
                                });
                                // Acknowledge to prevent infinite retries.
                                await this.client.xack(this.options.streamName, this.options.groupName, id);
                                // Send a new message to the stream with an incremented retry count
                                const retryCount = message.retryCount ? Number(message.retryCount) + 1 : 1;
                                if (retryCount < this.options.retryCount) {
                                    // Send a new message to the stream with an incremented retry count.
                                    await this.client.xadd(
                                        this.options.streamName, "*",
                                        "j", message.j,
                                        "retryCount", retryCount.toString(),
                                    );
                                } else {
                                    // Send to a dead-letter stream after a max number of retries.
                                    await this.client.xadd(
                                        `${this.options.streamName}:dead-letter`, "*",
                                        "j", message.j,
                                    );
                                }
                            }
                        }
                    }
                }
            };

            /* ── 2️⃣ periodic XAUTOCLAIM to rescue orphaned msgs ─────── */
            const autoClaimLoop = async () => {
                while (this.subRunning) {
                    try {
                        let cursor = "0-0";

                        const [nextStartId, claimedRaw] = await this.client.xautoclaim(
                            this.options.streamName,
                            this.options.groupName,
                            consumer,
                            this.options.autoClaimEveryMs,
                            cursor,
                            "COUNT",
                            this.options.batchSize.toString(),
                        ) as [string, Array<[string, Record<string, string>]>];  // ✨ only a *destructuring type annotation*

                        for (const [id, message] of claimedRaw) {
                            try {
                                await callback(JSON.parse(message.j));
                                await this.client.xack(this.options.streamName, this.options.groupName, id);
                            } catch (err) {
                                this.subscribeErrorCount++;
                                logger.error("[RedisStreamBus] autoclaim handler", {
                                    trace: "REDIS-STREAM-BUS-AUTOCLAIM-LOOP",
                                    messageId: id,
                                    message: err instanceof Error ? err.message : String(err),
                                });
                            }
                        }

                        cursor = nextStartId;
                    } catch (err) {
                        logger.error("[RedisStreamBus] XAUTOCLAIM error", {
                            trace: "REDIS-STREAM-BUS-AUTOCLAIM-LOOP",
                            name: err instanceof Error ? err.name : undefined,
                            message: err instanceof Error ? err.message : String(err),
                            stack: err instanceof Error && err.stack ? err.stack : undefined,
                        });
                    }
                    await sleep(this.options.autoClaimEveryMs);
                }
            };

            consumeLoop().catch(logger.error);
            autoClaimLoop().catch(logger.error);
        })().catch(logger.error);
    }

    async close() {
        this.subRunning = false;
        this.emitClose();
        // Give loops a moment to finish current batch
        await sleep(this.options.closeTimeoutMs);
        await this.client.quit();
    }

    /**
     * Collects comprehensive metrics about the Redis stream bus.
     * @returns A promise resolving to the BusMetrics object.
     */
    async metrics(): Promise<ServiceHealth> {
        const now = Date.now();

        try {
            await this.ensure();
            await this.client.ping();   // verifies connectivity

            /* ---------- STREAM BASICS -------------------------------- */
            const streamInfoRaw = await this.client.call(
                "XINFO", "STREAM", this.options.streamName,
            ) as XInfoStreamResponse;

            const { length, firstId, lastId } = this.parseXInfoStream(streamInfoRaw);

            /* ---------- GROUPS / CONSUMERS --------------------------- */
            const groupsInfoRaw = await this.client.call(
                "XINFO", "GROUPS", this.options.streamName,
            ) as any[];

            const consumerGroups: Record<string, {
                consumersCount: number;
                pending: number;
                lag: number;
                consumers: Record<string, { pending: number; idle: number }>;
            }> = {};

            for (let i = 0; i < groupsInfoRaw.length; i += 1) {
                const groupArr = groupsInfoRaw[i] as any[];
                const grp = Object.fromEntries(
                    new Array(groupArr.length / 2).fill(0).map((_, idx) => {
                        return [groupArr[idx * 2], groupArr[idx * 2 + 1]];
                    }),
                ) as Record<string, string>;

                const groupName = grp.name as string;
                const consumersCount = Number(grp.consumers);
                const pending = Number(grp.pending);
                const lag = Number(grp.lag);

                /* fetch consumers */
                const consumersInfoRaw = await this.client.call(
                    "XINFO", "CONSUMERS", this.options.streamName, groupName,
                ) as any[];

                const consumers: Record<string, { pending: number; idle: number }> = {};
                for (const c of consumersInfoRaw) {
                    const obj = Object.fromEntries(
                        new Array(c.length / 2).fill(0).map((_, idx) => [c[idx * 2], c[idx * 2 + 1]]),
                    ) as Record<string, string>;
                    consumers[obj.name] = { pending: Number(obj.pending), idle: Number(obj.idle) };
                }

                consumerGroups[groupName] = { consumersCount, pending, lag, consumers };
            }

            /* ---------- DEAD-LETTER --------------------------------- */
            const deadLetterStream = `${this.options.streamName}:dead-letter`;
            const deadLetterQueueSize = await this.client.xlen(deadLetterStream).catch(() => 0);

            return {
                healthy: true,
                status: "Operational",
                lastChecked: now,
                details: {
                    connected: true,
                    stream: { length, firstId, lastId },
                    consumerGroups,
                    deadLetterQueueSize,
                    publishErrorCount: this.publishErrorCount,
                    subscribeErrorCount: this.subscribeErrorCount,
                },
            };
        } catch (err) {
            return {
                healthy: false,
                status: "Down",
                lastChecked: now,
                details: {
                    connected: false,
                    stream: { length: 0, firstId: "", lastId: "" },
                    consumerGroups: {},
                    deadLetterQueueSize: 0,
                    publishErrorCount: this.publishErrorCount,
                    subscribeErrorCount: this.subscribeErrorCount,
                },
            };
        }
    }

    /**
     * Parses the raw XINFO STREAM response into a structured object.
     * @param raw The raw response array from Redis.
     * @returns An object with length, firstId, and lastId.
     */
    private parseXInfoStream(raw: XInfoStreamResponse): { length: number; firstId: string; lastId: string } {
        const length = parseInt(String(raw[1])); // 'length'
        const firstEntry = raw[9]; // 'first-entry'
        const lastEntry = raw[11]; // 'last-entry'
        const firstId = firstEntry && Array.isArray(firstEntry) ? String(firstEntry[0]) : "";
        const lastId = lastEntry && Array.isArray(lastEntry) ? String(lastEntry[0]) : "";
        return { length, firstId, lastId };
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// ─── BusService ─────────────────────────────────────────────────────────────
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Singleton service for managing the EventBus instance
 * Provides methods to start, stop, and access the bus
 */
export class BusService {
    private static instance: BusService;
    private bus: EventBus | null = null;
    private started = false;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Get the singleton instance of BusService
     */
    public static get(): BusService {
        if (!BusService.instance) {
            BusService.instance = new BusService();
        }
        return BusService.instance;
    }

    /**
     * Reset singleton instance (for testing)
     * This ensures each test gets a fresh connection
     */
    public static async reset(): Promise<void> {
        if (BusService.instance) {
            await BusService.instance.shutdown();
            BusService.instance = null;
        }
    }

    /**
     * Start the EventBus if not already started
     */
    public async startEventBus(): Promise<void> {
        if (this.started) return;
        this.bus = process.env.NODE_ENV === "test"
            ? new InMemoryEventBus()
            : new RedisStreamBus();
        await this.bus.init(); // Await the init method
        this.started = true;
    }

    /**
     * Get the EventBus instance, throws if not started
     */
    public getBus(): EventBus {
        if (!this.started || !this.bus) throw new Error("EventBus not started");
        return this.bus;
    }

    /**
     * Stop the EventBus and clean up resources
     */
    public async stopEventBus(): Promise<void> {
        if (!this.started || !this.bus) return;
        await this.bus.close();        // emits "close" for workers, quits Redis
        this.bus = null;
        this.started = false;
    }
}

type WorkerState = { started: boolean; res: unknown };

const STATE = new WeakMap<typeof BusWorker, WorkerState>();

export abstract class BusWorker {
    /** idempotent */
    static async start(): Promise<void> {
        const st = STATE.get(this);
        if (st?.started) return;

        await BusService.get().startEventBus();
        const bus = BusService.get().getBus();

        const res = await this.init(bus);

        bus.onClose(async () => {
            const cur = STATE.get(this);
            if (!cur?.started) return;
            await this.shutdown?.();
            STATE.set(this, { ...cur, started: false });
        });

        STATE.set(this, { started: true, res });
    }

    /** blocking accessor */
    static get(): unknown {
        const st = STATE.get(this);
        if (!st?.started) throw new Error(`${this.name}.start() not called`);
        return st.res;
    }

    /* ---------- to implement ---------- */
    protected static init(_bus: EventBus): Promise<unknown> | unknown { throw 0; }
    protected static shutdown?(): Promise<void> | void;
}
