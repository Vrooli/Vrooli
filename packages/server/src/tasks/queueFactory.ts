/*
 * queueFactory.ts
 *
 * This module provides a factory for creating and managing BullMQ queues, workers, and event listeners in a standardized, scalable way.
 *
 * Purpose:
 *   - Centralizes queue creation and configuration for the Vrooli backend.
 *   - Ensures consistent job options, worker options, and Redis connection reuse.
 *   - Supports horizontal scaling by allowing multiple Node.js processes to run workers for the same queue, leveraging Redis as a distributed backend.
 *   - Provides hooks for custom queue initialization logic (e.g., cron scheduling, state machines).
 *
 * Usage:
 *   - Import this module to define new queues for background processing tasks (e.g., email, agent runs).
 *   - Use the ManagedQueue class to instantiate a queue with business logic and configuration.
 *   - Add jobs to the queue using the `add` helper method, which abstracts away direct BullMQ usage.
 *   - Extend or override job and worker options as needed for specific queues.
 *
 * Horizontal Scaling:f
 *   - By using Redis as the central broker, multiple Node.js processes (across servers/containers) can run workers for the same queue.
 *   - Each worker instance will compete for jobs, enabling true distributed processing and high availability.
 *   - Graceful shutdown and event handling are built-in for robust operation in production environments.
 *
 * Best Practices:
 *   - Keep queue business logic (the processor) stateless and idempotent for safe retries and scaling.
 *   - Use the provided defaults for most queues, but override options for special cases (e.g., high-priority or long-running jobs).
 *   - Monitor queue events and failures using the built-in event listeners and logging.
 *
 * For more on documentation best practices, see:
 *   - https://guides.lib.berkeley.edu/how-to-write-good-documentation
 *   - https://github.com/resources/articles/software-development/tools-and-techniques-for-effective-code-documentation
 */
import { DAYS_1_S, MINUTES_1_MS, SECONDS_10_MS, type Success } from "@local/shared";
import { type Job, type JobsOptions, Queue, QueueEvents, Worker, type WorkerOptions } from "bullmq";
import IORedis from "ioredis";
import { logger } from "../events/logger.js";
import type { AnyTask, LLMCompletionTask, RunTask, SandboxTask } from "./taskTypes.js";

// Cache Redis connections by URL to reuse a single connection per process.
const redisClients: Record<string, IORedis> = {};
const connectionEstablishmentPromises: Record<string, Promise<IORedis>> = {};

/**
 * Creates and returns a shared Redis connection for BullMQ queues and workers.
 *
 * @param url - The Redis connection URL.
 * @returns An IORedis instance for use with BullMQ.
 *
 * @remarks
 *   - Reuses a single Redis connection per Node.js process to reduce overhead.
 *   - Pass this connection to all queues and workers in the process.
 *   - This function is async to properly handle cleanup of stale connections and prevent race conditions.
 */
