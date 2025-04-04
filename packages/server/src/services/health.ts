import { DAYS_1_S, GB_1_BYTES, HttpStatus, MCPEndpoint, SERVER_VERSION, endpointsAuth, endpointsFeed } from "@local/shared";
import { exec as execCb } from "child_process";
import { Express } from "express";
import os from "os";
import Stripe from "stripe";
import { promisify } from "util";
import { DbProvider } from "../db/provider.js";
import { initializeRedis } from "../redisConn.js";
import { API_URL, MCP_SITE_WIDE_URL, SERVER_URL } from "../server.js";
import { io } from "../sockets/io.js";
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
        api: ServiceHealth;
        cronJobs: ServiceHealth;
        database: ServiceHealth;
        llm: {
            [key: string]: ServiceHealth;
        };
        mcp: ServiceHealth;
        memory: ServiceHealth;
        queues: {
            [key: string]: ServiceHealth;
        };
        redis: ServiceHealth;
        ssl: ServiceHealth;
        stripe: ServiceHealth;
        system: ServiceHealth;
        websocket: ServiceHealth;
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

// HTTP Status ranges
// eslint-disable-next-line no-magic-numbers
const HTTP_STATUS_OK_MIN = 200;
// eslint-disable-next-line no-magic-numbers
const HTTP_STATUS_OK_MAX = 300;

