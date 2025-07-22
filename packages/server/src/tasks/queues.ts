// queues/index.ts
// AI_CHECK: TYPE_SAFETY=critical-generics | LAST: 2025-07-04 - Added proper types for lazy-loaded modules and queue limits
import { type Success } from "@vrooli/shared";
import { type Job, type JobsOptions, type WorkerOptions } from "bullmq";
import { logger } from "../events/logger.js";
import { checkLongRunningTasksInRegistry, type ActiveTaskRegistryLimits } from "./activeTaskRegistry.js";
import { emailProcess } from "./email/process.js";
import { exportProcess } from "./export/process.js";
import { importProcess } from "./import/process.js";
import { notificationCreateProcess } from "./notification/process.js";
import { pushProcess } from "./push/process.js";
import type { BaseQueueConfig } from "./queueFactory.js";
import { ManagedQueue } from "./queueFactory.js";
import { sandboxProcess } from "./sandbox/process.js";
import { smsProcess } from "./sms/process.js";


// Lazy load queue limits to avoid circular dependencies
let _RUN_QUEUE_LIMITS: ActiveTaskRegistryLimits | null = null;
let _SWARM_QUEUE_LIMITS: ActiveTaskRegistryLimits | null = null;

// Default limits used before async load completes
const DEFAULT_RUN_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: 5,
    highLoadCheckIntervalMs: 5000,
    taskTimeoutMs: 600000,
    highLoadThresholdPercentage: 0.8,
    longRunningThresholdFreeMs: 60000,
    longRunningThresholdPremiumMs: 300000,
    shutdownGracePeriodMs: 10000,
    onLongRunningFirstThreshold: "pause",
    longRunningPauseRetries: 1,
    longRunningStopRetries: 0,
};

const DEFAULT_SWARM_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: 10,
    highLoadCheckIntervalMs: 5000,
    taskTimeoutMs: 600000,
    highLoadThresholdPercentage: 0.8,
    longRunningThresholdFreeMs: 60000,
    longRunningThresholdPremiumMs: 300000,
    shutdownGracePeriodMs: 10000,
    onLongRunningFirstThreshold: "pause",
    longRunningPauseRetries: 1,
    longRunningStopRetries: 0,
};

async function getRunQueueLimits(): Promise<ActiveTaskRegistryLimits> {
    if (!_RUN_QUEUE_LIMITS) {
        const module = await import("./run/limits.js");
        _RUN_QUEUE_LIMITS = module.RUN_QUEUE_LIMITS;
    }
    return _RUN_QUEUE_LIMITS;
}

async function getSwarmQueueLimits(): Promise<ActiveTaskRegistryLimits> {
    if (!_SWARM_QUEUE_LIMITS) {
        const module = await import("./swarm/limits.js");
        _SWARM_QUEUE_LIMITS = module.SWARM_QUEUE_LIMITS;
    }
    return _SWARM_QUEUE_LIMITS;
}

// Synchronously try to get limits, returning defaults if not yet loaded
function getRunQueueLimitsSync(): ActiveTaskRegistryLimits {
    return _RUN_QUEUE_LIMITS || DEFAULT_RUN_LIMITS;
}

function getSwarmQueueLimitsSync(): ActiveTaskRegistryLimits {
    return _SWARM_QUEUE_LIMITS || DEFAULT_SWARM_LIMITS;
}

import type { AnyTask, BaseTaskData, EmailTask, ExportUserDataTask, ImportUserDataTask, NotificationCreateTask, PushNotificationTask, RunTask, SandboxTask, SMSTask, SwarmTask } from "./taskTypes.js";
import { QueueTaskType } from "./taskTypes.js";

/**
 * Lazy imports to break circular dependencies
 */
async function getSwarmDependencies() {
    const swarmModule = await import("./swarm/process.js");
    return {
        activeSwarmRegistry: swarmModule.activeSwarmRegistry,
        llmProcess: swarmModule.llmProcess,
    };
}

async function getRunDependencies() {
    const runModule = await import("./run/process.js");
    return {
        activeRunRegistry: runModule.activeRunRegistry,
        runProcess: runModule.runProcess,
    };
}

/**
 * Generic representation of an active task record from any registry.
 */
export interface GenericActiveTaskRecord {
    hasPremium: boolean;
    startTime: number;
    // Include common identifiers, e.g., a generic id or specific ones if they align
    // For now, we'll assume the specific checking function will know how to get the task's unique ID for logging.
    // Alternatively, the registry's getOrderedActiveRecords method could return a consistently structured object.
    id: string; // runId or conversationId
    userId?: string; // Optional: if available, for logging or notifications
}

