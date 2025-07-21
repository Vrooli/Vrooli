import { HttpStatus, SECONDS_1_MS } from "@vrooli/shared";
import type { Request, Response } from "express";
import { type Express } from "express";
import os from "os";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { CacheService } from "../redisConn.js";
import { SocketService } from "../sockets/io.js";
import { QueueService } from "../tasks/queues.js";
import { AIServiceRegistry } from "./response/registry.js";

const debug = process.env.NODE_ENV === "development";

export interface SystemMetrics {
    timestamp: number;
    uptime: number;
    system: SystemResourceMetrics;
    application: ApplicationMetrics;
    database: DatabaseMetrics;
    redis: RedisMetrics;
    api: ApiMetrics;
    execution?: ExecutionMetrics; // Optional for development
}

interface SystemResourceMetrics {
    cpu: {
        usage: number;
        cores: number;
        loadAvg: number[];
    };
    memory: {
        heapUsed: number;
        heapTotal: number;
        heapUsedPercent: number;
        rss: number;
        external: number;
        arrayBuffers: number;
    };
    disk?: {
        total: number;
        used: number;
        usagePercent: number;
    };
}

interface ApplicationMetrics {
    nodeVersion: string;
    pid: number;
    platform: string;
    environment: string;
    websockets: {
        connections: number;
        rooms: number;
    };
    queues: {
        [queueName: string]: {
            waiting: number;
            active: number;
            completed: number;
            failed: number;
            delayed: number;
            total: number;
        };
    };
    llmServices: {
        [serviceId: string]: {
            state: string;
            cooldownUntil?: number;
        };
    };
}

interface DatabaseMetrics {
    connected: boolean;
    poolSize?: number;
    // Add more database-specific metrics as needed
}

interface RedisMetrics {
    connected: boolean;
    memory?: {
        used: number;
        peak: number;
    };
    stats?: {
        connections: number;
        commands: number;
    };
}

interface ApiMetrics {
    requestsTotal?: number;
    responseTimes?: {
        min: number;
        max: number;
        avg: number;
    };
    // Add more API metrics as needed
}

interface ExecutionMetrics {
    routinesExecuting: number;
    strategiesUsed: {
        conversational: number;
        reasoning: number;
        deterministic: number;
    };
    averageExecutionTime: number;
    // Add more execution-specific metrics as needed
}

/**
 * Service for collecting and reporting system metrics
 */
export class MetricsService {
    private static instance: MetricsService;
    private startTime: number;

    private constructor() {
        this.startTime = Date.now();
    }

    /**
     * Get the singleton instance of MetricsService
     */
    public static get(): MetricsService {
        if (!MetricsService.instance) {
            MetricsService.instance = new MetricsService();
        }
        return MetricsService.instance;
    }

    /**
     * Get CPU usage percentage
     */
    private getCpuUsage(): number {
        const cpus = os.cpus();
        const totalCpu = cpus.reduce((acc, cpu) => {
            const total = Object.values(cpu.times).reduce((sum, time) => sum + time, 0);
            const idle = cpu.times.idle;
            return acc + ((total - idle) / total) * 100;
        }, 0);
        return Math.round(totalCpu / cpus.length);
    }

    /**
     * Get memory metrics
     */
    private getMemoryMetrics(): SystemResourceMetrics["memory"] {
        const usage = process.memoryUsage();
        return {
            heapUsed: usage.heapUsed,
            heapTotal: usage.heapTotal,
            heapUsedPercent: Math.round((usage.heapUsed / usage.heapTotal) * 100),
            rss: usage.rss,
            external: usage.external,
            arrayBuffers: usage.arrayBuffers || 0,
        };
    }

    /**
     * Get system resource metrics
     */
    private async getSystemMetrics(): Promise<SystemResourceMetrics> {
        const memory = this.getMemoryMetrics();
        const cpu = {
            usage: this.getCpuUsage(),
            cores: os.cpus().length,
            loadAvg: os.loadavg(),
        };

        // Disk metrics (optional, might fail in some environments)
        let disk: SystemResourceMetrics["disk"];
        try {
            const { exec } = await import("child_process");
            const { promisify } = await import("util");
            const execAsync = promisify(exec);
            const { stdout } = await execAsync("df / --output=size,used --block-size=1 | tail -1");
            const [size, used] = stdout.trim().split(/\s+/).map(Number);
            disk = {
                total: size,
                used,
                usagePercent: Math.round((used / size) * 100),
            };
        } catch (error) {
            // Disk metrics unavailable
            disk = undefined;
        }

        return { cpu, memory, disk };
    }

