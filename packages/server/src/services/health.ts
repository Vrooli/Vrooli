import { GB_1_BYTES } from "@local/shared";
import { exec as execCb } from "child_process";
import os from "os";
import Stripe from "stripe";
import { promisify } from "util";
import { DbProvider } from "../db/provider.js";
import { initializeRedis } from "../redisConn.js";
import { QueueStatus } from "../tasks/base/queue.js";
import { emailQueue } from "../tasks/email/queue.js";
import { exportQueue } from "../tasks/export/queue.js";
import { importQueue } from "../tasks/import/queue.js";
import { llmQueue } from "../tasks/llm/queue.js";
import { LlmServiceRegistry, LlmServiceState } from "../tasks/llm/registry.js";
import { llmTaskQueue } from "../tasks/llmTask/queue.js";
import { pushQueue } from "../tasks/push/queue.js";
import { runQueue } from "../tasks/run/queue.js";
import { sandboxQueue } from "../tasks/sandbox/queue.js";
import { smsQueue } from "../tasks/sms/queue.js";

const exec = promisify(execCb);

export enum ServiceStatus {
    Operational = "Operational",
    Degraded = "Degraded",
    Down = "Down",
}

interface ServiceHealth {
    healthy: boolean;
    status: ServiceStatus;
    lastChecked: number;
    details?: Record<string, unknown>;
}

interface SystemHealth {
    status: ServiceStatus;
    version: string;
    services: {
        database: ServiceHealth;
        redis: ServiceHealth;
        queues: {
            [key: string]: ServiceHealth;
        };
        memory: ServiceHealth;
        system: ServiceHealth;
        llm: {
            [key: string]: ServiceHealth;
        };
        stripe: ServiceHealth;
    };
    timestamp: number;
}

const CACHE_TIMEOUT_MS = 30_000;

// Memory thresholds in bytes (80% and 90% of default Node.js heap)
// eslint-disable-next-line no-magic-numbers
const MEMORY_WARNING_THRESHOLD = 1.2 * GB_1_BYTES;
// eslint-disable-next-line no-magic-numbers
const MEMORY_CRITICAL_THRESHOLD = 1.4 * GB_1_BYTES;

// System thresholds
// eslint-disable-next-line no-magic-numbers
const CPU_WARNING_THRESHOLD = 80; // 80% CPU usage
// eslint-disable-next-line no-magic-numbers
const CPU_CRITICAL_THRESHOLD = 90; // 90% CPU usage
// eslint-disable-next-line no-magic-numbers
const DISK_WARNING_THRESHOLD = 85; // 85% disk usage
// eslint-disable-next-line no-magic-numbers
const DISK_CRITICAL_THRESHOLD = 95; // 95% disk usage

/**
 * Service for checking and reporting system health status
 */
export class HealthService {
    private static instance: HealthService;
    private lastCheck: SystemHealth | null = null;
    private readonly cacheTimeout = CACHE_TIMEOUT_MS;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Get the singleton instance of HealthService
     */
    public static get(): HealthService {
        if (!HealthService.instance) {
            HealthService.instance = new HealthService();
        }
        return HealthService.instance;
    }

    /**
     * Check if a cached health check is still valid
     */
    private isCacheValid(): boolean {
        if (!this.lastCheck) return false;
        return Date.now() - this.lastCheck.timestamp < this.cacheTimeout;
    }

    /**
     * Check database health
     */
    private async checkDatabase(): Promise<ServiceHealth> {
        try {
            await DbProvider.get().$queryRaw`SELECT 1`;
            return {
                healthy: true,
                status: ServiceStatus.Operational,
                lastChecked: Date.now(),
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
            };
        }
    }

    /**
     * Check Redis health
     */
    private async checkRedis(): Promise<ServiceHealth> {
        try {
            const redisClient = await initializeRedis();
            if (!redisClient) throw new Error("Redis client not initialized");
            await redisClient.ping();
            return {
                healthy: true,
                status: ServiceStatus.Operational,
                lastChecked: Date.now(),
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
            };
        }
    }

    /**
     * Check queue health
     */
    private async checkQueues(): Promise<{ [key: string]: ServiceHealth }> {
        const queues = {
            email: emailQueue,
            export: exportQueue,
            import: importQueue,
            llm: llmQueue,
            llmTask: llmTaskQueue,
            push: pushQueue,
            run: runQueue,
            sandbox: sandboxQueue,
            sms: smsQueue,
        };

        const queueHealth: { [key: string]: ServiceHealth } = {};

        for (const [name, queue] of Object.entries(queues)) {
            const status = await queue.checkHealth();
            queueHealth[name] = {
                healthy: status === QueueStatus.Healthy,
                status: status === QueueStatus.Healthy ? ServiceStatus.Operational :
                    status === QueueStatus.Degraded ? ServiceStatus.Degraded : ServiceStatus.Down,
                lastChecked: Date.now(),
            };
        }

        return queueHealth;
    }