// Cron job monitoring settings
const CRON_MAX_FAILURE_AGE_MS = DAYS_1_S; // 24 hours
const CRON_WARNING_THRESHOLD = 1; // Number of failures before status is degraded

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
     * Check SSL certificate health by attempting an HTTPS connection
     */
    private async checkSslCertificate(): Promise<ServiceHealth> {
        try {
            const url = new URL(SERVER_URL);
            if (!url.protocol.startsWith("https")) {
                return {
                    healthy: true,
                    status: ServiceStatus.Operational,
                    lastChecked: Date.now(),
                    details: { info: "SSL check skipped - not using HTTPS" },
                };
            }

            const response = await fetch(SERVER_URL, {
                method: "HEAD",  // Only get headers, don't download body
                // Don't follow redirects to ensure we're checking the actual certificate
                redirect: "manual",
            });

            // Check if response is successful (2xx status code)
            const healthy = response.ok;
            return {
                healthy,
                status: healthy ? ServiceStatus.Operational : ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    protocol: url.protocol,
                    host: url.host,
                    statusCode: response.status,
                },
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to verify HTTPS connection" },
            };
        }
    }

    /**
     * Check MCP service health by testing core endpoints
     */
    private async checkMcp(): Promise<ServiceHealth> {
        try {
            // Build MCP endpoint URL
            const listToolsUrl = `${MCP_SITE_WIDE_URL}${MCPEndpoint.ListTools}`;

            const response = await fetch(listToolsUrl);

            if (!response.ok) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: {
                        error: "MCP service not responding",
                        statusCode: response.status,
                    },
                };
            }

            const data = await response.json();
            return {
                healthy: true,
                status: ServiceStatus.Operational,
                lastChecked: Date.now(),
                details: {
                    endpoints: {
                        listTools: true,
                        execute: true,
                        search: true,
                        register: true,
                    },
                    toolCount: data.tools?.length ?? 0,
                },
            };
        } catch (error) {
            console.error(`[HealthCheck] Error in checkMcp: ${(error as Error).message}`);
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to check MCP service status" },
            };
        }
    }

    /**
     * Check API health by testing core endpoints
     */
    private async checkApi(): Promise<ServiceHealth> {
        try {
            // Define critical API endpoints to test using the pairs objects
            const endpointsToTest = [
                {
                    name: "popular",
                    ...endpointsFeed.popular,
                },
                {
                    name: "validateSession",
                    ...endpointsAuth.validateSession,
                },
            ];

            const results = await Promise.all(
                endpointsToTest.map(async (endpoint) => {
                    const url = `${API_URL}/${SERVER_VERSION}/rest${endpoint.endpoint}`;
                    try {
                        const options = {
                            method: endpoint.method,
                            headers: {
                                "Content-Type": "application/json",
                            },
                            // Properly format the input for POST requests
                            body: endpoint.method === "POST" ? JSON.stringify({ input: {} }) : undefined,
                        };

                        const response = await fetch(url, options);
                        return {
                            name: endpoint.name,
                            endpoint: endpoint.endpoint,
                            method: endpoint.method,
                            status: response.status,
                            ok: response.status >= HTTP_STATUS_OK_MIN && response.status < HTTP_STATUS_OK_MAX,
                        };
                    } catch (error) {
                        console.error(`[HealthCheck] Error checking API endpoint '${endpoint.name}': ${(error as Error).message}`);
                        return {
                            name: endpoint.name,
                            endpoint: endpoint.endpoint,
                            method: endpoint.method,
                            status: 0,
                            ok: false,
                            error: (error as Error).message,
                        };
                    }
                }),
            );

            // Calculate how many endpoints are working
            const workingEndpoints = results.filter(r => r.ok).length;
            const totalEndpoints = endpointsToTest.length;

            // Determine overall API status
            let status = ServiceStatus.Operational;
            if (workingEndpoints === 0) {
                status = ServiceStatus.Down;
            } else if (workingEndpoints < totalEndpoints) {
                status = ServiceStatus.Degraded;
            }

            return {
                healthy: status === ServiceStatus.Operational,
                status,
                lastChecked: Date.now(),
                details: {
                    endpoints: results,
                    working: workingEndpoints,
                    total: totalEndpoints,
                },
            };
        } catch (error) {
            console.error(`[HealthCheck] Error in checkApi: ${(error as Error).message}`);
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to check API health" },
            };
        }
    }

    /**
     * Check WebSocket server health
     */
    private checkWebSocket(): ServiceHealth {
        try {
            // Check if Socket.IO server is initialized and running
            const isRunning = !!(io && io.engine && io.sockets && io.sockets.adapter);

            return {
                healthy: isRunning,
                status: isRunning ? ServiceStatus.Operational : ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    connectedClients: io.engine?.clientsCount ?? 0,
                    activeRooms: io.sockets?.adapter?.rooms?.size ?? 0,
                    namespaces: io.sockets?.adapter?.nsp?.size ?? 0,
                },
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to check WebSocket server status" },
            };
        }
    }

    /**
     * Check cron job health
     * 
     * This method uses Redis to track cron job execution history.
     * Each cron job records its success/failure to Redis.
     */
    private async checkCronJobs(): Promise<ServiceHealth> {
        try {
            const redis = await initializeRedis();
            if (!redis) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: { error: "Redis client not initialized" },
                };
            }

            // Get all registered cron jobs
            const jobNames = await redis.sMembers("cron:jobs");

            if (!jobNames.length) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: { error: "No cron jobs registered in Redis" },
                };
            }

            const now = Date.now();
            let hasFailures = false;
            let hasCriticalFailures = false;
            const jobDetails: Record<string, any> = {};

            // Check each job's status
            for (const jobName of jobNames) {
                // Get the last success and failure times
                const [lastSuccessStr, lastFailureStr, failureKeys] = await Promise.all([
                    redis.get(`cron:lastSuccess:${jobName}`),
                    redis.get(`cron:lastFailure:${jobName}`),
                    redis.keys(`cron:failures:${jobName}:*`),
                ]);

                const lastSuccess = lastSuccessStr ? parseInt(lastSuccessStr, 10) : 0;
                const lastFailure = lastFailureStr ? parseInt(lastFailureStr, 10) : 0;
                const recentFailures = failureKeys.length;

                const jobStatus = {
                    lastSuccess,
                    lastFailure,
                    recentFailures,
                    status: ServiceStatus.Operational,
                };

                // Check if the job has recent failures
                if (recentFailures >= CRON_WARNING_THRESHOLD) {
                    jobStatus.status = ServiceStatus.Degraded;
                    hasFailures = true;
                }

                // Check if the job has had no successful runs or has recent failures
                if (lastSuccess === 0 || (lastFailure > lastSuccess && (now - lastFailure) < CRON_MAX_FAILURE_AGE_MS)) {
                    jobStatus.status = ServiceStatus.Down;
                    hasCriticalFailures = true;
                }

                jobDetails[jobName] = jobStatus;
            }

            // Determine overall status
            const status = hasCriticalFailures ? ServiceStatus.Down :
                hasFailures ? ServiceStatus.Degraded : ServiceStatus.Operational;

            return {
                healthy: status === ServiceStatus.Operational,
                status,
                lastChecked: now,
                details: {
                    jobs: jobDetails,
                    total: jobNames.length,
                },
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: "Failed to check cron job health" },
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
        const [
            apiHealth,
            cronJobsHealth,
            dbHealth,
            llmHealth,
            mcpHealth,
            memoryHealth,
            queueHealths,
            redisHealth,
            sslHealth,
            stripeHealth,
            systemHealth,
            websocketHealth,
        ] = await Promise.all([
            this.checkApi(),
            this.checkCronJobs(),
            this.checkDatabase(),
            this.checkLlmServices(),
            this.checkMcp(),
            this.checkMemory(),
            this.checkQueues(),
            this.checkRedis(),
            this.checkSslCertificate(),
            this.checkStripe(),
            this.checkSystem(),
            this.checkWebSocket(),
        ]);

        // Determine overall status
        const allServices = [
            apiHealth,
            cronJobsHealth,
            dbHealth,
            mcpHealth,
            memoryHealth,
            redisHealth,
            sslHealth,
            stripeHealth,
            systemHealth,
            websocketHealth,
            ...Object.values(queueHealths),
            ...Object.values(llmHealth),
        ];
        const status = allServices.every(s => s.status === ServiceStatus.Operational) ? ServiceStatus.Operational :
            allServices.some(s => s.status === ServiceStatus.Down) ? ServiceStatus.Down : ServiceStatus.Degraded;

        const health: SystemHealth = {
            status,
            version: process.env.npm_package_version || "unknown",
            services: {
                api: apiHealth,
                cronJobs: cronJobsHealth,
                database: dbHealth,
                llm: llmHealth,
                mcp: mcpHealth,
                memory: memoryHealth,
                queues: queueHealths,
                redis: redisHealth,
                ssl: sslHealth,
                stripe: stripeHealth,
                system: systemHealth,
                websocket: websocketHealth,
            },
            timestamp: Date.now(),
        };

        this.lastCheck = health;
        return health;
    }
}

/**
 * Setup the health check endpoint.
 */
export function setupHealthCheck(app: Express): void {
    app.get("/healthcheck", async (_req, res) => {
        try {
            const health = await HealthService.get().getHealth();
            res.status(HttpStatus.Ok).json(health);
        } catch (error) {
            res.status(HttpStatus.ServiceUnavailable).json({
                status: ServiceStatus.Down,
                version: process.env.npm_package_version || "unknown",
                error: "Health check failed",
            });
        }
    });
}
