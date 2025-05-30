// queues/index.ts
import { MINUTES_1_MS, MINUTES_5_MS, type Success } from "@local/shared";
import { type Job, type JobsOptions } from "bullmq";
import type IORedis from "ioredis";
import { logger } from "../events/logger.js";
import { emailProcess } from "./email/process.js";
import { exportProcess } from "./export/process.js";
import { importProcess } from "./import/process.js";
import { llmProcess } from "./llm/process.js";
import { notificationCreateProcess } from "./notification/process.js";
import { pushProcess } from "./push/process.js";
import type { BaseTaskData } from "./queueFactory.js";
import { buildRedis, ManagedQueue } from "./queueFactory.js";
import { activeRunsRegistry, RUN_TIMEOUT_MS, runProcess, type ActiveRunRecord } from "./run/process.js";
import { sandboxProcess } from "./sandbox/process.js";
import { smsProcess } from "./sms/process.js";
import type { AnyTask, EmailTask, ExportUserDataTask, ImportUserDataTask, LLMTask, NotificationCreateTask, PushNotificationTask, RunTask, SandboxTask, SMSTask } from "./taskTypes.js";
import { QueueTaskType } from "./taskTypes.js";

// High-load pause constants for run queue
const RUN_MAX_ACTIVE = 100;
// eslint-disable-next-line no-magic-numbers
// const RUN_HIGH_LOAD_THRESHOLD = Math.floor(RUN_MAX_ACTIVE * 0.8);
const RUN_HIGH_LOAD_CHECK_INTERVAL_MS = MINUTES_1_MS;
const RUN_LONG_RUNNING_THRESHOLD_FREE_MS = MINUTES_1_MS;
const RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS = MINUTES_5_MS;

/**
 * Structure representing task status information
 */
export interface TaskStatusInfo {
    /** Task identifier */
    id: string;
    /** Current status of the task, null if task not found */
    status: string | null;
    /** Queue name the task belongs to (optional) */
    queueName?: string;
}

/**
 * Interface for success responses
 */
export interface SuccessResponse {
    success: boolean;
}

// ---------- queue service ---------------------------------------------------

/**
 * Singleton class for managing task queues with lazy loading.
 * Queues are only created when first accessed, allowing for swapping 
 * things like the Redis URL during testing.
 */
export class QueueService {
    private static instance: QueueService;
    private redisUrl: string | null = null;
    private connection: IORedis | null = null;
    private queueInstances: Record<string, ManagedQueue<any>> = {};
    private runMonitorInterval: ReturnType<typeof setInterval> | null = null;
    private allQueuesInitialized = false;
    private isInitializingConnection = false; // Prevent re-entrancy for async init
    private connectionPromise: Promise<IORedis | null> | null = null; // To store the promise of init

    private constructor() {
        // Graceful shutdown: ensure all queues are cleaned up on SIGTERM and SIGINT
        const shutdownHandler = async (signal: string) => {
            logger.info(`QueueService received ${signal}. Shutting down all queues...`);
            try {
                await this.shutdown();
            } catch (err) {
                logger.error(`Error shutting down queues on ${signal}`, { error: err });
            } finally {
                process.exit(0);
            }
        };
        process.once("SIGTERM", () => shutdownHandler("SIGTERM"));
        process.once("SIGINT", () => shutdownHandler("SIGINT"));
    }

    /**
     * Get the singleton instance
     */
    public static get(): QueueService {
        if (!QueueService.instance) {
            QueueService.instance = new QueueService();
        }
        return QueueService.instance;
    }