    /**
     * Check memory usage health
     */
    private checkMemory(): ServiceHealth {
        const used = process.memoryUsage();
        const heapUsedPercent = (used.heapUsed / used.heapTotal) * 100;

        const status = used.heapUsed > MEMORY_CRITICAL_THRESHOLD ? ServiceStatus.Down :
            used.heapUsed > MEMORY_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                ServiceStatus.Operational;

        return {
            healthy: status === ServiceStatus.Operational,
            status,
            lastChecked: Date.now(),
            details: {
                heapUsed: used.heapUsed,
                heapTotal: used.heapTotal,
                heapUsedPercent: Math.round(heapUsedPercent),
                rss: used.rss,
                external: used.external,
            },
        };
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

    private async getDiskSpace(): Promise<{ size: number; used: number }> {
        const { stdout } = await exec("df / --output=size,used --block-size=1 | tail -1");
        const [size, used] = stdout.trim().split(/\s+/).map(Number);
        return { size, used };
    }

    /**
     * Check system resources health (CPU and Disk)
     */
    private async checkSystem(): Promise<ServiceHealth> {
        try {
            // Get CPU usage
            const cpuUsage = this.getCpuUsage();

            // Get disk space
            const { size, used } = await this.getDiskSpace();
            const diskUsagePercent = Math.round((used / size) * 100);

            // Determine status
            const status = cpuUsage > CPU_CRITICAL_THRESHOLD || diskUsagePercent > DISK_CRITICAL_THRESHOLD ? ServiceStatus.Down :
                cpuUsage > CPU_WARNING_THRESHOLD || diskUsagePercent > DISK_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                    ServiceStatus.Operational;

            return {
                healthy: status === ServiceStatus.Operational,
                status,
                lastChecked: Date.now(),
                details: {
                    cpu: {
                        usage: cpuUsage,
                        cores: os.cpus().length,
                    },
                    disk: {
                        total: size,
                        used,
                        usagePercent: diskUsagePercent,
                    },
                    uptime: os.uptime(),
                    loadAvg: os.loadavg(),
                },
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to check system resources" },
            };
        }
    }

    /**
     * Check LLM services health
     */
    private checkLlmServices(): { [key: string]: ServiceHealth } {
        const registry = LlmServiceRegistry.get();
        const services = registry["serviceStates"];
        const llmHealth: { [key: string]: ServiceHealth } = {};

        for (const [serviceId, state] of services.entries()) {
            llmHealth[serviceId] = {
                healthy: state.state === LlmServiceState.Active,
                status: state.state === LlmServiceState.Active ? ServiceStatus.Operational :
                    state.state === LlmServiceState.Cooldown ? ServiceStatus.Degraded :
                        ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    state: state.state,
                    cooldownUntil: state.cooldownUntil,
                },
            };
        }

        return llmHealth;
    }

    /**
     * Check Stripe API health
     */
    private async checkStripe(): Promise<ServiceHealth> {
        try {
            if (!process.env.STRIPE_SECRET_KEY) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: { error: "Stripe secret key not configured" },
                };
            }

            const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
                apiVersion: "2022-11-15",
            });

            // Make a simple API call that doesn't affect any data
            await stripe.balance.retrieve();

            return {
                healthy: true,
                status: ServiceStatus.Operational,
                lastChecked: Date.now(),
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to connect to Stripe API" },
            };
        }
    }

    /**
     * Get overall system health status
     */
    public async getHealth(): Promise<SystemHealth> {
        // Return cached result if valid
        if (this.isCacheValid() && this.lastCheck) {
            return this.lastCheck;
        }

        // Perform health checks
        const [dbHealth, redisHealth, queueHealths, systemHealth, stripeHealth] = await Promise.all([
            this.checkDatabase(),
            this.checkRedis(),
            this.checkQueues(),
            this.checkSystem(),
            this.checkStripe(),
        ]);
        const memoryHealth = this.checkMemory();
        const llmHealth = this.checkLlmServices();

        // Determine overall status
        const allServices = [
            dbHealth,
            redisHealth,
            memoryHealth,
            systemHealth,
            stripeHealth,
            ...Object.values(queueHealths),
            ...Object.values(llmHealth),
        ];
        const status = allServices.every(s => s.status === ServiceStatus.Operational) ? ServiceStatus.Operational :
            allServices.some(s => s.status === ServiceStatus.Down) ? ServiceStatus.Down : ServiceStatus.Degraded;

        const health: SystemHealth = {
            status,
            version: process.env.npm_package_version || "unknown",
            services: {
                database: dbHealth,
                redis: redisHealth,
                queues: queueHealths,
                memory: memoryHealth,
                system: systemHealth,
                llm: llmHealth,
                stripe: stripeHealth,
            },
            timestamp: Date.now(),
        };

        this.lastCheck = health;
        return health;
    }
} 
