/**
 * Production-ready queue factory implementation
 * Addresses connection isolation issues while maintaining efficiency
 */
// AI_CHECK: TYPE_SAFETY=critical-generics | LAST: 2025-07-04 - Fixed generic task data types and removed any usage

import { DAYS_1_S, MINUTES_1_MS, SECONDS_10_MS } from "@vrooli/shared";
import { Queue, QueueEvents, Worker, type ConnectionOptions, type Job, type JobsOptions, type WorkerOptions } from "bullmq";
import { Redis, type Cluster, type RedisOptions } from "ioredis";
import { logger } from "../events/logger.js";
import { extractOwnerId } from "../utils/typeGuards.js";
import type { BaseTaskData } from "./taskTypes.js";

/**
 * Safely serialize error objects, handling circular references and non-enumerable properties
 * @param error Any error object or value
 * @returns Serializable error representation
 */
function serializeError(error: unknown): Record<string, unknown> {
    // Handle null/undefined
    if (error == null) {
        return { error: "null or undefined error" };
    }

    // Handle Error instances
    if (error instanceof Error) {
        const serialized: Record<string, unknown> = {
            name: error.name,
            message: error.message,
            stack: error.stack,
        };

        // Extract additional enumerable properties
        const errorObj = error as any;
        for (const key of Object.keys(errorObj)) {
            if (key !== "name" && key !== "message" && key !== "stack") {
                try {
                    const value = errorObj[key];
                    // Skip circular references and functions
                    if (typeof value !== "function" && value !== error) {
                        serialized[key] = value;
                    }
                } catch {
                    serialized[key] = "[Unserializable]";
                }
            }
        }

        // Special handling for Redis/ioredis errors
        if ("code" in errorObj) serialized.code = errorObj.code;
        if ("errno" in errorObj) serialized.errno = errorObj.errno;
        if ("syscall" in errorObj) serialized.syscall = errorObj.syscall;
        if ("hostname" in errorObj) serialized.hostname = errorObj.hostname;
        if ("port" in errorObj) serialized.port = errorObj.port;
        if ("address" in errorObj) serialized.address = errorObj.address;

        return serialized;
    }

    // Handle objects with circular references
    if (typeof error === "object") {
        try {
            // Try to stringify with replacer for circular refs
            const seen = new WeakSet();
            const stringified = JSON.stringify(error, (key, value) => {
                if (typeof value === "object" && value !== null) {
                    if (seen.has(value)) {
                        return "[Circular Reference]";
                    }
                    seen.add(value);
                }
                if (typeof value === "function") {
                    return "[Function]";
                }
                return value;
            });
            return { serialized: JSON.parse(stringified), type: "object" };
        } catch (e) {
            // If stringify fails, extract what we can
            return {
                type: "object",
                error: String(error),
                serializationError: e instanceof Error ? e.message : String(e),
                keys: Object.keys(error as any).slice(0, 10), // First 10 keys for debugging
            };
        }
    }

    // Fallback for primitives and other types
    return { error: String(error), type: typeof error };
}

/**
 * Default job options for all queues unless overridden.
 */
export const DEFAULT_JOB_OPTIONS: JobsOptions = {
    removeOnComplete: { age: DAYS_1_S, count: 5_000 },
    removeOnFail: { age: DAYS_1_S, count: 5_000 },
    attempts: 5,
    backoff: { type: "exponential", delay: SECONDS_10_MS },
};

/**
 * Base worker options for all queues unless overridden.
 */
export const BASE_WORKER_OPTS: Partial<WorkerOptions> = {
    lockDuration: MINUTES_1_MS,
    concurrency: 4,
};

// Sample end index for active jobs in health checks (0-9 => 10 jobs)
const SAMPLE_ACTIVE_JOB_END = 9;

export enum QueueStatus {
    Healthy = "healthy",
    Degraded = "degraded",
    Down = "down"
}

type ActiveJobInfo = {
    id: string;
    name: string;
    duration: number;        // ms
    processedOn?: number;    // epoch ms
};

/**
 * Health information for a queue, including status and counts.
 */
