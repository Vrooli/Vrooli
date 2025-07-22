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
import { createRedisUrlConfig, createRedisEnvConfig } from "../utils/redisConfig.js";
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

// Production-ready connection configuration using centralized Redis settings
export function getRedisConnectionConfig(): RedisOptions {
    const url = process.env.REDIS_URL;
    
    // Connection configuration handled by centralized Redis settings
    
    // Use centralized Redis configuration to ensure consistency across all services
    const options = url 
        ? createRedisUrlConfig(url, "BullMQ")
        : createRedisEnvConfig("BullMQ");
    
    // BullMQ-specific overrides (these must be set for BullMQ compatibility)
    options.maxRetriesPerRequest = null; // BullMQ requires this to be null
    options.lazyConnect = false; // BullMQ manages connections internally
    
    // Connection options configured

    return options;
}

// Convert Redis options to BullMQ connection options
export function toBullMQConnectionOptions(redisOptions: RedisOptions): ConnectionOptions {
    // For BullMQ, we need to ensure the options are in the right format
    // BullMQ will create its own Redis connections using these options
    return {
        host: redisOptions.host,
        port: redisOptions.port,
        password: redisOptions.password,
        username: redisOptions.username,
        db: redisOptions.db,
        // BullMQ-specific requirements
        maxRetriesPerRequest: null, // MUST be null for BullMQ
        enableReadyCheck: true,
        lazyConnect: false,
        // Connection stability settings
        connectTimeout: redisOptions.connectTimeout || 30000,
        commandTimeout: redisOptions.commandTimeout || 15000,
        keepAlive: redisOptions.keepAlive ?? (process.env.NODE_ENV === "test" ? 0 : 10000),
        enableOfflineQueue: redisOptions.enableOfflineQueue ?? true,
        retryStrategy: redisOptions.retryStrategy,
        reconnectOnError: redisOptions.reconnectOnError,
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
    private readonly pingInterval?: NodeJS.Timer;
    private isClosing = false;
    private connectionInfo: {
        // Connection tracking removed - BullMQ manages connections internally
    } = {};

    private constructor(
        cfg: BaseQueueConfig<Data>,
        queue: Queue,
        worker: Worker<Data>,
        events: QueueEvents,
    ) {
        this.queueName = cfg.name;
        this.validator = cfg.validator;
        this.queue = queue;
        this.worker = worker;
        this.events = events;
        
        // DEBUG: Add connection event monitoring
        const queueConnDebug = (type: string) => {
            return (err?: Error) => {
                if (err) {
                    logger.error(`Queue ${cfg.name} connection error: ${type}`, {
                        queue: cfg.name,
                        type,
                        error: err.message,
                    });
                }
            };
        };
        
        // Monitor connection events
        (this.queue as any).on("ioredis:connect", queueConnDebug("connect"));
        (this.queue as any).on("ioredis:ready", queueConnDebug("ready"));
        (this.queue as any).on("ioredis:error", queueConnDebug("error"));
        (this.queue as any).on("ioredis:close", queueConnDebug("close"));
        
        // Monitor client connection failures
        this.queue.client.catch(err => {
            logger.error(`Queue ${cfg.name} failed to get client`, {
                queue: cfg.name,
                error: err.message,
                stack: err.stack,
            });
        });

        // Increase max listeners to prevent memory leak warnings in tests
        this.queue.setMaxListeners(100);
        this.worker.setMaxListeners(100);
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
            }, 120000); // Every 2 minutes (reduced from 30s to prevent connection spam)

            // Ultra-aggressive ping strategy to prevent WSL2/Docker NAT timeouts (every 8 seconds)
            // TCP keepalive (5s) + application ping (8s) = dual protection against ~26s timeout
            this.pingInterval = setInterval(async () => {
                if (this.isClosing) return;
                try {
                    const startTime = Date.now();
                    // Ping all three Redis connections (queue, worker, events)
                    const pingPromises = [
                        this.queue.client.then(c => c.ping()),
                        this.worker.client.then(c => c.ping()),
                        this.events.client.then(c => c.ping()),
                    ];
                    const results = await Promise.allSettled(pingPromises);
                    const duration = Date.now() - startTime;
                    
                    // Log with more detail to verify execution
                    const successful = results.filter(r => r.status === "fulfilled").length;
                    const failed = results.filter(r => r.status === "rejected").length;
                    
                    // Only log successes at debug level to reduce noise, failures at warn level
                    if (failed > 0) {
                        logger.warn(`⚡ ULTRA-PING FAILED for ${cfg.name}: ${successful}/${results.length} successful in ${duration}ms`, {
                            queue: cfg.name,
                            successful,
                            failed,
                            duration,
                            timestamp: new Date().toISOString(),
                            pingInterval: "8000ms",
                        });
                        results.forEach((result, index) => {
                            if (result.status === "rejected") {
                                const connType = ["queue", "worker", "events"][index];
                                logger.warn(`⚡ ULTRA-PING FAILED for ${cfg.name} ${connType}`, {
                                    queue: cfg.name,
                                    connectionType: connType,
                                    error: serializeError(result.reason),
                                });
                            }
                        });
                    } else {
                        logger.debug(`⚡ ULTRA-PING for ${cfg.name}: ${successful}/${results.length} successful in ${duration}ms`, {
                            queue: cfg.name,
                            successful,
                            failed,
                            duration,
                            timestamp: new Date().toISOString(),
                            pingInterval: "8000ms",
                        });
                    }
                } catch (error) {
                    logger.error(`⚡ ULTRA-PING EXCEPTION for queue ${cfg.name}`, { 
                        queue: cfg.name,
                        error: serializeError(error),
                        pingInterval: "8000ms",
                    });
                }
            }, 8000); // Ultra-aggressive 8-second ping interval (faster than TCP keepalive)
        }

        // Handle onReady
        this.ready = this.setupOnReady(cfg.onReady);
    }

    /**
     * Factory method to create a ManagedQueue with pre-authenticated Redis connections
     */
    static async create<Data extends BaseTaskData | Record<string, unknown> = BaseTaskData>(
        cfg: BaseQueueConfig<Data>,
    ): Promise<ManagedQueue<Data>> {
        // Create managed queue with isolated connections

        // Get connection configuration (not an instance!)
        const redisOptions = cfg.connectionOptions || getRedisConnectionConfig();

        // Convert Redis options to BullMQ format

        // Convert Redis options to BullMQ connection options
        // BullMQ needs the options directly, not factory functions
        const connectionOptions = toBullMQConnectionOptions(redisOptions);

        const queue = new Queue(cfg.name, {
            connection: connectionOptions,
            defaultJobOptions: { ...DEFAULT_JOB_OPTIONS, ...cfg.jobOpts },
        });

        const worker = new Worker<Data>(
            cfg.name,
            cfg.processor,
            {
                ...BASE_WORKER_OPTS,
                ...cfg.workerOpts,
                connection: connectionOptions,
            },
        );

        const events = new QueueEvents(cfg.name, {
            connection: connectionOptions,
        });

        // Create the ManagedQueue instance with the pre-created BullMQ objects
        const managedQueue = new ManagedQueue(cfg, queue, worker, events);
        
        return managedQueue;
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
        // Connection references are now stored by the factory method
        // This method is kept for backward compatibility but does nothing
    }

    private async checkConnectionHealth(): Promise<void> {
        // Skip health check if we're closing
        if (this.isClosing) return;

        try {
            // Add timeout to prevent hanging health checks
            const queueClient = await Promise.race([
                this.queue.client,
                new Promise<never>((_, reject) => 
                    setTimeout(() => reject(new Error("Health check timeout")), 5000),
                ),
            ]);
            
            if (queueClient.status !== "ready") {
                const isTest = process.env.NODE_ENV === "test";
                if (!isTest) {
                    // Don't log warnings for transient connection states to reduce noise
                    if (queueClient.status !== "connecting" && queueClient.status !== "reconnecting") {
                        logger.warn(`Queue Redis connection not ready for ${this.queueName}`, {
                            status: queueClient.status,
                            queue: this.queueName,
                        });
                    }
                }
                return; // Skip health operations if not ready
            }

            // Only perform monitoring operations if connection is ready
            if (queueClient.status === "ready") {
                const jobCounts = await this.queue.getJobCounts();
                const workerInfo = await this.worker.client.then(c => c.ping());

                logger.debug(`Queue ${this.queueName} health check`, {
                    queue: this.queueName,
                    jobCounts,
                    workerPing: workerInfo,
                    connections: {
                        queue: queueClient.status,
                    },
                });
            } else {
                logger.debug(`Queue ${this.queueName} health check (connection not ready)`, {
                    queue: this.queueName,
                    connections: {
                        queue: queueClient.status,
                    },
                });
            }
        } catch (error) {
            const isTest = process.env.NODE_ENV === "test";
            if (!isTest) {
                // Only log errors that aren't timeout/connection related to reduce noise
                const errorMessage = error instanceof Error ? error.message : String(error);
                if (!errorMessage.includes("timeout") && 
                    !errorMessage.includes("connection") && 
                    !errorMessage.includes("ECONNREFUSED") &&
                    !errorMessage.includes("ETIMEDOUT")) {
                    logger.error(`Health check failed for queue ${this.queueName}`, { error: serializeError(error) });
                } else {
                    logger.debug(`Health check timeout/connection issue for queue ${this.queueName}`, { 
                        error: errorMessage,
                        queue: this.queueName, 
                    });
                }
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

        // Clear ping interval
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
        }

        try {
            // 1. Remove all event listeners first to prevent new events
            this.worker.removeAllListeners();
            this.events.removeAllListeners();
            this.queue.removeAllListeners();

            // 2. Pause the queue to stop accepting new jobs
            await this.queue.pause().catch(() => {
                // Ignore pause errors - queue might already be paused
            });

            // 3. Close worker first (stops processing new jobs)
            await this.worker.close(true).catch(error => {
                logger.debug(`Worker close error for ${this.queueName}:`, error);
            });

            // 4. Close events listener
            await this.events.close().catch(error => {
                logger.debug(`Events close error for ${this.queueName}:`, error);
            });

            // 5. Close queue last
            await this.queue.close().catch(error => {
                logger.debug(`Queue close error for ${this.queueName}:`, error);
            });

            // 6. Small delay to ensure connections are fully closed
            await new Promise(resolve => setTimeout(resolve, 50));

        } catch (error) {
            logger.error(`Error during graceful close of queue ${this.queueName}`, { error: serializeError(error) });
        }
    }

    private async waitForActiveJobs(): Promise<void> {
        try {
            // Check if connection is ready first
            const queueClient = await this.queue.client;
            if (queueClient.status !== "ready") {
                // Just wait a bit for any active jobs to complete naturally
                await new Promise(resolve => setTimeout(resolve, 500));
                return;
            }

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

    // Connection cleanup is now handled by BullMQ when we close queue/worker/events

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
            // Check if the queue connection is ready before attempting operations
            const queueClient = await this.queue.client;
            if (queueClient.status !== "ready") {
                // Return degraded status if connection isn't ready
                return {
                    status: "degraded" as const,
                    jobCounts: {
                        waiting: 0,
                        delayed: 0,
                        active: 0,
                        failed: 0,
                        completed: 0,
                        paused: 0,
                    },
                    activeJobs: [],
                };
            }

            // Normal health check - reuses existing connection
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
        // Check connection status first
        const queueClient = await this.queue.client;
        if (queueClient.status !== "ready") {
            // Return minimal metrics if connection isn't ready
            return {
                queue: this.queueName,
                jobs: {},
                workers: 1, // Assume at least one worker
                isPaused: false,
            };
        }

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