    /**
     * Initialize the queue manager with a Redis URL.
     * This method is idempotent and handles concurrent calls.
     * @param redisUrl - Redis connection URL
     */
    public async init(redisUrl: string): Promise<void> {
        // If already connected and ready, do nothing.
        if (this.connection && this.connection.status === "ready") {
            logger.info("QueueService Redis connection already initialized and ready.");
            return;
        }

        // If initialization is already in progress by another call, await that promise.
        if (this.isInitializingConnection && this.connectionPromise) {
            logger.info("QueueService Redis connection initialization already in progress, awaiting existing promise.");
            try {
                await this.connectionPromise;
                // After awaiting, check status again. If successful, this.connection will be set.
                if (this.connection && this.connection.status === "ready") {
                    logger.info("QueueService: Existing initialization promise resolved successfully.");
                    return;
                }
                logger.warn("QueueService: Existing initialization promise resolved, but connection not ready. Proceeding with new init.");
            } catch (error) {
                logger.warn("QueueService: Awaiting existing initialization promise failed. Proceeding with new init.", { error });
                // Fall through to re-attempt initialization by this call.
            }
            // Ensure isInitializingConnection is false if we are falling through to retry
            this.isInitializingConnection = false;
            this.connectionPromise = null; // Clear the promise as we are retrying
        }

        if (this.isInitializingConnection) {
            // Should ideally not happen if the logic above is correct, but as a safeguard.
            logger.warn("QueueService: Init called while isInitializingConnection is true but no connectionPromise to await. This indicates a potential race or logic error.");
            // Depending on desired behavior, could throw, or wait a bit, or proceed cautiously.
            // For now, let it proceed to re-attempt creating a promise.
        }

        this.isInitializingConnection = true;
        this.redisUrl = redisUrl; // Set or update the redisUrl

        const doInit = async (): Promise<IORedis | null> => {
            try {
                // Gracefully shut down existing resources before creating new ones.
                // This is a simplified shutdown for the connection part;
                // full queue/worker shutdown should be handled by a more explicit shutdown() call if needed here.
                if (this.connection) {
                    try {
                        await this.connection.quit();
                        logger.info("QueueService (doInit): Successfully quit previous Redis connection.");
                    } catch (quitError) {
                        logger.error("QueueService (doInit): Error quitting previous Redis connection. Proceeding anyway.", { quitError });
                    }
                    this.connection = null;
                }

                if (!this.redisUrl) {
                    // This should not happen if init was called with a valid redisUrl, but as a safeguard.
                    logger.error("QueueService (doInit): redisUrl is not set. Cannot build Redis connection.");
                    throw new Error("QueueService: redisUrl not set during connection build.");
                }
                logger.info(`QueueService: Attempting to build new Redis connection to ${this.redisUrl}`);
                this.connection = await buildRedis(this.redisUrl); // buildRedis returns a connected client or throws
                logger.info("QueueService: New Redis connection established successfully.");
                return this.connection;
            } catch (error) {
                logger.error("QueueService: Failed to initialize Redis connection in doInit.", { error });
                this.connection = null; // Ensure connection is null on failure
                throw error; // Re-throw to signal failure
            }
        };

        this.connectionPromise = doInit();

        try {
            await this.connectionPromise;
        } catch (error) {
            // Error is already logged by doInit.
            // The goal here is to ensure isInitializingConnection is reset.
        } finally {
            this.isInitializingConnection = false;
            // Keep connectionPromise to reflect the outcome of the last attempt until a new init starts
            // Or clear it: this.connectionPromise = null; -- choosing to clear for simplicity on next attempt.
            this.connectionPromise = null;
        }

        // After init, if connection is still null, it means it failed.
        if (!this.connection) {
            throw new Error("QueueService: Redis connection initialization failed.");
        }
    }

    /**
     * Reset all queue connections - useful for testing.
     * This will shut down current connections/workers and re-initialize the connection.
     */
    public async reset(): Promise<void> {
        logger.info("QueueService: Resetting queues and Redis connection.");
        await this.shutdown(); // Full shutdown of queues and current connection

        if (this.redisUrl) {
            logger.info(`QueueService: Re-initializing Redis connection to ${this.redisUrl} after reset.`);
            // Call init to re-establish the connection. init itself handles previous connection cleanup.
            await this.init(this.redisUrl);
        } else {
            logger.warn("QueueService: Cannot re-initialize Redis connection after reset as redisUrl is not set.");
            this.connection = null; // Ensure connection is null if no URL
        }
    }