export interface QueueHealth {
    status: QueueStatus;
    jobCounts: Record<string, number>;
    activeJobs: ActiveJobInfo[];
}

/**
* Cheap runtime probe that looks only at Redis metadata – no job bodies are loaded.
* Tune the thresholds to your workload if you need something stricter.
*/
export async function getQueueHealth(
    q: Queue,
    thresholds = { backlog: 500, failed: 25 },
): Promise<QueueHealth> {

    const jobCounts = await q.getJobCounts(
        "waiting",
        "delayed",
        "active",
        "failed",
        "completed",
        "paused",
    );

    const backlog = jobCounts.waiting + jobCounts.delayed;
    const failed = jobCounts.failed;

    let status = QueueStatus.Healthy;
    if (failed > thresholds.failed || backlog > thresholds.backlog * 2) {
        status = QueueStatus.Down;
    } else if (failed > 0 || backlog > thresholds.backlog) {
        status = QueueStatus.Degraded;
    }

    // grab a *small* sample of the oldest active jobs (cheap, bounded query)
    const activeJobsRaw = await q.getJobs(["active"], 0, SAMPLE_ACTIVE_JOB_END, true);
    const activeJobs: ActiveJobInfo[] = activeJobsRaw.map(j => ({
        id: j.id as string,
        name: j.name,
        duration: (Date.now() - (j.processedOn ?? Date.now())),
        processedOn: j.processedOn,
    }));

    return { status, jobCounts, activeJobs };
}

// Production-ready connection configuration
export function getRedisConnectionConfig(): RedisOptions {
    const url = process.env.REDIS_URL;
    if (url) {
        // Parse Redis URL properly
        const urlObj = new URL(url);
        const options: RedisOptions = {
            host: urlObj.hostname,
            port: parseInt(urlObj.port) || 6379,
            maxRetriesPerRequest: null, // BullMQ requires this to be null
            enableReadyCheck: true,
            lazyConnect: false,
        };

        if (urlObj.password) {
            options.password = urlObj.password;
        }

        if (urlObj.pathname && urlObj.pathname.length > 1) {
            options.db = parseInt(urlObj.pathname.slice(1)) || 0;
        }

        // Add test-specific settings
        if (process.env.NODE_ENV === "test") {
            options.connectTimeout = 5000;
            options.lazyConnect = true;
            options.retryStrategy = (times) => {
                if (times > 2) return null; // Stop retrying after 2 attempts in tests
                return times * 100;
            };
            // Enable offline queue for BullMQ compatibility in tests
            options.enableOfflineQueue = true;
            options.reconnectOnError = () => false; // Don't reconnect on errors in tests
        } else {
            // Production settings with improved connection handling
            options.connectTimeout = 30000; // Increase timeout for initial connection (30s for startup)
            options.commandTimeout = 15000; // Increase command timeout to 15s for heavy operations
            options.enableOfflineQueue = true; // Queue commands when disconnected
            
            options.reconnectOnError = (err) => {
                logger.error("Redis reconnect error", { error: serializeError(err) });
                return true; // Always try to reconnect in production
            };
            
            options.retryStrategy = (times) => {
                // Use exponential backoff with jitter to prevent thundering herd
                const baseDelay = Math.min(times * 100, 3000);
                const jitter = Math.random() * 100; // Add 0-100ms jitter
                const delay = baseDelay + jitter;
                
                const stack = new Error().stack;
                const callingFunction = stack?.split("\n")[2]?.trim().replace(/^at\s+/, "") || "unknown";

                logger.warn(`Redis retry attempt ${times}, waiting ${Math.round(delay)}ms`, {
                    attempt: times,
                    delayMs: Math.round(delay),
                    operation: "queue_connection",
                    component: "BullMQ",
                    caller: callingFunction,
                    stackTrace: process.env.NODE_ENV === "development" ? stack : undefined,
                    redisUrl: process.env.REDIS_URL ? "configured" : "not_configured",
                    timestamp: new Date().toISOString(),
                });
                
                // Give up after 10 attempts
                if (times > 10) {
                    logger.error("Redis connection failed after 10 attempts", {
                        operation: "queue_connection",
                        component: "BullMQ",
                    });
                    return null;
                }
                
                return delay;
            };
        }

        return options;
    }

    return {
        host: process.env.REDIS_HOST || "localhost",
        port: parseInt(process.env.REDIS_PORT || "6379"),
        password: process.env.REDIS_PASSWORD,
        db: parseInt(process.env.REDIS_DB || "0"),
        maxRetriesPerRequest: null, // BullMQ requires this to be null
        enableReadyCheck: true,
        lazyConnect: process.env.NODE_ENV === "test",
        connectTimeout: process.env.NODE_ENV === "test" ? 5000 : 30000,
        commandTimeout: process.env.NODE_ENV === "test" ? 5000 : 15000,
        enableOfflineQueue: true,
    };
}

