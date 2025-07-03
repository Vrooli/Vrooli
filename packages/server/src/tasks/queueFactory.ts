/**
 * Production-ready queue factory implementation
 * Addresses connection isolation issues while maintaining efficiency
 */

import { DAYS_1_S, MINUTES_1_MS, SECONDS_10_MS } from "@vrooli/shared";
import { Queue, Worker, QueueEvents, type Job, type WorkerOptions, type JobsOptions, type ConnectionOptions } from "bullmq";
import IORedis, { type RedisOptions } from "ioredis";
import { logger } from "../events/logger.js";

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

/**
 * Base interface for all task data types in the system.
 */
export interface BaseTaskData {
    type: string;
    id: string;
    userId?: string;
    status?: string;
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
            options.enableOfflineQueue = false; // Don't queue commands when disconnected
            options.reconnectOnError = () => false; // Don't reconnect on errors in tests
        } else {
            options.reconnectOnError = (err) => {
                logger.error("Redis reconnect error", { error: err });
                return true; // Always try to reconnect in production
            };
            options.retryStrategy = (times) => {
                const delay = Math.min(times * 50, 2000);
                logger.warn(`Redis retry attempt ${times}, waiting ${delay}ms`);
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
export class ManagedQueue<Data> {
    public readonly queue: Queue;
    public readonly worker: Worker<Data>;
    public readonly events: QueueEvents;
    public readonly ready: Promise<void>;
    
    private readonly queueName: string;
    private readonly validator?: (data: Data) => { valid: boolean; errors?: string[] };
    private readonly connectionMonitor: NodeJS.Timer;
    private isClosing = false;
    
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
        
        this.worker = new Worker<Data>(
            cfg.name,
            cfg.processor,
            {
                ...BASE_WORKER_OPTS,
                ...cfg.workerOpts,
                connection: workerConnectionOptions,
            },
        );
        
        this.events = new QueueEvents(cfg.name, {
            connection: eventsConnectionOptions,
        });
        
        // Set up comprehensive error handling
        this.setupErrorHandlers();
        
        // Monitor connection health
        this.connectionMonitor = setInterval(() => {
            this.checkConnectionHealth();
        }, 30000); // Every 30 seconds
        
        // Handle onReady
        this.ready = this.setupOnReady(cfg.onReady);
    }
    
    private setupErrorHandlers(): void {
        // Worker errors
        this.worker.on("error", (error) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;
            
            logger.error(`Worker error in queue ${this.queueName}`, { 
                error,
                queue: this.queueName,
                component: "worker",
            });
        });
        
        // Queue events errors
        this.events.on("error", (error) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;
            
            logger.error(`QueueEvents error in queue ${this.queueName}`, { 
                error,
                queue: this.queueName,
                component: "events",
            });
        });
        
        // Job failures
        this.events.on("failed", ({ jobId, failedReason, prev }) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;
            
            logger.error(`Job failed in queue ${this.queueName}`, { 
                jobId,
                failedReason,
                previousState: prev,
                queue: this.queueName,
            });
        });
        
        // Stalled jobs (important for production monitoring)
        this.events.on("stalled", ({ jobId }) => {
            // Ignore errors if we're closing
            if (this.isClosing) return;
            
            logger.warn(`Job stalled in queue ${this.queueName}`, { 
                jobId,
                queue: this.queueName,
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
                    logger.error(`onReady hook failed for queue ${this.queueName}`, { error });
                    reject(error);
                }
            });
        });
    }
    
    private async checkConnectionHealth(): Promise<void> {
        // Skip health check if we're closing
        if (this.isClosing) return;
        
        try {
            // Check queue health
            const queueClient = await this.queue.client;
            if (queueClient.status !== "ready") {
                logger.warn(`Queue Redis connection not ready for ${this.queueName}`, {
                    status: queueClient.status,
                    queue: this.queueName,
                });
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
            logger.error(`Health check failed for queue ${this.queueName}`, { error });
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
    async addTask<T extends Data & { id?: string; type: string; userId?: string }>(
        data: T,
        opts: Partial<JobsOptions> = {},
    ): Promise<{ __typename: "Success"; success: boolean }> {
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
            if (jobOpts.jobId == null && data.id) {
                jobOpts.jobId = data.id;
            }
            
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
    
    async close(): Promise<void> {
        // Prevent multiple close attempts
        if (this.isClosing) {
            return;
        }
        this.isClosing = true;
        
        // Clear health monitor first
        clearInterval(this.connectionMonitor);
        
        try {
            // 1. Pause the queue to stop accepting new jobs
            await this.queue.pause().catch(() => {
                // Ignore pause errors - queue might already be paused
            });
            
            // 2. Remove event listeners to stop processing events
            this.worker.removeAllListeners();
            this.events.removeAllListeners();
            this.queue.removeAllListeners();
            
            // 3. Close worker first (stops processing jobs)
            await this.worker.close().catch(error => {
                logger.debug(`Worker close error for ${this.queueName}:`, error);
            });
            
            // 4. Small delay to allow worker to finish cleanup
            await new Promise(resolve => setTimeout(resolve, 50));
            
            // 5. Close events (stops listening for job events)
            await this.events.close().catch(error => {
                logger.debug(`Events close error for ${this.queueName}:`, error);
            });
            
            // 6. Close queue last (closes the main connection)
            await this.queue.close().catch(error => {
                logger.debug(`Queue close error for ${this.queueName}:`, error);
            });
            
            // 7. Final delay to ensure all connections are fully closed
            if (process.env.NODE_ENV === "test") {
                await new Promise(resolve => setTimeout(resolve, 100));
            }
            
        } catch (error) {
            logger.error(`Error during graceful close of queue ${this.queueName}`, { error });
        }
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
                        status: (job.data as any).status ?? state ?? null,
                    };
                } else {
                    return { id: taskId, status: null };
                }
            } catch (error) {
                logger.error(`Failed to retrieve task ${taskId}`, { error });
                return { id: taskId, status: null };
            }
        }));
    }
    
    /**
     * Update a task's status with authentication check.
     * Compatible with ManagedQueue interface.
     */
    async changeTaskStatus<T extends { id?: string; type: string; userId?: string; status?: string }>(
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
            
            const data = job.data as T;
            
            // Check ownership
            const ownerId = data.userId ?? (data as any).startedById ?? (data as any).userData?.id;
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
        logger.debug("clearRedisCache called - no cached connections to clear in isolated architecture");
        
        // Force garbage collection to clean up any lingering connection references
        if (global.gc) {
            global.gc();
        }
    }
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