export async function buildRedis(url: string): Promise<IORedis> {
    // Check 1: Existing ready client in the primary cache
    if (redisClients[url]?.status === "ready") {
        return redisClients[url];
    }

    // Check 2: Existing client in primary cache, but not ready (stale)
    // This might happen if a client disconnected and its 'end'/'close' handler hasn't removed it yet,
    // or if a previous attempt failed to become ready.
    if (redisClients[url]) {
        const staleClient = redisClients[url];
        logger.info(`Stale Redis client found for ${url} (status: ${staleClient.status}). Attempting to quit.`);
        try {
            await staleClient.quit();
        } catch (error) {
            logger.error(`Error quitting stale Redis client for ${url}. Proceeding to replace.`, { error });
        }
        delete redisClients[url]; // Remove from cache regardless of quit success
    }

    // Check 3: Is a connection already being established by another concurrent call?
    // If so, await that existing promise.
    if (connectionEstablishmentPromises[url] !== undefined) {
        logger.info(`Connection establishment already in progress for ${url}. Reusing promise.`);
        return connectionEstablishmentPromises[url];
    }

    // Check 4: No ready client and no connection establishment in progress. Create a new one.
    const newConnectionPromise = (async (): Promise<IORedis> => {
        logger.info(`Creating new Redis client for ${url}.`);
        const client = new IORedis(url, {
            maxRetriesPerRequest: null, // Keep retrying connection
            // Enable lazy connect to allow event listeners to be attached before connection.
            // This is the default, but explicit for clarity.
            lazyConnect: true,
        });

        client.on("error", (error) =>
            logger.error(`Redis client error for ${url}`, { error, clientStatus: client.status }),
        );

        client.on("end", () => {
            logger.info(`Redis connection ended for ${url}. Removing from primary cache.`);
            if (redisClients[url] === client) {
                delete redisClients[url];
            }
            // Note: No need to touch connectionEstablishmentPromises here as it's handled by the finally block
            // of the promise that creates the client.
        });

        client.on("close", () => {
            logger.info(`Redis connection closed for ${url}. Removing from primary cache.`);
            if (redisClients[url] === client) {
                delete redisClients[url];
            }
        });

        // Explicitly wait for the client to be 'ready'.
        // IORedis connects lazily, so we must wait for 'ready' before considering it usable.
        await new Promise<void>((resolve, reject) => {
            const READY_TIMEOUT_MS = 30000; // 30 seconds timeout for client readiness

            const timeoutId = setTimeout(() => {
                client.removeListener("ready", onReady); // Clean up listener
                // No explicit 'error' listener for rejection here as 'maxRetriesPerRequest: null'
                // means it will keep trying. The timeout is the main guard against indefinite hang.
                // If IORedis emits a fatal connection error, it might reject the .connect() promise below.
                reject(new Error(`Timeout (${READY_TIMEOUT_MS}ms) waiting for Redis client to be ready for ${url}`));
            }, READY_TIMEOUT_MS);

            function onReady() {
                clearTimeout(timeoutId);
                logger.info(`Redis client connected and ready for ${url}.`);
                resolve();
            }

            client.once("ready", onReady);

            // Start the connection attempt if lazyConnect is true (which it is by default).
            // If connect() throws an error (e.g. invalid URL), it will reject this outer promise.
            client.connect().catch(connectError => {
                clearTimeout(timeoutId);
                client.removeListener("ready", onReady);
                logger.error(`Redis client failed to connect for ${url}.`, { error: connectError });
                reject(connectError);
            });
        });

        // Once ready, store in the primary cache.
        redisClients[url] = client;
        return client;
    })();

    // Store the promise in the map so other concurrent calls can await it.
    connectionEstablishmentPromises[url] = newConnectionPromise;

    try {
        // Await the promise for this current call.
        const establishedClient = await newConnectionPromise;
        return establishedClient;
    } catch (error) {
        // If client creation/connection failed, the promise was rejected.
        logger.error(`Failed to establish Redis connection for ${url} during buildRedis.`, { error });
        // The 'finally' block will clean up connectionEstablishmentPromises[url].
        throw error; // Re-throw to the caller of buildRedis.
    } finally {
        // Once this specific attempt's promise is settled (resolved or rejected),
        // remove it from the establishment map. Subsequent calls will either:
        // 1. Find the client in redisClients (if successful).
        // 2. Or start a fresh connection attempt (if this one failed and was removed from redisClients by handlers).
        delete connectionEstablishmentPromises[url];
    }
}

/**
 * Default job options for all queues unless overridden.
 *
 * @remarks
 *   - Jobs are removed after completion or failure (with age and count limits).
 *   - Retries failed jobs up to 5 times with exponential backoff.
 *   - These defaults are suitable for most background tasks.
 */
export const DEFAULT_JOB_OPTIONS: JobsOptions = {
    removeOnComplete: { age: DAYS_1_S, count: 5_000 },
    removeOnFail: { age: DAYS_1_S, count: 5_000 },
    attempts: 5,
    backoff: { type: "exponential", delay: SECONDS_10_MS },
};

/**
 * Base worker options for all queues unless overridden.
 *
 * @remarks
 *   - lockDuration: Maximum time (ms) a job is locked for processing.
 *   - concurrency: Number of jobs processed in parallel by each worker instance.
 *   - These can be overridden per queue for custom needs.
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

// ---------- base task type --------------------------------------------------

/**
 * Base interface for all task data types in the system.
 * Every job added to any queue should at minimum include this information.
 */
export interface BaseTaskData {
    /** The type of task being processed */
    type: string;
    /** Unique identifier for the task */
    id: string;
    /** User ID of the user who created/owns the task (optional for system tasks) */
    userId?: string;
    /** Current status of the task */
    status?: string;
}

// ---------- abstract queue class -------------------------------------------

/**
 * Configuration object for creating a managed queue.
 *
 * @template Data - The type of job data handled by the queue.
 */
export interface BaseQueueConfig<Data> {
    /** queue name, e.g. "email" or "run"  */
    name: string;
    /** actual business logic – usually imported from ./<queue>/process */
    processor: (job: Job<Data>) => Promise<unknown>;
    /** optional tweaks */
    jobOpts?: Partial<JobsOptions>;
    workerOpts?: Partial<WorkerOptions>;
    /** custom init hook (e.g. schedule cron, register state-machine) */
    onReady?(): Promise<void> | void;
    /** optional validator function for tasks */
    validator?: (data: Data) => { valid: boolean; errors?: string[] };
}