// Convert Redis options to BullMQ connection options
export function toBullMQConnectionOptions(redisOptions: RedisOptions): ConnectionOptions {
    return {
        ...redisOptions,
        // BullMQ-specific options - make sure these are preserved
        maxRetriesPerRequest: null, // BullMQ requires this to be null
        enableReadyCheck: redisOptions.enableReadyCheck ?? true,
        lazyConnect: redisOptions.lazyConnect ?? false,
    } as ConnectionOptions;
}

export interface BaseQueueConfig<Data> {
    name: string;
    processor: (job: Job<Data>) => Promise<unknown>;
    jobOpts?: Partial<JobsOptions>;
    workerOpts?: Partial<WorkerOptions>;
    onReady?(): Promise<void> | void;
    validator?: (data: Data) => { valid: boolean; errors?: string[] };
    // Optional: Override connection config for specific queues
    connectionOptions?: RedisOptions;
}

/**
 * ManagedQueue implementation with connection isolation
 * Each BullMQ component gets its own Redis connection for isolation
 */
export class ManagedQueue<Data extends BaseTaskData | Record<string, unknown> = BaseTaskData> {
    public readonly queue: Queue;
    public readonly worker: Worker<Data>;
    public readonly events: QueueEvents;
    public readonly ready: Promise<void>;

    private readonly queueName: string;
    private readonly validator?: (data: Data) => { valid: boolean; errors?: string[] };
    private readonly connectionMonitor?: NodeJS.Timer;
    private isClosing = false;
    private readonly connectionInfo: {
        queueConnection?: Redis | Cluster;
        workerConnection?: Redis | Cluster;
        eventsConnection?: Redis | Cluster;
    } = {};

    constructor(cfg: BaseQueueConfig<Data>) {
        this.queueName = cfg.name;
        this.validator = cfg.validator;

        // Get connection configuration (not an instance!)
        const redisOptions = cfg.connectionOptions || getRedisConnectionConfig();

        // Create separate configuration objects for each BullMQ component
        // This ensures BullMQ doesn't try to share connections between components
        const queueConnectionOptions = toBullMQConnectionOptions({
            ...redisOptions,
            // Add unique identifier to prevent connection pooling
            connectionName: `queue_${cfg.name}_${Date.now()}_${Math.random()}`,
        });
        const workerConnectionOptions = toBullMQConnectionOptions({
            ...redisOptions,
            // Add unique identifier to prevent connection pooling
            connectionName: `worker_${cfg.name}_${Date.now()}_${Math.random()}`,
        });
        const eventsConnectionOptions = toBullMQConnectionOptions({
            ...redisOptions,
            // Add unique identifier to prevent connection pooling
            connectionName: `events_${cfg.name}_${Date.now()}_${Math.random()}`,
        });

        // Each component creates its own connection
        this.queue = new Queue(cfg.name, {
            connection: queueConnectionOptions,
            defaultJobOptions: { ...DEFAULT_JOB_OPTIONS, ...cfg.jobOpts },
        });

        // Increase max listeners to prevent memory leak warnings in tests
        this.queue.setMaxListeners(100);

        this.worker = new Worker<Data>(
            cfg.name,
            cfg.processor,
            {
                ...BASE_WORKER_OPTS,
                ...cfg.workerOpts,
                connection: workerConnectionOptions,
            },
        );

        this.worker.setMaxListeners(100);

        this.events = new QueueEvents(cfg.name, {
            connection: eventsConnectionOptions,
        });

        this.events.setMaxListeners(100);

        // Set up comprehensive error handling
        this.setupErrorHandlers();

        // Store connection references for monitoring
        this.storeConnectionReferences();

        // Monitor connection health (disabled in tests to reduce noise)
        const isTest = process.env.NODE_ENV === "test";
        if (!isTest) {
            this.connectionMonitor = setInterval(() => {
                this.checkConnectionHealth();
            }, 30000); // Every 30 seconds
        }

        // Handle onReady
        this.ready = this.setupOnReady(cfg.onReady);
    }