    /**
     * Gracefully shut down all queue workers, event listeners, and Redis connection.
     * Use this for clean application shutdown or when tearing down unit tests.
     * @returns Promise that resolves when all resources have been closed
     */
    public async shutdown(): Promise<void> {
        logger.info("QueueService: Starting shutdown process.");

        if (this.isInitializingConnection && this.connectionPromise) {
            logger.info("QueueService: Shutdown called while connection initialization was in progress. Awaiting its completion.");
            try {
                await this.connectionPromise;
            } catch (e) {
                logger.info("QueueService: Connection initialization during shutdown attempt failed or was already in a failed state.", { error: e });
            }
        }
        this.isInitializingConnection = false;
        this.connectionPromise = null;

        // Close all queue workers, event listeners, and client connections
        const closePromises = Object.entries(this.queueInstances).map(async ([queueName, queueInstance]) => {
            try {
                if (queueInstance.worker) await queueInstance.worker.close();
            } catch (e) { logger.error("Error closing worker", { queueName, error: e }); }
            try {
                if (queueInstance.events) await queueInstance.events.close();
            } catch (e) { logger.error("Error closing queue events", { queueName, error: e }); }
            try {
                if (queueInstance.queue) await queueInstance.queue.close();
            } catch (e) { logger.error("Error closing BullMQ queue", { queueName, error: e }); }
        });

        try {
            await Promise.all(closePromises);
            logger.info("QueueService: All BullMQ queue resources closed.");
        } catch (error) {
            logger.error("QueueService: Error during bulk closure of BullMQ queue resources.", { error });
        }

        // Disconnect Redis connection if it exists
        if (this.connection) {
            try {
                logger.info("QueueService: Quitting main Redis connection.");
                await this.connection.quit(); // Use quit for graceful shutdown
                logger.info("QueueService: Main Redis connection quit successfully.");
            } catch (error) {
                logger.error("QueueService: Error quitting main Redis connection.", { error });
                // As a fallback, attempt to disconnect if quit fails for some reason
                try {
                    this.connection.disconnect();
                    logger.info("QueueService: Main Redis connection disconnected (fallback after quit failed).");
                } catch (disconnectError) {
                    logger.error("QueueService: Error disconnecting main Redis connection (fallback).", { disconnectError });
                }
            }
            this.connection = null;
        }

        // Always clear the run-monitoring interval to avoid orphaned intervals
        if (this.runMonitorInterval) {
            clearInterval(this.runMonitorInterval);
            this.runMonitorInterval = null;
        }

        // Clear queue instances
        this.queueInstances = {};
        // Reset initialization flag
        this.allQueuesInitialized = false;
    }

    /**
     * Initialize all queue instances at once
     * @returns Object containing all initialized queue names
     * 
     * This is useful for health checks and ensuring all queues are 
     * properly initialized during application startup.
     */
    public initializeAllQueues(): Record<string, ManagedQueue<any>> {
        if (this.allQueuesInitialized) {
            return this.queueInstances;
        }
        // Ensure connection is ready before trying to init queues.
        // getConnection() will throw if not ready.
        this.getConnection();

        // Access each getter to trigger initialization
        this.email;
        this.export;
        this.import;
        this.llm;
        this.push;
        this.run;
        this.sandbox;
        this.sms;
        this.notification;
        // Mark initialization as done
        this.allQueuesInitialized = true;
        return this.queueInstances;
    }

    /**
     * Get the Redis connection, initializing with environment value if not already set
     */
    private getConnection(): IORedis {
        if (!this.connection || this.connection.status !== "ready") {
            logger.error(
                "QueueService: getConnection() called but Redis connection is not ready or not initialized. " +
                "This indicates a critical issue, possibly init() was not called or failed.",
                { currentStatus: this.connection?.status },
            );
            throw new Error("QueueService: Redis connection not available or not ready. Ensure init() was called and succeeded at application startup.");
        }
        return this.connection;
    }

    /**
     * Get a queue instance, creating it if it doesn't exist
     */
    private getQueue<T>(name: string, processor: (job: Job<T>) => Promise<unknown>, options?: {
        workerOpts?: any;
        jobOpts?: Partial<JobsOptions>;
        onReady?: () => void | Promise<void>;
        validator?: (data: T) => { valid: boolean; errors?: string[] };
    }): ManagedQueue<T> {
        // If queue doesn't exist, create it
        if (!this.queueInstances[name]) {
            this.queueInstances[name] = new ManagedQueue<T>(
                {
                    name,
                    processor,
                    validator: options?.validator, // Ensure validator is passed
                    jobOpts: options?.jobOpts,     // Ensure jobOpts is passed
                    workerOpts: options?.workerOpts, // Ensure workerOpts is passed
                    onReady: options?.onReady,     // Ensure onReady is passed
                },
                this.getConnection(), // This will now throw if connection not ready
            );
        }
        return this.queueInstances[name] as ManagedQueue<T>;
    }