/**
 * Interface for a generic active task registry.
 * Both ActiveRunsRegistry and ActiveSwarmRegistry should conform to this (duck-typing for now).
 */
export interface GenericActiveTaskRegistry {
    count(): number;
    // Method to get all active records, ordered by start time (oldest first)
    // The actual method names are getOrderedRuns and getOrderedSwarms.
    // The generic function will need to accept these specific registry types for now.
    getOrderedActiveRecords(): GenericActiveTaskRecord[];
}

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
    private static instance: QueueService | null = null;
    private queueInstances: Record<string, ManagedQueue<any>> = {};
    private runMonitorInterval: ReturnType<typeof setInterval> | null = null;
    private swarmMonitorInterval: ReturnType<typeof setInterval> | null = null;
    private allQueuesInitialized = false;
    private processListeners: { signal: string; handler: () => void }[] = []; // Track process listeners

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

        function sigtermHandler() {
            shutdownHandler("SIGTERM");
        }
        function sigintHandler() {
            shutdownHandler("SIGINT");
        }

        process.once("SIGTERM", sigtermHandler);
        process.once("SIGINT", sigintHandler);

        // Store references for cleanup
        this.processListeners.push(
            { signal: "SIGTERM", handler: sigtermHandler },
            { signal: "SIGINT", handler: sigintHandler },
        );
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
     * Reset singleton instance (for testing)
     * This ensures each test gets a fresh connection
     */
    public static async reset(): Promise<void> {
        if (QueueService.instance) {
            await QueueService.instance.shutdown();
            QueueService.instance = null;
        }
    }

    /**
     * Initialize the queue manager with a Redis URL.
     * @param redisUrl - Redis connection URL (for reference, queues create their own connections)
     */
    public async init(redisUrl: string): Promise<void> {
        // Set Redis URL in environment for queues to use
        if (!process.env.REDIS_URL) {
            process.env.REDIS_URL = redisUrl;
        }
        // Redis URL configured for queue connections
    }

    /**
     * Reset all queue connections - useful for testing.
     */
    public async reset(): Promise<void> {
        await this.shutdown(); // Full shutdown of all queues

        // Also clean up test Redis connections if in test environment
        if (process.env.NODE_ENV === "test") {
            try {
                const { closeRedisConnections, clearRedisCache } = await import("./queueFactory.js");
                await closeRedisConnections();
                clearRedisCache();
                // Test Redis connections cleaned up
            } catch (error) {
                // Error cleaning up test Redis connections
            }
        }
    }

    /**
     * Gracefully shut down all queue workers, event listeners, and connections.
     * @returns Promise that resolves when all resources have been closed
     */
    public async shutdown(): Promise<void> {

        // Close all queue instances
        const closePromises = Object.entries(this.queueInstances).map(async ([queueName, queueInstance]) => {
            try {
                await queueInstance.close();
                logger.debug(`QueueService: Closed queue ${queueName}`);
            } catch (e) {
                logger.error("Error closing queue", { queueName, error: e });
            }
        });

        try {
            await Promise.all(closePromises);
        } catch (error) {
            logger.error("QueueService: Error during queue closure.", { error });
        }

        // Clear monitoring intervals
        if (this.runMonitorInterval) {
            clearInterval(this.runMonitorInterval);
            this.runMonitorInterval = null;
        }
        if (this.swarmMonitorInterval) {
            clearInterval(this.swarmMonitorInterval);
            this.swarmMonitorInterval = null;
        }

        // Remove process event listeners to prevent memory leaks
        this.removeProcessListeners();

        // Clear queue instances and creation promises
        this.queueInstances = {};
        this.queueCreationPromises = {};
        this.allQueuesInitialized = false;

        // Also shutdown SocketService if it was initialized by queue operations
        try {
            const { SocketService } = await import("../sockets/io.js");
            if (SocketService) {
                await SocketService.shutdown();
                logger.debug("QueueService: SocketService shutdown completed during queue shutdown");
            }
        } catch (e) {
            logger.debug("QueueService: SocketService not available or failed to shutdown", { error: e });
        }
    }

    /**
     * Remove process event listeners to prevent memory leaks
     * This is especially important in testing environments
     */
    private removeProcessListeners(): void {
        for (const { signal, handler } of this.processListeners) {
            try {
                process.removeListener(signal as NodeJS.Signals, handler);
                logger.debug(`QueueService: Removed ${signal} listener`);
            } catch (error) {
                logger.warn(`QueueService: Failed to remove ${signal} listener`, { error });
            }
        }
        this.processListeners = [];
    }

    /**
     * Initialize all queue instances with staggered connections
     * @returns Promise that resolves to object containing all initialized queue names
     * 
     * This is useful for health checks and ensuring all queues are 
     * properly initialized during application startup. Queues are initialized
     * with a small delay between each to prevent Redis connection storms.
     */
    public async initializeAllQueues(): Promise<Record<string, ManagedQueue<any>>> {
        if (this.allQueuesInitialized) {
            return this.queueInstances;
        }
        
        // Define queue getters with names for logging
        const queueGetters: Array<{ name: string; getter: () => ManagedQueue<any> }> = [
            { name: "email", getter: () => this.email },
            { name: "export", getter: () => this.export },
            { name: "import", getter: () => this.import },
            { name: "swarm", getter: () => this.swarm },
            { name: "push", getter: () => this.push },
            { name: "run", getter: () => this.run },
            { name: "sandbox", getter: () => this.sandbox },
            { name: "sms", getter: () => this.sms },
            { name: "notification", getter: () => this.notification },
        ];
        
        // Initialize queues with a small delay between each
        for (const { name, getter } of queueGetters) {
            try {
                getter(); // This will start the async creation
                
                // Add a delay between queue initializations to prevent connection storm
                // Skip delay in test environment for faster tests
                if (process.env.NODE_ENV !== "test") {
                    await new Promise(resolve => setTimeout(resolve, 300));
                }
            } catch (error) {
                logger.error(`QueueService: Failed to start initialization of ${name} queue`, { error });
                // Continue with other queues even if one fails
            }
        }
        
        // Wait for all queues to be created
        try {
            await Promise.all(Object.values(this.queueCreationPromises));
        } catch (error) {
            logger.error("QueueService: Error waiting for queue creation", { error });
        }
        
        // Mark initialization as done
        this.allQueuesInitialized = true;
        return this.queueInstances;
    }


    private queueCreationPromises: Record<string, Promise<ManagedQueue<any>>> = {};

    /**
     * Get a queue instance, creating it if it doesn't exist
     */
    private getQueue<T extends BaseTaskData | Record<string, unknown>>(name: string, processor: (job: Job<T>) => Promise<unknown>, options?: {
        workerOpts?: Partial<WorkerOptions>;
        jobOpts?: Partial<JobsOptions>;
        onReady?: () => void | Promise<void>;
        validator?: (data: T) => { valid: boolean; errors?: string[] };
    }): ManagedQueue<T> {
        // If queue doesn't exist, create it
        if (!this.queueInstances[name]) {
            // Create a proxy that will defer to the actual queue once it's created
            const proxyQueue = new Proxy({} as ManagedQueue<T>, {
                get: (target, prop) => {
                    // For critical properties that might be accessed before initialization,
                    // ensure the queue is created
                    if (name in this.queueCreationPromises) {
                        // Return a function that waits for the queue to be created
                        if (typeof prop === "string" && ["addTask", "getTaskStatuses", "changeTaskStatus", "close", "checkHealth", "getMetrics", "add"].includes(prop)) {
                            return async (...args: any[]) => {
                                const actualQueue = await this.queueCreationPromises[name];
                                return (actualQueue as any)[prop](...args);
                            };
                        }
                        
                        // For properties like queue, worker, events, throw an error
                        if (["queue", "worker", "events"].includes(prop as string)) {
                            throw new Error(`Queue ${name} is still being initialized. Please await queue operations before accessing ${String(prop)}.`);
                        }
                        
                        // For ready, return the promise
                        if (prop === "ready") {
                            return this.queueCreationPromises[name].then(q => q.ready);
                        }
                    }
                    
                    // If the queue is already created, return the property
                    const actualQueue = this.queueInstances[name];
                    if (actualQueue) {
                        return (actualQueue as any)[prop];
                    }
                    
                    throw new Error(`Queue ${name} not initialized`);
                },
            });
            
            // Store the proxy immediately
            this.queueInstances[name] = proxyQueue;
            
            // Start the async creation
            const config: BaseQueueConfig<T> = {
                name,
                processor,
                validator: options?.validator,
                jobOpts: options?.jobOpts,
                workerOpts: options?.workerOpts,
                onReady: options?.onReady,
            };
            
            this.queueCreationPromises[name] = ManagedQueue.create<T>(config).then(queue => {
                // Replace the proxy with the actual queue
                this.queueInstances[name] = queue;
                return queue;
            }).catch(error => {
                logger.error(`Failed to create queue ${name}`, { error });
                delete this.queueInstances[name];
                delete this.queueCreationPromises[name];
                throw error;
            });
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
            case QueueTaskType.SWARM_EXECUTION:
                return this.swarm.addTask(data as SwarmTask, opts);
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
                const taskTypeStr = (data as AnyTask).type as string;
                logger.warn(`QueueService: addTask received unhandled explicit task type: ${taskTypeStr}. Attempting fallback routing.`);
                // Fallback prefix-based routing for unknown task types (this part can remain if needed)
                const fallbackQueueName = this.getQueueNameForTaskType(taskTypeStr);
                if (fallbackQueueName) {
                    logger.info(`QueueService: Fallback routing task type ${taskTypeStr} to queue ${fallbackQueueName}.`);
                    // Use the named queue getter to ensure the queue is instantiated with the correct processor
                    const queue = (this as Record<string, unknown>)[fallbackQueueName] as ManagedQueue<BaseTaskData>;
                    return queue.addTask(data as BaseTaskData, opts);
                }
                logger.error(`Unsupported task type: ${taskTypeStr}`);
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
        if (taskTypeStr.startsWith("swarm:")) return "swarm";
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
    public async changeTaskStatus(
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
            return queue.changeTaskStatus(taskId, status, userId);
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
                return queue.changeTaskStatus(taskId, status, userId);
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

    get swarm(): ManagedQueue<SwarmTask> {
        // Use synchronous limits (defaults if not loaded yet)
        const limits = getSwarmQueueLimitsSync();
        return this.getQueue(QueueTaskType.LLM_COMPLETION, async (job) => {
            const { llmProcess } = await getSwarmDependencies();
            return llmProcess(job);
        }, {
            workerOpts: {
                concurrency: limits.maxActive,
            },
            jobOpts: { priority: 50, timeout: limits.taskTimeoutMs } as Partial<JobsOptions>,
            onReady: async () => {
                // Load actual limits asynchronously
                const actualLimits = await getSwarmQueueLimits();

                const isTest = process.env.NODE_ENV === "test";

                if (this.swarmMonitorInterval) {
                    clearInterval(this.swarmMonitorInterval);
                }

                // Only start monitor in non-test environments
                if (!isTest) {
                    this.swarmMonitorInterval = setInterval(
                        async () => {
                            try {
                                const { activeSwarmRegistry } = await getSwarmDependencies();
                                await checkLongRunningTasksInRegistry(
                                    activeSwarmRegistry,
                                    actualLimits,
                                    "Swarm",
                                );
                            } catch (error) {
                                logger.error("Error in swarm monitor interval", { error });
                            }
                        },
                        actualLimits.highLoadCheckIntervalMs,
                    );
                    logger.info("Swarm queue is ready, swarm monitor started.");
                }
            },
        });
    }

    get push(): ManagedQueue<PushNotificationTask> {
        return this.getQueue(QueueTaskType.PUSH_NOTIFICATION, pushProcess, {
            workerOpts: { concurrency: parseInt(process.env.WORKER_PUSH_CONCURRENCY || "10") },
        });
    }

    get run(): ManagedQueue<RunTask> {
        // Use synchronous limits (defaults if not loaded yet)
        const limits = getRunQueueLimitsSync();
        const runQueue = this.getQueue<RunTask>(QueueTaskType.RUN_START, async (job: Job<RunTask>) => {
            const { runProcess } = await getRunDependencies();
            return runProcess(job);
        }, {
            workerOpts: {
                concurrency: limits.maxActive,
            },
            jobOpts: { priority: 40, timeout: limits.taskTimeoutMs } as Partial<JobsOptions>,
            onReady: async () => {
                // Load actual limits asynchronously
                const actualLimits = await getRunQueueLimits();

                const isTest = process.env.NODE_ENV === "test";

                if (this.runMonitorInterval) {
                    clearInterval(this.runMonitorInterval);
                }

                // Only start monitor in non-test environments
                if (!isTest) {
                    this.runMonitorInterval = setInterval(
                        async () => {
                            try {
                                const { activeRunRegistry } = await getRunDependencies();
                                await checkLongRunningTasksInRegistry(
                                    activeRunRegistry,
                                    actualLimits,
                                    "Run",
                                );
                            } catch (error) {
                                logger.error("Error in run monitor interval", { error });
                            }
                        },
                        actualLimits.highLoadCheckIntervalMs,
                    );
                    logger.info("Run queue is ready, run monitor started.");
                }
            },
        });
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