    /**
     * Get application-level metrics
     */
    private async getApplicationMetrics(): Promise<ApplicationMetrics> {
        // WebSocket metrics
        const socketService = SocketService.get();
        const socketDetails = socketService.getHealthDetails();
        const websockets = {
            connections: socketDetails?.connectedClients || 0,
            rooms: socketDetails?.activeRooms || 0,
        };

        // Queue metrics
        const queueService = QueueService.get();
        const queues: ApplicationMetrics["queues"] = {};

        // Only check queues that are already initialized
        for (const [name, queue] of Object.entries(queueService.queues)) {
            try {
                const health = await queue.checkHealth();
                queues[name] = {
                    waiting: health.jobCounts.waiting,
                    active: health.jobCounts.active,
                    completed: health.jobCounts.completed,
                    failed: health.jobCounts.failed,
                    delayed: health.jobCounts.delayed,
                    total: health.jobCounts.total,
                };
            } catch (error) {
                queues[name] = {
                    waiting: 0,
                    active: 0,
                    completed: 0,
                    failed: 0,
                    delayed: 0,
                    total: 0,
                };
            }
        }

        // LLM service metrics
        const registry = AIServiceRegistry.get();
        const services = registry["serviceStates"];
        const llmServices: ApplicationMetrics["llmServices"] = {};

        for (const [serviceId, state] of services.entries()) {
            llmServices[serviceId] = {
                state: state.state,
                cooldownUntil: state.cooldownUntil ? state.cooldownUntil.getTime() : undefined,
            };
        }

        return {
            nodeVersion: process.version,
            pid: process.pid,
            platform: os.platform(),
            environment: process.env.NODE_ENV || "unknown",
            websockets,
            queues,
            llmServices,
        };
    }

    /**
     * Get database metrics
     */
    private async getDatabaseMetrics(): Promise<DatabaseMetrics> {
        try {
            const connected = DbProvider.isConnected();
            if (!connected) {
                return { connected: false };
            }

            // Try a simple query to verify connection
            await DbProvider.get().$queryRaw`SELECT 1`;

            return {
                connected: true,
                // Add more database metrics here as needed
                // poolSize: DbProvider.getPoolSize(), // if available
            };
        } catch (error) {
            return { connected: false };
        }
    }

    /**
     * Get Redis metrics
     */
    private async getRedisMetrics(): Promise<RedisMetrics> {
        try {
            const redisClient = await CacheService.get().raw();
            await redisClient.ping();

            // Get Redis info
            const info = await redisClient.info();
            const lines = info.split("\r\n");
            const memoryUsed = lines.find(line => line.startsWith("used_memory:"))
                ?.split(":")[1];
            const memoryPeak = lines.find(line => line.startsWith("used_memory_peak:"))
                ?.split(":")[1];
            const connections = lines.find(line => line.startsWith("connected_clients:"))
                ?.split(":")[1];
            const commands = lines.find(line => line.startsWith("total_commands_processed:"))
                ?.split(":")[1];

            return {
                connected: true,
                memory: {
                    used: parseInt(memoryUsed || "0", 10),
                    peak: parseInt(memoryPeak || "0", 10),
                },
                stats: {
                    connections: parseInt(connections || "0", 10),
                    commands: parseInt(commands || "0", 10),
                },
            };
        } catch (error) {
            return { connected: false };
        }
    }

    /**
     * Get API metrics (placeholder for future implementation)
     */
    private getApiMetrics(): ApiMetrics {
        // Placeholder - would integrate with request tracking middleware
        return {
            // requestsTotal: RequestTracker.getTotalRequests(),
            // responseTimes: RequestTracker.getResponseTimes(),
        };
    }

    /**
     * Get execution metrics (development only)
     */
    private getExecutionMetrics(): ExecutionMetrics | undefined {
        if (!debug) {
            return undefined;
        }

        // Placeholder for execution architecture metrics
        return {
            routinesExecuting: 0, // Would track active routines
            strategiesUsed: {
                conversational: 0,
                reasoning: 0,
                deterministic: 0,
            },
            averageExecutionTime: 0,
        };
    }

    /**
     * Get comprehensive system metrics
     */
    public async getMetrics(): Promise<SystemMetrics> {
        const now = Date.now();
        const uptime = Math.floor((now - this.startTime) / SECONDS_1_MS);

        const [system, application, database, redis] = await Promise.all([
            this.getSystemMetrics(),
            this.getApplicationMetrics(),
            this.getDatabaseMetrics(),
            this.getRedisMetrics(),
        ]);

        const api = this.getApiMetrics();
        const execution = this.getExecutionMetrics();

        const metrics: SystemMetrics = {
            timestamp: now,
            uptime,
            system,
            application,
            database,
            redis,
            api,
        };

        if (execution) {
            metrics.execution = execution;
        }

        return metrics;
    }
}

/**
 * Setup the metrics endpoint
 */
export function setupMetrics(app: Express): void {
    app.get("/metrics", async (_req: Request, res: Response) => {
        const METRICS_TIMEOUT_MS = 10000; // 10 seconds (shorter than health check)

        const timeoutPromise = new Promise((_, reject) => {
            setTimeout(() => {
                reject(new Error("Metrics collection timed out"));
            }, METRICS_TIMEOUT_MS);
        });

        try {
            const metrics = await Promise.race([
                MetricsService.get().getMetrics(),
                timeoutPromise,
            ]) as SystemMetrics;

            res.status(HttpStatus.Ok).json(metrics);
        } catch (error) {
            let errorMessage = "Metrics collection failed";
            if (error instanceof Error && error.message === "Metrics collection timed out") {
                errorMessage = "Metrics collection timed out after " + (METRICS_TIMEOUT_MS / SECONDS_1_MS) + " seconds";
            } else if (error instanceof Error) {
                errorMessage = `Metrics collection failed: ${error.message}`;
            }

            // Log the error for server-side diagnostics
            logger.error(`🚨 ${errorMessage}`, { trace: "metrics-error", error });

            res.status(HttpStatus.ServiceUnavailable).json({
                error: errorMessage,
                timestamp: Date.now(),
            });
        }
    });
}