    /**
     * Add a task to a specific queue with proper error handling and typing
     * Use type checking to ensure the task type matches the queue
     * 
     * @param data Task data that must include base task fields
     * @param opts Optional job options
     * @returns Success response with __typename
     */
    public async addTask<T extends AnyTask>(
        data: T,
        opts: Partial<JobsOptions> = {},
    ): Promise<Success> {
        switch (data.type) {
            case QueueTaskType.EMAIL_SEND:
                return this.email.addTask(data as EmailTask, opts);
            case QueueTaskType.EXPORT_USER_DATA:
                return this.export.addTask(data as ExportUserDataTask, opts);
            case QueueTaskType.IMPORT_USER_DATA:
                return this.import.addTask(data as ImportUserDataTask, opts);
            case QueueTaskType.LLM_COMPLETION:
                return this.llm.addTask(data as LLMTask, opts);
            case QueueTaskType.PUSH_NOTIFICATION:
                return this.push.addTask(data as PushNotificationTask, opts);
            case QueueTaskType.RUN_START:
                return this.run.addTask(data as RunTask, opts);
            case QueueTaskType.SANDBOX_EXECUTION:
                return this.sandbox.addTask(data as SandboxTask, opts);
            case QueueTaskType.SMS_MESSAGE:
                return this.sms.addTask(data as SMSTask, opts);
            case QueueTaskType.NOTIFICATION_CREATE:
                return this.notification.addTask(data as NotificationCreateTask, opts);
            default: {
                const taskTypeStr = (data as any).type as string;
                logger.warn(`QueueService: addTask received unhandled explicit task type: ${taskTypeStr}. Attempting fallback routing.`);
                // Fallback prefix-based routing for unknown task types (this part can remain if needed)
                const fallbackQueueName = this.getQueueNameForTaskType(taskTypeStr);
                if (fallbackQueueName) {
                    logger.info(`QueueService: Fallback routing task type ${taskTypeStr} to queue ${fallbackQueueName}.`);
                    // Use the named queue getter to ensure the queue is instantiated with the correct processor
                    const queue = (this as any)[fallbackQueueName] as ManagedQueue<any>;
                    return queue.addTask(data as any, opts);
                }
                logger.error(`Unsupported task type: ${(data as any).type}`);
                return { __typename: "Success" as const, success: false };
            }
        }
    }

    /**
     * Get the appropriate queue name for a given task type
     * @param taskType The task type to map to a queue
     * @returns The queue name, or null if no mapping exists
     */
    private getQueueNameForTaskType(taskType: QueueTaskType | string): string | null {
        const taskTypeStr = String(taskType);

        // Map task types to queue names based on prefix
        if (taskTypeStr.startsWith("email:")) return "email";
        if (taskTypeStr.startsWith("export:")) return "export";
        if (taskTypeStr.startsWith("import:")) return "import";
        if (taskTypeStr.startsWith("llm:")) return "llm";
        if (taskTypeStr.startsWith("push:")) return "push";
        if (taskTypeStr.startsWith("run:")) return "run";
        if (taskTypeStr.startsWith("sandbox:")) return "sandbox";
        if (taskTypeStr.startsWith("sms:")) return "sms";
        if (taskTypeStr.startsWith("notification:")) return "notification";

        return null;
    }

    /**
     * Get statuses of tasks across all queues or from a specific queue
     * 
     * @param taskIds Task IDs to check
     * @param queueName Optional queue name to check (if known)
     * @returns Array of task status information
     */
    public async getTaskStatuses(
        taskIds: string[],
        queueName?: string,
    ): Promise<TaskStatusInfo[]> {
        // Reuse existing managed queues for efficient status checks
        this.initializeAllQueues();
        const queueNames = queueName
            ? [queueName]
            : Object.keys(this.queueInstances);
        const resultsMap = new Map<string, TaskStatusInfo>();

        // Query each managed queue instance directly
        for (const name of queueNames) {
            const queue = this.queueInstances[name];
            // Fetch statuses from the managed queue
            const statuses = await queue.getTaskStatuses(taskIds);
            for (const { id, status } of statuses) {
                if (status) {
                    // Normalize to PascalCase
                    const normalized = status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
                    resultsMap.set(id, { id, status: normalized, queueName: name });
                }
            }
        }

        // Return results in the original order, defaulting to null status when not found
        return taskIds.map(id => resultsMap.get(id) ?? { id, status: null });
    }

