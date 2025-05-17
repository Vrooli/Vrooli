// queues/index.ts
import { Success } from "@local/shared";
import { Job, JobsOptions } from "bullmq";
import IORedis from "ioredis";
import { logger } from "../events/logger.js";
import { getRedisUrl } from "../redisConn.js";
import { emailProcess } from "./email/process.js";
import { exportProcess } from "./export/process.js";
import { importProcess } from "./import/process.js";
import { llmProcess } from "./llm/process.js";
import { pushProcess } from "./push/process.js";
import { BaseTaskData, ManagedQueue, buildRedis } from "./queueFactory.js";
import { runProcess } from "./run/process.js";
import { sandboxProcess } from "./sandbox/process.js";
import { smsProcess } from "./sms/process.js";
import { AnyTask, EmailTask, ExportUserDataTask, ImportUserDataTask, LLMTask, PushNotificationTask, QueueTaskType, RunTask, SMSTask, SandboxExecutionTask } from "./taskTypes.js";

// ---------- task interfaces --------------------------------------------------

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

    private constructor() {
        // Private constructor to enforce singleton pattern
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
     * Initialize the queue manager with a Redis URL
     * @param redisUrl - Redis connection URL
     */
    public init(redisUrl: string): void {
        this.redisUrl = redisUrl;
        this.reset();
    }

    /**
     * Reset all queue connections - useful for testing
     */
    public reset(): void {
        // Close existing connection if it exists
        if (this.connection) {
            this.connection.disconnect();
        }

        // Reset all queue instances
        this.queueInstances = {};

        // Create a new connection if we have a Redis URL
        if (this.redisUrl) {
            this.connection = buildRedis(this.redisUrl);
        } else {
            this.connection = null;
        }
    }

    /**
     * Gracefully shut down all queue workers, event listeners, and Redis connection.
     * Use this for clean application shutdown or when tearing down unit tests.
     * @returns Promise that resolves when all resources have been closed
     */
    public async shutdown(): Promise<void> {
        // Close all queue workers and event listeners
        const closePromises = Object.values(this.queueInstances).map(async (queue) => {
            // Close worker
            await queue.worker.close();
            // Close event listeners
            await queue.events.close();
        });

        // Wait for all workers and event listeners to close
        await Promise.all(closePromises);

        // Disconnect Redis connection if it exists
        if (this.connection) {
            this.connection.disconnect();
            this.connection = null;
        }

        // Clear queue instances
        this.queueInstances = {};
    }

    /**
     * Initialize all queue instances at once
     * @returns Object containing all initialized queue names
     * 
     * This is useful for health checks and ensuring all queues are 
     * properly initialized during application startup.
     */
    public initializeAllQueues(): Record<string, ManagedQueue<any>> {
        // Access each getter to trigger initialization
        this.email;
        this.export;
        this.import;
        this.llm;
        this.push;
        this.run;
        this.sandbox;
        this.sms;

        return this.queueInstances;
    }

    /**
     * Get the Redis connection, initializing with environment value if not already set
     */
    private getConnection(): IORedis {
        if (!this.connection) {
            if (!this.redisUrl) {
                this.redisUrl = getRedisUrl();
            }
            this.connection = buildRedis(this.redisUrl);
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
                    ...options,
                },
                this.getConnection(),
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
        // Determine the correct queue based on the task type
        const queueName = this.getQueueNameForTaskType(data.type as QueueTaskType);
        if (!queueName) {
            return { __typename: "Success" as const, success: false };
        }

        const queue = this.queueInstances[queueName];
        if (!queue) {
            return { __typename: "Success" as const, success: false };
        }

        const result = await queue.addTask(data, opts);
        return { __typename: "Success" as const, success: result.success };
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

        return null;
    }

    /**
     * Get statuses of tasks across all queues or from a specific queue
     * 
     * @param taskIds Task IDs to check
     * @param queueName Optional queue name to check (if known)
     * @returns Array of task status information
     */
    public async getTaskStatuses<T extends BaseTaskData>(
        taskIds: string[],
        queueName?: string,
    ): Promise<TaskStatusInfo[]> {
        // If queue name is provided, check only that queue
        if (queueName && this.queueInstances[queueName]) {
            const statuses = await this.queueInstances[queueName].getTaskStatuses<T>(taskIds);
            return statuses.map(status => ({
                ...status,
                queueName,
            }));
        }

        // Otherwise, check all queues for the tasks
        const allResults: TaskStatusInfo[] = [];
        const foundTasks = new Set<string>();

        // Get all initialized queues
        const queues = Object.entries(this.queueInstances);

        // For each queue, check for our tasks
        for (const [queueName, queue] of queues) {
            // Only check for tasks we haven't found yet
            const remainingTaskIds = taskIds.filter(id => !foundTasks.has(id));
            if (remainingTaskIds.length === 0) break;

            const results = await queue.getTaskStatuses<T>(remainingTaskIds);

            for (const result of results) {
                if (result.status !== null) {
                    foundTasks.add(result.id);
                    allResults.push({
                        ...result,
                        queueName,
                    });
                }
            }
        }

        // Add any tasks we didn't find
        for (const taskId of taskIds) {
            if (!foundTasks.has(taskId)) {
                allResults.push({
                    id: taskId,
                    status: null,
                });
            }
        }

        return allResults;
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
        // If queue name is provided, update only in that queue
        if (queueName && this.queueInstances[queueName]) {
            return this.queueInstances[queueName].changeTaskStatus<T>(taskId, status, userId);
        }

        // Otherwise, try to find the task in all queues
        const queues = Object.values(this.queueInstances);

        for (const queue of queues) {
            const job = await queue.queue.getJob(taskId);
            if (job) {
                // Check if task has a userId - if not, it cannot have its status updated
                if (!job.data.userId) {
                    logger.error(`Task ${taskId} does not have a userId and cannot have its status updated`);
                    return { __typename: "Success" as const, success: false };
                }

                return queue.changeTaskStatus<T>(taskId, status, userId);
            }
        }

        // Task not found in any queue
        if (["Completed", "Failed", "Suggested"].includes(status)) {
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
        return this.getQueue<EmailTask>("email", emailProcess, {
            validator: (data: EmailTask) => {
                const errors: string[] = [];

                // Validate "to" is an array with at least one recipient
                if (!Array.isArray(data.to) || data.to.length === 0) {
                    errors.push("Email must have at least one recipient in the 'to' field");
                }

                // Validate subject is non-empty
                if (!data.subject || data.subject.trim() === "") {
                    errors.push("Email must have a non-empty subject");
                }

                // Validate text content is provided
                if (!data.text || data.text.trim() === "") {
                    errors.push("Email must have text content");
                }

                return {
                    valid: errors.length === 0,
                    errors: errors.length > 0 ? errors : undefined,
                };
            },
        });
    }

    get export(): ManagedQueue<ExportUserDataTask> {
        return this.getQueue<ExportUserDataTask>("export", exportProcess);
    }

    get import(): ManagedQueue<ImportUserDataTask> {
        return this.getQueue<ImportUserDataTask>("import", importProcess);
    }

    get llm(): ManagedQueue<LLMTask> {
        return this.getQueue<LLMTask>("llm", llmProcess);
    }

    get push(): ManagedQueue<PushNotificationTask> {
        return this.getQueue<PushNotificationTask>("push", pushProcess);
    }

    get run(): ManagedQueue<RunTask> {
        return this.getQueue<RunTask>("run", runProcess, {
            workerOpts: { concurrency: 8 },
            onReady() {
                // Load watchdog will be implemented in the future
                // For now, the high-load checking functionality is in run/queue.ts
            },
        });
    }

    get sandbox(): ManagedQueue<SandboxExecutionTask> {
        return this.getQueue<SandboxExecutionTask>("sandbox", sandboxProcess);
    }

    get sms(): ManagedQueue<SMSTask> {
        return this.getQueue<SMSTask>("sms", smsProcess);
    }
}
