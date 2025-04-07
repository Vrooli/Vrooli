import Bull from "bull";

export enum QueueStatus {
    Healthy = "healthy",
    Degraded = "degraded",
    Down = "down"
}

export interface QueueHealthDetails {
    status: QueueStatus;
    jobCounts: {
        waiting: number;
        active: number;
        delayed: number;
        failed: number;
        completed: number;
        total: number;
    };
    activeJobs?: {
        id: string;
        name?: string;
        timestamp: number;
        processedOn?: number;
        duration: number;
    }[];
}

export class BaseQueue<T> {
    private queue: Bull.Queue<T>;

    constructor(name: string, opts: Bull.QueueOptions) {
        this.queue = new Bull<T>(name, opts);
    }

    /**
     * Get the underlying Bull queue
     */
    getQueue(): Bull.Queue<T> {
        return this.queue;
    }

    /**
     * Process jobs in the queue
     */
    process(processor: Bull.ProcessCallbackFunction<T>): void {
        this.queue.process(processor);
    }

    /**
     * Add a job to the queue
     */
    add(data: T, opts?: Bull.JobOptions): Promise<Bull.Job<T>> {
        return this.queue.add(data, opts);
    }

    /**
     * Check the health status of the queue
     * @returns The current health status and detailed information about the queue
     */
    async checkHealth(): Promise<QueueHealthDetails> {
        try {
            // Check if we can communicate with Redis
            await this.queue.client.ping();

            // Get queue statistics
            const jobCounts = await this.queue.getJobCounts();
            const { waiting, active, delayed, failed, completed } = jobCounts;
            const total = waiting + active + delayed + failed + completed;

            // Define thresholds for queue health
            const highWaterMark = 1000; // Example threshold
            const failureThreshold = 100; // Example threshold

            // Get detailed information about active jobs
            const activeJobs = await this.queue.getJobs(["active"]);
            const activeJobsDetails = activeJobs.map(job => {
                const now = Date.now();
                const processedOn = job.processedOn || now;
                return {
                    id: job.id?.toString() || "",
                    name: job.name,
                    timestamp: job.timestamp,
                    processedOn: job.processedOn,
                    duration: now - processedOn, // Duration in milliseconds
                };
            });

            // Determine the health status
            let status = QueueStatus.Healthy;
            if (waiting + active > highWaterMark || failed > failureThreshold) {
                status = QueueStatus.Degraded;
            }

            // Return comprehensive health information
            return {
                status,
                jobCounts: {
                    waiting,
                    active,
                    delayed,
                    failed,
                    completed,
                    total,
                },
                activeJobs: activeJobsDetails,
            };
        } catch (error) {
            // If there's an error, return a minimal response indicating the queue is down
            return {
                status: QueueStatus.Down,
                jobCounts: {
                    waiting: 0,
                    active: 0,
                    delayed: 0,
                    failed: 0,
                    completed: 0,
                    total: 0,
                },
            };
        }
    }
} 