    /**
     * Change task status with authentication
     * Tasks without a userId cannot have their status updated.
     * 
     * @param taskId Task ID to update
     * @param status New status to set
     * @param userId User ID for authentication
     * @param queueName Optional queue name if known
     * @returns Success response with __typename
     */
    public async changeTaskStatus<T extends BaseTaskData>(
        taskId: string,
        status: string,
        userId: string,
        queueName?: string,
    ): Promise<Success> {
        // Normalize status for case-insensitive comparisons
        const normalizedStatus = status.trim().toLowerCase();
        // Ensure all queue instances are initialized before attempting to change status
        this.initializeAllQueues();
        // If queue name is provided, update only in that queue
        if (queueName && this.queueInstances[queueName]) {
            // Ownership check for specific queue
            const queue = this.queueInstances[queueName];
            const job = await queue.queue.getJob(taskId);
            if (!job) {
                // Terminal statuses should be idempotent when the job no longer exists
                if (["completed", "failed", "suggested"].includes(normalizedStatus)) {
                    return { __typename: "Success" as const, success: true };
                }
                logger.error(`Task ${taskId} not found in queue ${queueName}`);
                return { __typename: "Success" as const, success: false };
            }
            // Determine task owner (supports userId, startedById, or userData.id)
            const ownerId = ManagedQueue.getTaskOwner(job.data as AnyTask);
            if (!ownerId) {
                logger.error(`Task ${taskId} does not have an owner and cannot have its status updated`);
                return { __typename: "Success" as const, success: false };
            }
            if (ownerId !== userId) {
                logger.error(`User ${userId} is not allowed to update task ${taskId} owned by ${ownerId}`);
                return { __typename: "Success" as const, success: false };
            }
            return queue.changeTaskStatus<T>(taskId, status, userId);
        }

        // Otherwise, try to find the task in all queues
        const queues = Object.values(this.queueInstances);

        for (const queue of queues) {
            const job = await queue.queue.getJob(taskId);
            if (job) {
                // Determine task owner (supports userId, startedById, or userData.id)
                const ownerId2 = ManagedQueue.getTaskOwner(job.data as AnyTask);
                if (!ownerId2) {
                    logger.error(`Task ${taskId} does not have an owner and cannot have its status updated`);
                    return { __typename: "Success" as const, success: false };
                }
                // Enforce ownership: only the task owner may change its status
                if (ownerId2 !== userId) {
                    logger.error(`User ${userId} is not allowed to update task ${taskId} owned by ${ownerId2}`);
                    return { __typename: "Success" as const, success: false };
                }
                // Perform the status change
                return queue.changeTaskStatus<T>(taskId, status, userId);
            }
        }

        // Task not found in any queue
        if (["completed", "failed", "suggested"].includes(normalizedStatus)) {
            return { __typename: "Success" as const, success: true };
        }

        return { __typename: "Success" as const, success: false };
    }

    /** Read-only handle for external diagnostics (e.g. HealthService) */
    public get queues(): Readonly<Record<string, ManagedQueue<any>>> {
        return this.queueInstances;
    }

    // Expose all queue instances with lazy initialization