    private setupErrorHandlers(): void {
        // Worker errors
        this.worker.on("error", (error) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;

            logger.error(`Worker error in queue ${this.queueName}`, {
                error: serializeError(error),
                queue: this.queueName,
                component: "worker",
                timestamp: new Date().toISOString(),
            });
        });

        // Queue events errors
        this.events.on("error", (error) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;

            logger.error(`QueueEvents error in queue ${this.queueName}`, {
                error: serializeError(error),
                queue: this.queueName,
                component: "events",
                timestamp: new Date().toISOString(),
            });
        });

        // Job failures
        this.events.on("failed", ({ jobId, failedReason, prev }) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;

            logger.error(`Job failed in queue ${this.queueName}`, {
                jobId,
                failedReason: typeof failedReason === "string" ? failedReason : serializeError(failedReason),
                previousState: prev,
                queue: this.queueName,
                timestamp: new Date().toISOString(),
            });
        });

        // Stalled jobs (important for production monitoring)
        this.events.on("stalled", ({ jobId }) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;

            logger.warn(`Job stalled in queue ${this.queueName}`, {
                jobId,
                queue: this.queueName,
                timestamp: new Date().toISOString(),
            });
        });
    }

    private async setupOnReady(onReady?: () => Promise<void> | void): Promise<void> {
        if (!onReady) return;

        return new Promise((resolve, reject) => {
            this.worker.once("ready", async () => {
                try {
                    await Promise.resolve(onReady());
                    resolve();
                } catch (error) {
                    logger.error(`onReady hook failed for queue ${this.queueName}`, { error: serializeError(error) });
                    reject(error);
                }
            });
        });
    }

    private async storeConnectionReferences(): Promise<void> {
        // Store connection references for better monitoring and cleanup
        try {
            this.connectionInfo.queueConnection = await this.queue.client;
            this.connectionInfo.workerConnection = await this.worker.client;
            this.connectionInfo.eventsConnection = await this.events.client;
        } catch (error) {
            logger.debug(`Error storing connection references for ${this.queueName}:`, error);
        }
    }

    private async checkConnectionHealth(): Promise<void> {
        // Skip health check if we're closing
        if (this.isClosing) return;

        try {
            // Check queue health
            const queueClient = await this.queue.client;
            if (queueClient.status !== "ready") {
                const isTest = process.env.NODE_ENV === "test";
                if (!isTest) {
                    logger.warn(`Queue Redis connection not ready for ${this.queueName}`, {
                        status: queueClient.status,
                        queue: this.queueName,
                    });
                }
            }

            // Get queue metrics for monitoring
            const jobCounts = await this.queue.getJobCounts();
            const workerInfo = await this.worker.client.then(c => c.ping());

            logger.debug(`Queue ${this.queueName} health check`, {
                queue: this.queueName,
                jobCounts,
                workerPing: workerInfo,
                connections: {
                    queue: queueClient.status,
                    // Note: worker and events clients are internal to BullMQ
                },
            });
        } catch (error) {
            const isTest = process.env.NODE_ENV === "test";
            if (!isTest) {
                logger.error(`Health check failed for queue ${this.queueName}`, { error: serializeError(error) });
            }
        }
    }

    /**
     * Add a job to the queue (compatibility method).
     */
    add(data: Data, opts: Partial<JobsOptions> = {}) {
        return this.queue.add(this.queueName, data, opts);
    }

    /**
     * Helper function to add a task job to a queue with standardized error handling.
     * Compatible with ManagedQueue interface.
     */
    async addTask<T extends Data>(
        data: T,
        opts: Partial<JobsOptions> = {},
    ): Promise<{ __typename: "Success"; success: boolean; data?: { id: string } }> {
        try {
            // Validate if validator provided
            if (this.validator) {
                const validation = this.validator(data);
                if (!validation.valid) {
                    logger.error(`Task validation failed for ${this.queueName}`, {
                        errors: validation.errors,
                        data,
                    });
                    return { __typename: "Success" as const, success: false };
                }
            }

            // Use the task ID as the job ID if available
            const jobOpts: Partial<JobsOptions> = { ...opts };
            if (jobOpts.jobId == null && "id" in data && typeof data.id === "string") {
                jobOpts.jobId = data.id;
            }

            const job = await this.add(data, jobOpts);
            return {
                __typename: "Success" as const,
                success: true,
                data: { id: job.id as string },
            };

        } catch (error) {
            logger.error("Error adding task to queue", {
                queueName: this.queueName,
                error: serializeError(error),
                data,
            });
            return { __typename: "Success" as const, success: false };
        }
    }

    async close(): Promise<void> {
        // Prevent multiple close attempts
        if (this.isClosing) {
            return;
        }
        this.isClosing = true;

        // Clear health monitor first
        if (this.connectionMonitor) {
            clearInterval(this.connectionMonitor);
        }

        try {
            // 1. Pause the queue to stop accepting new jobs
            await this.queue.pause().catch(() => {
                // Ignore pause errors - queue might already be paused
            });

            // 2. Wait for any active jobs to complete (with timeout)
            await this.waitForActiveJobs();

            // 3. Remove event listeners to stop processing events
            this.worker.removeAllListeners();
            this.events.removeAllListeners();
            this.queue.removeAllListeners();

            // 4. Add error handlers to catch any errors during close
            this.worker.on("error", () => {
                // Silently ignore errors during close
            });
            this.events.on("error", () => {
                // Silently ignore errors during close
            });
            (this.queue as any).on("error", () => {
                // Silently ignore errors during close
            });

            // 5. Close components in reverse dependency order
            const closePromises = [
                this.worker.close().catch(error => {
                    logger.debug(`Worker close error for ${this.queueName}:`, error);
                }),
                this.events.close().catch(error => {
                    logger.debug(`Events close error for ${this.queueName}:`, error);
                }),
                this.queue.close().catch(error => {
                    logger.debug(`Queue close error for ${this.queueName}:`, error);
                }),
            ];

            // Wait for all components to close
            await Promise.allSettled(closePromises);

            // 6. Give connections time to close gracefully
            await new Promise(resolve => setTimeout(resolve, 100));

            // 7. Force close any remaining connections
            await this.closeIndividualConnections();

            // 8. Final delay to ensure all async operations complete
            if (process.env.NODE_ENV === "test") {
                await new Promise(resolve => setTimeout(resolve, 200));
            }

        } catch (error) {
            logger.error(`Error during graceful close of queue ${this.queueName}`, { error: serializeError(error) });
        }
    }

    private async waitForActiveJobs(): Promise<void> {
        try {
            const jobCounts = await this.queue.getJobCounts();
            if (jobCounts.active > 0) {
                logger.debug(`Waiting for ${jobCounts.active} active jobs to complete in ${this.queueName}`);
                // Wait up to 5 seconds for active jobs to complete
                const maxWait = 5000;
                const checkInterval = 100;
                let waited = 0;

                while (waited < maxWait) {
                    const currentCounts = await this.queue.getJobCounts();
                    if (currentCounts.active === 0) {
                        break;
                    }
                    await new Promise(resolve => setTimeout(resolve, checkInterval));
                    waited += checkInterval;
                }
            }
        } catch (error) {
            // Ignore errors when checking job counts
            logger.debug(`Error checking job counts during close for ${this.queueName}:`, error);
        }
    }

    private async closeIndividualConnections(): Promise<void> {
        const connections = [
            { name: "queue", client: this.connectionInfo.queueConnection },
            { name: "worker", client: this.connectionInfo.workerConnection },
            { name: "events", client: this.connectionInfo.eventsConnection },
        ];

        const closePromises = connections.map(async ({ name, client }) => {
            if (!client) return;

            // Check if connection is already closed
            if (client.status === "end" || client.status === "close") {
                logger.debug(`${this.queueName}: ${name} connection already closed`);
                return;
            }

            try {
                // Remove all listeners to prevent unhandled events
                client.removeAllListeners();

                // Add comprehensive error handlers for the close process
                client.on("error", (err) => {
                    // Silently ignore connection errors during close
                    logger.debug(`${this.queueName}: ${name} connection error during close (ignored):`, err.message);
                });

                client.on("close", () => {
                    logger.debug(`${this.queueName}: ${name} connection closed event`);
                });

                client.on("end", () => {
                    logger.debug(`${this.queueName}: ${name} connection end event`);
                });

                // Attempt graceful close first
                if (client.status === "ready") {
                    logger.debug(`${this.queueName}: Gracefully closing ${name} connection`);
                    await Promise.race([
                        client.quit(),
                        new Promise((_, reject) =>
                            setTimeout(() => reject(new Error("Quit timeout")), 2000),
                        ),
                    ]);
                } else {
                    logger.debug(`${this.queueName}: Force disconnecting ${name} connection (status: ${client.status})`);
                    client.disconnect();
                }

                logger.debug(`${this.queueName}: ${name} connection closed successfully`);
            } catch (error) {
                logger.debug(`${this.queueName}: Error closing ${name} connection:`, error);
                // Force disconnect if quit fails
                try {
                    client.disconnect();
                    logger.debug(`${this.queueName}: Force disconnected ${name} connection`);
                } catch (e) {
                    logger.debug(`${this.queueName}: Final disconnect error for ${name} (ignored):`, e);
                }
            }
        });

        // Wait for all connections to close
        await Promise.allSettled(closePromises);

        // Clear connection references
        this.connectionInfo.queueConnection = undefined;
        this.connectionInfo.workerConnection = undefined;
        this.connectionInfo.eventsConnection = undefined;
    }

    /**
     * Get the statuses of multiple tasks from this queue.
     * Compatible with ManagedQueue interface.
     */
    async getTaskStatuses(taskIds: string[]): Promise<Array<{ id: string, status: string | null }>> {
        return Promise.all(taskIds.map(async (taskId) => {
            try {
                const job = await this.queue.getJob(taskId);
                if (job) {
                    const state = await job.getState();
                    return {
                        id: taskId,
                        status: (job.data && typeof job.data === "object" && "status" in job.data ? String(job.data.status) : null) ?? state ?? null,
                    };
                } else {
                    return { id: taskId, status: null };
                }
            } catch (error) {
                logger.error(`Failed to retrieve task ${taskId}`, { error: serializeError(error) });
                return { id: taskId, status: null };
            }
        }));
    }

    /**
     * Update a task's status with authentication check.
     * Compatible with ManagedQueue interface.
     */
    async changeTaskStatus(
        taskId: string,
        status: string,
        userId: string,
    ): Promise<{ __typename: "Success"; success: boolean }> {
        try {
            const normalizedStatus = status.trim().toLowerCase();
            const job = await this.queue.getJob(taskId);

            if (!job) {
                const trulyTerminalStates = ["completed", "failed"];
                if (trulyTerminalStates.includes(normalizedStatus)) {
                    logger.info(`Task with id ${taskId} not found, but considered a success as status is being changed to a terminal state '${normalizedStatus}'.`);
                    return { __typename: "Success" as const, success: true };
                }
                logger.error(`Task with id ${taskId} not found. Cannot change status to '${normalizedStatus}'.`);
                return { __typename: "Success" as const, success: false };
            }

            const data = job.data as Data;

            // Check ownership
            const ownerId = extractOwnerId(data);
            if (!ownerId) {
                logger.error(`Task ${taskId} does not have an owner and cannot have its status updated`);
                return { __typename: "Success" as const, success: false };
            }
            if (ownerId !== userId) {
                logger.error(`User ${userId} not authorized to change status of task ${taskId} (owner=${ownerId}).`);
                return { __typename: "Success" as const, success: false };
            }

            // Update the job data with the new status
            await job.update({
                ...data,
                status: normalizedStatus,
            });

            return { __typename: "Success" as const, success: true };
        } catch (error) {
            logger.error(`Failed to change status for task ${taskId}`, { error: serializeError(error) });
            return { __typename: "Success" as const, success: false };
        }
    }

    /**
     * Check the health of the queue.
     * Compatible with ManagedQueue interface.
     */
    async checkHealth() {
        try {
            const jobCounts = await this.queue.getJobCounts(
                "waiting",
                "delayed",
                "active",
                "failed",
                "completed",
                "paused",
            );

            const backlog = jobCounts.waiting + jobCounts.delayed;
            const failed = jobCounts.failed;

            let status = "healthy";
            if (failed > 25 || backlog > 1000) {
                status = "down";
            } else if (failed > 0 || backlog > 500) {
                status = "degraded";
            }

            const activeJobsRaw = await this.queue.getJobs(["active"], 0, 9, true);
            const activeJobs = activeJobsRaw.map(j => ({
                id: j.id as string,
                name: j.name,
                duration: (Date.now() - (j.processedOn ?? Date.now())),
                processedOn: j.processedOn,
            }));

            return { status, jobCounts, activeJobs };
        } catch (error) {
            logger.error(`Health check failed for queue ${this.queueName}`, { error });
            return {
                status: "down",
                jobCounts: {},
                activeJobs: [],
            };
        }
    }

    // Production monitoring methods
    async getMetrics() {
        const [jobCounts, workers] = await Promise.all([
            this.queue.getJobCounts(),
            this.queue.getWorkers(),
        ]);

        return {
            queue: this.queueName,
            jobs: jobCounts,
            workers: workers.length,
            isPaused: await this.queue.isPaused(),
        };
    }

    /**
     * Get the owner ID from task data (supports multiple formats)
     * @param data Task data
     * @returns Owner ID or null if not found
     */
    static getTaskOwner(data: unknown): string | null {
        return extractOwnerId(data);
    }
}