/**
 * ManagedQueue encapsulates a BullMQ queue, worker, and event listeners for a given task type.
 *
 * @template Data - The type of job data handled by the queue.
 *
 * @example
 *   const redis = buildRedis(process.env.REDIS_URL!);
 *   const emailQueue = new ManagedQueue({
 *     name: "email",
 *     processor: processEmailJob,
 *   }, redis);
 *
 *   // Add a job:
 *   await emailQueue.add({ to: "user@example.com", subject: "Welcome!" });
 *
 * @remarks
 *   - Each ManagedQueue instance owns its own queue, worker, and event listeners.
 *   - Designed for use in horizontally scaled environments (multiple processes/containers).
 *   - Handles graceful shutdown and error logging automatically.
 */
export class ManagedQueue<Data> {
    /** Internal queue implementation */
    private readonly _queue: Queue;
    /** The BullMQ worker instance for processing jobs. */
    public readonly worker: Worker<Data>;
    /** The BullMQ event listener for queue events (e.g., failures). */
    public readonly events: QueueEvents;
    /** Promise that resolves when the onReady hook has completed. */
    public readonly ready: Promise<void>;
    /** The name of this queue */
    private readonly queueName: string;
    /** Optional validator function for tasks */
    private readonly validator?: (data: Data) => { valid: boolean; errors?: string[] };

    /**
     * Constructs a new managed queue with the given configuration and Redis connection.
     *
     * @param cfg - The queue configuration (name, processor, options, hooks).
     * @param connection - The shared Redis connection.
     */
    constructor(
        cfg: BaseQueueConfig<Data>,
        connection: IORedis,
    ) {
        this.queueName = cfg.name;
        this.validator = cfg.validator;

        // 1. Queue: Handles job enqueuing and storage in Redis - using untyped Queue to avoid type issues
        this._queue = new Queue(cfg.name, {
            connection,
            defaultJobOptions: { ...DEFAULT_JOB_OPTIONS, ...cfg.jobOpts },
        });

        // 2. Worker: Processes jobs from the queue using the provided processor
        this.worker = new Worker<Data>(
            cfg.name,
            cfg.processor,
            { ...BASE_WORKER_OPTS, ...cfg.workerOpts, connection },
        );
        // Add error listener to catch uncaught 'error' events and prevent the process from crashing
        this.worker.on("error", (error) =>
            logger.error(`Worker for queue ${cfg.name} encountered an error`, { error }),
        );

        // 3. Metrics & failures: Listen for job failures and log them
        this.events = new QueueEvents(cfg.name, { connection });
        this.events.on("failed", ({ jobId, failedReason }) =>
            logger.error(`${cfg.name} job failed`, { jobId, failedReason }),
        );

        // 4. Optional post-setup logic (e.g., cron jobs, state machines)
        if (cfg.onReady) {
            // Promise that resolves when onReady hook has completed, catching any errors
            const onReadyFn = cfg.onReady;
            this.ready = Promise.resolve()
                .then(() => onReadyFn())
                .catch(error => {
                    const queueName = this.queueName; // Capture for the new error
                    logger.error(`onReady hook error for queue ${queueName}`, { originalError: error });
                    // Ensure we throw an actual Error object
                    if (error instanceof Error) {
                        // Optionally augment the error message if needed, or rethrow as is.
                        // For example: error.message = `Queue ${queueName} onReady hook failed: ${error.message}`;
                        throw error;
                    } else {
                        // Wrap non-Error rejections in a new Error object for consistency
                        throw new Error(`Queue ${queueName} onReady hook failed with non-Error value: ${String(error)}`);
                    }
                });
        } else {
            this.ready = Promise.resolve();
        }
    }

    /**
     * Access the underlying BullMQ queue.
     * For operations not covered by the ManagedQueue API.
     */
    get queue(): Queue {
        return this._queue;
    }

    /**
     * Adds a job to the queue.
     *
     * @param data - The job data payload.
     * @param opts - Optional job options (overrides defaults for this job).
     * @returns A promise resolving to the created job.
     *
     * @remarks
     *   - Use this helper to enqueue jobs without importing BullMQ directly.
     *   - The job name is set to the queue's name for consistency.
     */
    add(data: Data, opts: Partial<JobsOptions> = {}) {
        return this._queue.add(this.queueName, data, opts);
    }

