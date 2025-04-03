import Bull from "bull";

export enum QueueStatus {
    Healthy = "healthy",
    Degraded = "degraded",
    Down = "down"
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
     * @returns The current health status of the queue
     */
    async checkHealth(): Promise<QueueStatus> {
        try {
            // Check if we can communicate with Redis
            await this.queue.client.ping();

            // Get queue statistics
            const jobCounts = await this.queue.getJobCounts();
            const { waiting, active, failed } = jobCounts;

            // Define thresholds for queue health
            const highWaterMark = 1000; // Example threshold
            const failureThreshold = 100; // Example threshold

            // Check for degraded state
            if (waiting + active > highWaterMark || failed > failureThreshold) {
                return QueueStatus.Degraded;
            }

            // If we get here, the queue is healthy
            return QueueStatus.Healthy;
        } catch (error) {
            return QueueStatus.Down;
        }
    }
} 