// Test utility connection cache for backward compatibility
const testRedisClients: Record<string, Redis> = {};
const testConnectionPromises: Record<string, Promise<Redis>> = {};

/**
 * Test utility function to build Redis connections with caching.
 * This provides backward compatibility for tests that expect cached connections.
 * 
 * @internal Test utility function
 */
export async function buildRedis(url: string): Promise<Redis> {
    if (process.env.NODE_ENV !== "test") {
        throw new Error("buildRedis is only available in test environment");
    }

    // Check if we already have a ready connection
    if (testRedisClients[url]) {
        const client = testRedisClients[url];
        if (client.status === "ready") {
            return client;
        } else if (client.status === "end" || client.status === "close") {
            // Connection is stale, remove it and create a new one
            delete testRedisClients[url];
            delete testConnectionPromises[url];
        }
    }

    // Check if we're already connecting to this URL
    if (url in testConnectionPromises) {
        return testConnectionPromises[url];
    }

    // Create new connection
    const connectionPromise = (async () => {
        try {
            const options = getRedisConnectionConfig();
            const client = new Redis(url, {
                ...options,
                // Override with test-specific settings
                connectTimeout: 5000,
                lazyConnect: true,
                maxRetriesPerRequest: 1,
                retryStrategy: (times) => {
                    if (times > 1) return null;
                    return 100;
                },
                enableOfflineQueue: true, // Keep enabled for compatibility
                reconnectOnError: () => false,
            });

            // Set up error handling
            client.on("error", (err) => {
                // Only log non-timeout errors in tests
                if ((err as any).code !== "ETIMEDOUT") {
                    logger.debug("Test Redis client error:", err.message);
                }
                // Mark connection as failed and clean up
                delete testRedisClients[url];
                delete testConnectionPromises[url];
            });

            // Handle disconnection events
            client.on("close", () => {
                delete testRedisClients[url];
                delete testConnectionPromises[url];
            });

            client.on("end", () => {
                delete testRedisClients[url];
                delete testConnectionPromises[url];
            });

            // Connect explicitly since we're using lazyConnect
            await client.connect();

            // Store the connection
            testRedisClients[url] = client;
            return client;
        } catch (error) {
            // Clean up failed connection attempt
            delete testConnectionPromises[url];
            if (testRedisClients[url]) {
                delete testRedisClients[url];
            }
            throw error;
        }
    })();

    testConnectionPromises[url] = connectionPromise;
    return connectionPromise;
}