    /**
     * Helper function to add a task job to a queue with standardized error handling.
     * 
     * @param data The task data payload, which should extend BaseTaskData.
     * @param opts Optional job options like delays, priority, etc.
     * @returns A Success type with __typename indicator for GraphQL.
     */
    async addTask<T extends BaseTaskData & Data>(
        data: T,
        opts: Partial<JobsOptions> = {},
    ): Promise<Success> {
        try {
            // Validate the task data if a validator is provided
            if (this.validator) {
                const validationResult = this.validator(data);
                if (!validationResult.valid) {
                    const errors = validationResult.errors || ["Task validation failed"];
                    logger.error(`Task validation failed for ${this.queueName}`, {
                        errors,
                        data,
                    });
                    return { __typename: "Success" as const, success: false };
                }
            }

            // Use the task ID as the job ID if available
            const jobOpts: Partial<JobsOptions> = { ...opts };

            if (jobOpts.jobId == null && data.id) {
                jobOpts.jobId = data.id;
            }

            // this.add is expected to return a Job object or throw an error.
            // If it resolves, the job was successfully added.
            await this.add(data, jobOpts);
            return { __typename: "Success" as const, success: true };

        } catch (error) {
            logger.error("Error adding task to queue", {
                queueName: this.queueName,
                error,
                data,
            });
            return { __typename: "Success" as const, success: false };
        }
    }

    /**
     * Get the statuses of multiple tasks from this queue.
     * 
     * @param taskIds - Array of task IDs for which to fetch the statuses.
     * @returns Promise that resolves to an array of objects with task ID and status.
     */
    async getTaskStatuses(taskIds: string[]): Promise<Array<{ id: string, status: string | null }>> {
        return Promise.all(taskIds.map(async (taskId) => {
            try {
                const job = await this._queue.getJob(taskId);
                if (job) {
                    const state = await job.getState();
                    return {
                        id: taskId,
                        // Use the state stored in the data, or fallback to the state from BullMQ. 
                        // This is useful for tasks like runs that may have more specific statuses.
                        status: (job.data as RunTask).status ?? state ?? null,
                    };
                } else {
                    return { id: taskId, status: null };  // Task not found
                }
            } catch (error) {
                logger.error(`Failed to retrieve task ${taskId}`, { error });
                return { id: taskId, status: null };  // Error fetching the job
            }
        }));
    }

    /**
     * Determine the owner of a task (supports userId, startedById, or userData.id).
     */
    static getTaskOwner(task: AnyTask): string | undefined {
        const baseTask = task as BaseTaskData;
        return baseTask.userId ?? (task as RunTask).startedById ?? (task as LLMCompletionTask | SandboxTask | RunTask).userData?.id;
    }

    /**
     * Update a task's status with authentication check.
     * Tasks without a userId cannot have their status updated.
     * 
     * @param taskId - The task ID to update
     * @param status - The new status to set
     * @param userId - The user ID of the requester (for authorization)
     * @returns Promise resolving to a Success type with __typename
     */
    async changeTaskStatus<T extends BaseTaskData>(
        taskId: string,
        status: string,
        userId: string,
    ): Promise<Success> {
        try {
            // Normalize status for consistent comparisons
            const normalizedStatus = status.trim().toLowerCase();
            const job = await this._queue.getJob(taskId);

            if (!job) {
                // If job isn't found but we're changing to a conventionally terminal state,
                // consider it a success. This handles cases where the job might have been
                // auto-removed by BullMQ due to removeOnComplete/removeOnFail settings.
                const trulyTerminalStates = ["completed", "failed"];
                if (trulyTerminalStates.includes(normalizedStatus)) {
                    logger.info(`Task with id ${taskId} not found, but considered a success as status is being changed to a terminal state '${normalizedStatus}'.`);
                    return { __typename: "Success" as const, success: true };
                }

                // For other statuses (like "suggested" or any active/pending state),
                // not finding the job is an error.
                logger.error(`Task with id ${taskId} not found. Cannot change status to '${normalizedStatus}'.`);
                return { __typename: "Success" as const, success: false };
            }

            const data = job.data as T;

            // Determine the real owner using shared helper
            const ownerId = ManagedQueue.getTaskOwner(data as unknown as AnyTask);
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
            logger.error(`Failed to change status for task ${taskId}`, { error });
            return { __typename: "Success" as const, success: false };
        }
    }

    /**
     * Check the health of the queue.
     * 
     * @returns Promise resolving to a QueueHealth object
     */
    async checkHealth() {
        return getQueueHealth(this._queue);
    }

    /**
     * Gracefully closes the queue, worker, and event listeners.
     * This supports proper application shutdown by releasing Redis connections.
     */
    async close(): Promise<void> {
        await Promise.all([
            this.worker.close(),
            this.events.close(),
            this._queue.close(),
        ]);
    }
}

/**
 * Closes all shared Redis connections created by buildRedis.
 * Should be called on application shutdown to clean up resources.
 */
export async function closeRedisConnections(): Promise<void> {
    const clients = Object.values(redisClients);
    // Gracefully quit each Redis client
    await Promise.all(clients.map(client => client.quit()));
    // Clear the client cache
    for (const url in redisClients) {
        delete redisClients[url];
    }
}
