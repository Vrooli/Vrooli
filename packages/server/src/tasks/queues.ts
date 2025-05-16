// queues/index.ts
import { Job, JobsOptions } from "bullmq";
import IORedis from "ioredis";
import { getRedisUrl } from "../redisConn.js";
import { emailProcess } from "./email/process.js";
import { exportProcess } from "./export/process.js";
import { importProcess } from "./import/process.js";
import { llmProcess } from "./llm/process.js";
import { llmTaskProcess } from "./llmTask/process.js";
import { pushProcess } from "./push/process.js";
import { ManagedQueue, buildRedis } from "./queueFactory.js";
import { runProcess } from "./run/process.js";
import { sandboxProcess } from "./sandbox/process.js";
import { smsProcess } from "./sms/process.js";

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
        this.llmTask;
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

    // Expose all queue instances with lazy initialization

    get email() {
        return this.getQueue("email", emailProcess);
    }

    get export() {
        return this.getQueue("export", exportProcess);
    }

    get import() {
        return this.getQueue("import", importProcess);
    }

    get llm() {
        return this.getQueue("llm", llmProcess);
    }

    get llmTask() {
        return this.getQueue("llmTask", llmTaskProcess);
    }

    get push() {
        return this.getQueue("push", pushProcess);
    }

    get run() {
        return this.getQueue("run", runProcess, {
            workerOpts: { concurrency: 8 },
            onReady() {
                // Load watchdog will be implemented in the future
                // For now, the high-load checking functionality is in run/queue.ts
            },
        });
    }

    get sandbox() {
        return this.getQueue("sandbox", sandboxProcess);
    }

    get sms() {
        return this.getQueue("sms", smsProcess);
    }
}
