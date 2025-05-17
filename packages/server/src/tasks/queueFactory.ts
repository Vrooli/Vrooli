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
import { DAYS_1_S, MINUTES_1_MS, SECONDS_10_MS, Success } from "@local/shared";
import { JobsOptions, Queue, QueueEvents, Worker, WorkerOptions } from "bullmq";
import IORedis from "ioredis";
import { logger } from "../events/logger.js";

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

// ---------- shared objects --------------------------------------------------

/**
 * Creates and returns a shared Redis connection for BullMQ queues and workers.
 *
 * @param url - The Redis connection URL.
 * @returns An IORedis instance for use with BullMQ.
 *
 * @remarks
 *   - Reuses a single Redis connection per Node.js process to reduce overhead.
 *   - Pass this connection to all queues and workers in the process.
 */
export function buildRedis(url: string) {
    return new IORedis(url, { maxRetriesPerRequest: null });
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
* Cheap runtime probe that looks only at Redis metadata – no job bodies are loaded.
* Tune the thresholds to your workload if you need something stricter.
*/
export async function getQueueHealth(
    q: Queue,
    thresholds = { backlog: 500, failed: 25 },
): Promise<{ status: QueueStatus; jobCounts: JobCounts; activeJobs: ActiveJobInfo[] }> {

    const jobCounts = await q.getJobCounts(
        "waiting",
        "delayed",
        "active",
        "failed",
        "completed",
        "paused",
    );

    const backlog = jobCounts.waiting + jobCounts.delayed;
    const active = jobCounts.active;
    const failed = jobCounts.failed;

    let status = QueueStatus.Healthy;
    if (failed > thresholds.failed || backlog > thresholds.backlog * 2) {
        status = QueueStatus.Unhealthy;
    } else if (failed > 0 || backlog > thresholds.backlog) {
        status = QueueStatus.Degraded;
    }

    // grab a *small* sample of the oldest active jobs (cheap, bounded query)
    const activeJobsRaw = await q.getJobs(["active"], 0, 9, false);
    const activeJobs: ActiveJobInfo[] = activeJobsRaw.map(j => ({
        id: j.id as string,
        name: j.name,
        duration: (Date.now() - (j.processedOn ?? Date.now())),
        processedOn: j.processedOn,
    }));

    return { status, jobCounts, activeJobs };
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
    processor: (job: import("bullmq").Job<Data>) => Promise<unknown>;
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

        // 3. Metrics & failures: Listen for job failures and log them
        this.events = new QueueEvents(cfg.name, { connection });
        this.events.on("failed", ({ jobId, failedReason }) =>
            logger.error(`${cfg.name} job failed`, { jobId, failedReason }),
        );

        // 4. Graceful shutdown: Close the worker cleanly on SIGTERM
        process.once("SIGTERM", async () => {
            logger.info(`${cfg.name} worker draining…`);
            await this.worker.close();
        });

        // 5. Optional post-setup logic (e.g., cron jobs, state machines)
        cfg.onReady?.();
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
            const jobOpts: Partial<JobsOptions> = {
                ...opts,
            };

            if (data.id) {
                jobOpts.jobId = data.id;
            }

            const job = await this.add(data, jobOpts);

            if (job) {
                return { __typename: "Success" as const, success: true };
            } else {
                logger.error("Failed to queue the task.", { queueName: this.queueName, data });
                return { __typename: "Success" as const, success: false };
            }
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
    async getTaskStatuses<T extends BaseTaskData>(taskIds: string[]): Promise<Array<{ id: string, status: string | null }>> {
        return Promise.all(taskIds.map(async (taskId) => {
            try {
                const job = await this._queue.getJob(taskId);
                if (job) {
                    const data = job.data as T;
                    return {
                        id: taskId,
                        status: data.status || null,
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
            const job = await this._queue.getJob(taskId);

            if (!job) {
                // If job isn't found but we're changing to a terminal state, consider it a success
                if (["Completed", "Failed", "Suggested"].includes(status)) {
                    logger.info(`Task with id ${taskId} not found, but considered a success for terminal status.`);
                    return { __typename: "Success" as const, success: true };
                }

                logger.error(`Task with id ${taskId} not found.`);
                return { __typename: "Success" as const, success: false };
            }

            const data = job.data as T;

            // Check if the task has a userId - if not, it cannot have its status updated
            if (!data.userId) {
                logger.error(`Task ${taskId} does not have a userId and cannot have its status updated`);
                return { __typename: "Success" as const, success: false };
            }

            // Check if the user has permission to change the status
            if (data.userId !== userId) {
                logger.error(`User ${userId} not authorized to change status of task ${taskId}.`);
                return { __typename: "Success" as const, success: false };
            }

            // Update the job data with the new status
            await job.updateData({
                ...data,
                status,
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
}