/**
 * Close all test Redis connections.
 * 
 * @internal Test utility function
 */
export async function closeRedisConnections(): Promise<void> {
    if (process.env.NODE_ENV !== "test") {
        return;
    }

    // Wait for any pending connections to complete
    const pendingConnections = Object.values(testConnectionPromises);
    if (pendingConnections.length > 0) {
        try {
            await Promise.allSettled(pendingConnections);
        } catch (error) {
            logger.debug("Error waiting for pending connections:", error);
        }
    }

    // Close all existing connections
    const closePromises = Object.entries(testRedisClients).map(async ([url, client]) => {
        try {
            if (client && client.status !== "end" && client.status !== "close") {
                client.removeAllListeners();
                // Add error handler to prevent unhandled rejections
                client.on("error", () => {
                    // Silently ignore errors during close
                });
                if (client.status === "ready") {
                    await client.quit();
                } else {
                    client.disconnect();
                }
            }
        } catch (error) {
            logger.debug(`Error closing Redis connection for ${url}:`, error);
            // Force disconnect if quit fails
            try {
                client.disconnect();
            } catch (e) {
                // Ignore final disconnect errors
            }
        }
    });

    await Promise.allSettled(closePromises);

    // Clear the caches
    Object.keys(testRedisClients).forEach(key => delete testRedisClients[key]);
    Object.keys(testConnectionPromises).forEach(key => delete testConnectionPromises[key]);
}