    get email(): ManagedQueue<EmailTask> {
        return this.getQueue(QueueTaskType.EMAIL_SEND, emailProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_EMAIL_CONCURRENCY || "5") },
        });
    }

    get export(): ManagedQueue<ExportUserDataTask> {
        return this.getQueue(QueueTaskType.EXPORT_USER_DATA, exportProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_EXPORT_CONCURRENCY || "2") },
        });
    }

    get import(): ManagedQueue<ImportUserDataTask> {
        return this.getQueue(QueueTaskType.IMPORT_USER_DATA, importProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_IMPORT_CONCURRENCY || "2") },
        });
    }

    get llm(): ManagedQueue<LLMTask> {
        return this.getQueue(QueueTaskType.LLM_COMPLETION, llmProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_LLM_CONCURRENCY || "10") },
        });
    }

    get push(): ManagedQueue<PushNotificationTask> {
        return this.getQueue(QueueTaskType.PUSH_NOTIFICATION, pushProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_PUSH_CONCURRENCY || "10") },
        });
    }

    get run(): ManagedQueue<RunTask> {
        const runQueue = this.getQueue(QueueTaskType.RUN_START, runProcess, {
            workerOpts: {
                concurrency: RUN_MAX_ACTIVE,
                // BullMQ's basic limiter (if used) is usually for max jobs in a duration for a group, not direct concurrency.
                // Concurrency is the main control here.
                // For more advanced rate limiting (e.g., per IP/user), custom logic or BullMQ Pro would be needed.
            },
            jobOpts: { timeout: RUN_TIMEOUT_MS } as Partial<JobsOptions>, // Explicitly cast to satisfy linter if type inference is struggling. Ensure this is the correct type for defaultJobOptions.
            onReady: () => { // Start the run monitor when the 'run' queue is ready
                if (this.runMonitorInterval) {
                    clearInterval(this.runMonitorInterval);
                }
                this.runMonitorInterval = setInterval(
                    checkLongRunningRuns,
                    RUN_HIGH_LOAD_CHECK_INTERVAL_MS,
                );
                logger.info("Run queue is ready, run monitor started.");
            },
        });
        // Additional setup specific to 'run' queue, like pausing logic, could go here if needed
        // For example, monitor active jobs and pause/resume:
        // (This is conceptual and would need careful implementation to avoid race conditions)
        // setInterval(async () => {
        //     const activeJobs = await runQueue.worker.getActiveCount();
        //     if (activeJobs >= RUN_HIGH_LOAD_THRESHOLD && !runQueue.worker.isPaused()) {
        //         await runQueue.worker.pause();
        //         logger.info("Run queue paused due to high load.");
        //     } else if (activeJobs < RUN_HIGH_LOAD_THRESHOLD && runQueue.worker.isPaused()) {
        //         await runQueue.worker.resume();
        //         logger.info("Run queue resumed.");
        //     }
        // }, RUN_HIGH_LOAD_CHECK_INTERVAL_MS);
        return runQueue;
    }

    get sandbox(): ManagedQueue<SandboxTask> {
        return this.getQueue(QueueTaskType.SANDBOX_EXECUTION, sandboxProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_SANDBOX_CONCURRENCY || "5") },
        });
    }

    get sms(): ManagedQueue<SMSTask> {
        return this.getQueue(QueueTaskType.SMS_MESSAGE, smsProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_SMS_CONCURRENCY || "3") },
        });
    }

    /**
     * Queue for creating in-app notification records and emitting socket events.
     */
    get notification(): ManagedQueue<NotificationCreateTask> {
        return this.getQueue(QueueTaskType.NOTIFICATION_CREATE, notificationCreateProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_NOTIFICATION_CONCURRENCY || "5") },
        });
    }
}

/**
 * Periodically pause long-running runs under high load.
 */
function checkLongRunningRuns() {
    const now = Date.now();
    try {
        if (activeRunsRegistry.count() === 0) { // Use the count() method
            return;
        }

        // Use getOrderedRuns() which returns ActiveRunRecord[]
        const runsToProcess: ActiveRunRecord[] = activeRunsRegistry.getOrderedRuns();

        // The check `typeof runsToProcess[Symbol.iterator] !== 'function'` is usually for iterables,
        // but getOrderedRuns() directly returns an array, which is always iterable.
        // A simple check for an empty array (though count() already does this) or null/undefined is sufficient.
        if (!runsToProcess) { // getOrderedRuns should not return null/undefined based on its signature, but defensive check.
            logger.warn("[checkLongRunningRuns] activeRunsRegistry.getOrderedRuns() returned an unexpected value. Skipping check.");
            return;
        }

        for (const runInfo of runsToProcess) {
            // Ensure runInfo and its properties are valid before using them
            if (!runInfo || typeof runInfo.startTime !== "number" || typeof runInfo.runId !== "string") {
                logger.warn("[checkLongRunningRuns] Encountered invalid runInfo object in a theoretically valid array.", { runInfo });
                continue;
            }
            const threshold = runInfo.hasPremium ? RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS : RUN_LONG_RUNNING_THRESHOLD_FREE_MS;
            if (now - runInfo.startTime > threshold) {
                logger.warn(`Job related to run ${runInfo.runId} in queue 'run' has been active for more than ${threshold / MINUTES_1_MS} minutes.`, {
                    runId: runInfo.runId,
                    startTime: new Date(runInfo.startTime).toISOString(),
                    durationMs: now - runInfo.startTime,
                    hasPremium: runInfo.hasPremium,
                });
                // TODO: Implement further actions...
            }
        }
    } catch (error) {
        logger.error("[checkLongRunningRuns] Error during execution.", { error });
    }
}