/**
 * Clears any cached Redis connections for test cleanup.
 * 
 * In the new isolated connection architecture, this function primarily
 * serves as a compatibility layer for existing test code.
 * 
 * @internal Test utility function
 */
export function clearRedisCache(): void {
    if (process.env.NODE_ENV === "test") {
        logger.debug("clearRedisCache called - clearing test connection cache");

        // Synchronously clear the caches (connections should be closed first)
        Object.keys(testRedisClients).forEach(key => delete testRedisClients[key]);
        Object.keys(testConnectionPromises).forEach(key => delete testConnectionPromises[key]);

        // Force garbage collection to clean up any lingering connection references
        if (global.gc) {
            global.gc();
        }
    }
}

// Expose the test cache for compatibility with existing tests
if (process.env.NODE_ENV === "test") {
    (buildRedis as any).redisClients = testRedisClients;
    (buildRedis as any).connectionEstablishmentPromises = testConnectionPromises;
}

// Singleton for connection monitoring across all queues
export class QueueConnectionMonitor {
    private static instance: QueueConnectionMonitor;
    private queues: Map<string, ManagedQueue<any>> = new Map();

    static getInstance(): QueueConnectionMonitor {
        if (!this.instance) {
            this.instance = new QueueConnectionMonitor();
        }
        return this.instance;
    }

    registerQueue(queue: ManagedQueue<any>): void {
        this.queues.set(queue.queue.name, queue);
    }

    async getSystemMetrics() {
        const metrics = await Promise.all(
            Array.from(this.queues.values()).map(q => q.getMetrics()),
        );

        return {
            totalQueues: this.queues.size,
            queues: metrics,
            timestamp: new Date(),
        };
    }
}
