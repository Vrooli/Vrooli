import { HeadBucketCommand, S3Client } from "@aws-sdk/client-s3";
import { DAYS_1_S, GB_1_BYTES, HOURS_1_MS, HttpStatus, MINUTES_15_MS, MINUTES_1_MS, MINUTES_2_S, MINUTES_30_MS, SECONDS_1_MS, SERVER_VERSION, endpointsAuth, endpointsFeed } from "@local/shared";
import { exec as execCb } from "child_process";
import { Express } from "express";
import fs from "fs/promises";
import i18next from "i18next";
import os from "os";
import path from "path";
import Stripe from "stripe";
import { fileURLToPath } from "url";
import { promisify } from "util";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { Notify } from "../notify/notify.js";
import { initializeRedis } from "../redisConn.js";
import { API_URL, SERVER_URL } from "../server.js";
import { SocketService } from "../sockets/io.js";
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
import { checkNSFW, getS3Client } from "../utils/fileStorage.js";
import { BusService, RedisStreamBus } from "./bus.js";
import { EmbeddingService } from "./embedding.js";
import { getMcpServer } from "./mcp/index.js";
import { McpToolName } from "./mcp/registry.js";

const exec = promisify(execCb);

const debug = process.env.NODE_ENV === "development";

export enum ServiceStatus {
    Operational = "Operational",
    Degraded = "Degraded",
    Down = "Down",
}

export interface ServiceHealth {
    healthy: boolean;
    status: ServiceStatus | `${ServiceStatus}`;
    lastChecked: number;
    details?: object;
}

interface SystemHealth {
    status: ServiceStatus;
    version: string;
    services: {
        api: ServiceHealth;
        bus: ServiceHealth;
        cronJobs: ServiceHealth;
        database: ServiceHealth;
        i18n: ServiceHealth;
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
        imageStorage: ServiceHealth;
        embeddingService: ServiceHealth;
    };
    timestamp: number;
}

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

// Define a test routine ID (replace with actual seeded ID when available)
const MCP_TEST_ROUTINE_ID = "c9dd779d-ebf2-4e65-8429-4eef5c40aa4a"; // Daily Standup routine

// Per-service cache durations
const DEFAULT_SERVICE_CACHE_MS = MINUTES_2_S & SECONDS_1_MS; // 2 minutes
const EMBEDDING_SERVICE_CACHE_MS = HOURS_1_MS; // 1 hour
const STRIPE_SERVICE_CACHE_MS = MINUTES_30_MS; // 30 minutes
const LLM_SERVICE_CACHE_MS = MINUTES_15_MS; // 15 minutes
const API_SERVICE_CACHE_MS = MINUTES_1_MS; // 1 minute
// eslint-disable-next-line no-magic-numbers
const SSL_SERVICE_CACHE_MS = HOURS_1_MS * 6; // 6 hours

/**
 * Service for checking and reporting system health status
 */
export class HealthService {
    private static instance: HealthService;

    // Cache properties for individual services
    private apiHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private busHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private cronJobsHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private databaseHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private i18nHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private llmServicesCache: { health: { [key: string]: ServiceHealth }, expiresAt: number } | null = null;
    private mcpHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private memoryHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private queuesHealthCache: { health: { [key: string]: ServiceHealth }, expiresAt: number } | null = null;
    private redisHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private sslHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private stripeHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private systemHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private websocketHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private imageStorageHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;
    private embeddingServiceHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;

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
     * Helper function to get cached health data if it's still valid.
     * @param cacheEntry The cache entry to check.
     * @returns The cached ServiceHealth if valid, otherwise null.
     */
    private getCachedHealthIfValid(cacheEntry: { health: ServiceHealth, expiresAt: number } | null): ServiceHealth | null {
        if (cacheEntry && Date.now() < cacheEntry.expiresAt) {
            return cacheEntry.health;
        }
        return null;
    }

    /**
     * Helper function to create a service cache
     */
    private createServiceCache(status: ServiceStatus | `${ServiceStatus}`, cacheDurationMs: number, details?: object) {
        const health: ServiceHealth = {
            healthy: status === ServiceStatus.Operational,
            status,
            lastChecked: Date.now(),
            details,
        };
        return { health, expiresAt: Date.now() + cacheDurationMs };
    }

    private async checkBus(): Promise<ServiceHealth> {
        const cached = this.getCachedHealthIfValid(this.busHealthCache);
        if (cached) return cached;

        try {
            // Ensure bus exists (lazy init is idempotent)
            await BusService.get().startEventBus();
            const bus = BusService.get().getBus();

            // Default when no metrics() helper (e.g. InMemoryEventBus in tests)
            let metrics: any = { connected: true };
            if (typeof (bus as any).metrics === "function") {
                metrics = await (bus as RedisStreamBus).metrics();
            }

            const backlog = Object.values(metrics.pendingPerCG ?? {}).reduce((a, b) => a + b, 0) as number;
            const status =
                !metrics.connected ? ServiceStatus.Down
                    : backlog > 5000 ? ServiceStatus.Degraded   // tune threshold
                        : ServiceStatus.Operational;

            this.busHealthCache = this.createServiceCache(status, DEFAULT_SERVICE_CACHE_MS, metrics);
            return this.busHealthCache.health;

        } catch (err) {
            this.busHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: (err as Error).message ?? "Bus metrics failed" },
            );
            return this.busHealthCache.health;
        }
    }

    /**
     * Check database health
     */
    private async checkDatabase(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.databaseHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }

        try {
            // First check if we're connected at all
            if (!DbProvider.isConnected()) {
                this.databaseHealthCache = this.createServiceCache(ServiceStatus.Down, DEFAULT_SERVICE_CACHE_MS, {
                    connection: false,
                    message: "Database not connected",
                });
                return this.databaseHealthCache.health;
            }

            // Then try a simple query to verify connection is still active
            try {
                await DbProvider.get().$queryRaw`SELECT 1`;
            } catch (queryError) {
                this.databaseHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    {
                        connection: false,
                        message: "Database connection lost",
                        error: (queryError as Error).message,
                    });
                return this.databaseHealthCache.health;
            }

            // Check if seeding was successful
            const seedingSuccessful = DbProvider.isSeedingSuccessful();
            const isRetrying = DbProvider.isRetryingSeeding();
            const retryCount = DbProvider.getSeedRetryCount();
            const attemptCount = DbProvider.getSeedAttemptCount();
            const maxRetries = DbProvider.getMaxRetries();

            // If connection works but seeding failed
            if (!seedingSuccessful) {
                this.databaseHealthCache = this.createServiceCache(
                    isRetrying ? ServiceStatus.Degraded : ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    {
                        connection: true,
                        seeding: false,
                        isRetrying,
                        retryCount,
                        attemptCount,
                        maxRetries,
                        remainingRetries: isRetrying ? maxRetries - retryCount : 0,
                        message: isRetrying
                            ? `Database connected but seeding failed, retry ${retryCount} of ${maxRetries} in progress`
                            : `Database connected but seeding failed after ${attemptCount} attempts (max retries: ${maxRetries})`,
                    });
                return this.databaseHealthCache.health;
            }

            this.databaseHealthCache = this.createServiceCache(
                ServiceStatus.Operational,
                DEFAULT_SERVICE_CACHE_MS,
                {
                    connection: true,
                    seeding: true,
                    attemptCount,
                },
            );
            return this.databaseHealthCache.health;
        } catch (error) {
            this.databaseHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                {
                    connection: false,
                    message: "Failed to check database health",
                    error: (error as Error).message,
                },
            );
            return this.databaseHealthCache.health;
        }
    }

    /**
     * Check Redis health
     */
    private async checkRedis(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.redisHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }

        try {
            const redisClient = await initializeRedis();
            if (!redisClient) throw new Error("Redis client not initialized");
            await redisClient.ping();
            this.redisHealthCache = this.createServiceCache(
                ServiceStatus.Operational,
                DEFAULT_SERVICE_CACHE_MS,
            );
            return this.redisHealthCache.health;
        } catch (error) {
            this.redisHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                {
                    error: (error as Error).message,
                },
            );
            return this.redisHealthCache.health;
        }
    }

    /**
     * Check queue health
     */
    private async checkQueues(): Promise<{ [key: string]: ServiceHealth }> {
        if (this.queuesHealthCache && Date.now() < this.queuesHealthCache.expiresAt) {
            return this.queuesHealthCache.health;
        }
        const now = Date.now();

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
            const healthDetails = await queue.checkHealth();
            queueHealth[name] = {
                healthy: healthDetails.status === QueueStatus.Healthy,
                status: healthDetails.status === QueueStatus.Healthy ? ServiceStatus.Operational :
                    healthDetails.status === QueueStatus.Degraded ? ServiceStatus.Degraded : ServiceStatus.Down,
                lastChecked: now,
                details: {
                    counts: healthDetails.jobCounts,
                    activeJobs: healthDetails.activeJobs?.map(job => ({
                        id: job.id,
                        name: job.name,
                        duration: job.duration,
                        durationSeconds: Math.round(job.duration / SECONDS_1_MS),
                        startedAt: job.processedOn,
                    })),
                    metrics: {
                        queueLength: healthDetails.jobCounts.waiting + healthDetails.jobCounts.delayed,
                        activeCount: healthDetails.jobCounts.active,
                        failedCount: healthDetails.jobCounts.failed,
                        completedCount: healthDetails.jobCounts.completed,
                        totalJobs: healthDetails.jobCounts.total,
                    },
                },
            };
        }

        this.queuesHealthCache = { health: queueHealth, expiresAt: now + DEFAULT_SERVICE_CACHE_MS };
        return queueHealth;
    }

    /**
     * Check memory usage health
     */
    private checkMemory(): ServiceHealth {
        const cachedHealth = this.getCachedHealthIfValid(this.memoryHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        const used = process.memoryUsage();
        const heapUsedPercent = (used.heapUsed / used.heapTotal) * 100;
        const memoryStatus = used.heapUsed > MEMORY_CRITICAL_THRESHOLD ? ServiceStatus.Down :
            used.heapUsed > MEMORY_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                ServiceStatus.Operational;

        this.memoryHealthCache = this.createServiceCache(
            memoryStatus,
            DEFAULT_SERVICE_CACHE_MS,
            {
                heapUsed: used.heapUsed,
                heapTotal: used.heapTotal,
                heapUsedPercent: Math.round(heapUsedPercent),
                rss: used.rss,
                external: used.external,
            },
        );
        return this.memoryHealthCache.health;
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
        const cachedHealth = this.getCachedHealthIfValid(this.systemHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            // Get CPU usage
            const cpuUsage = this.getCpuUsage();

            // Get disk space
            const { size, used } = await this.getDiskSpace();
            const diskUsagePercent = Math.round((used / size) * 100);

            // Determine status
            const systemStatus = cpuUsage > CPU_CRITICAL_THRESHOLD || diskUsagePercent > DISK_CRITICAL_THRESHOLD ? ServiceStatus.Down :
                cpuUsage > CPU_WARNING_THRESHOLD || diskUsagePercent > DISK_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                    ServiceStatus.Operational;

            this.systemHealthCache = this.createServiceCache(
                systemStatus,
                DEFAULT_SERVICE_CACHE_MS,
                {
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
            );
            return this.systemHealthCache.health;
        } catch (error) {
            this.systemHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: "Failed to check system resources" },
            );
            return this.systemHealthCache.health;
        }
    }

    /**
     * Check LLM services health
     */
    private checkLlmServices(): { [key: string]: ServiceHealth } {
        if (this.llmServicesCache && Date.now() < this.llmServicesCache.expiresAt) {
            return this.llmServicesCache.health;
        }

        const now = Date.now();
        const registry = LlmServiceRegistry.get();
        const services = registry["serviceStates"];
        const llmHealth: { [key: string]: ServiceHealth } = {};

        for (const [serviceId, state] of services.entries()) {
            llmHealth[serviceId] = {
                healthy: state.state === LlmServiceState.Active,
                status: state.state === LlmServiceState.Active ? ServiceStatus.Operational :
                    state.state === LlmServiceState.Cooldown ? ServiceStatus.Degraded :
                        ServiceStatus.Down,
                lastChecked: now,
                details: {
                    state: state.state,
                    cooldownUntil: state.cooldownUntil,
                },
            };
        }
        this.llmServicesCache = { health: llmHealth, expiresAt: now + LLM_SERVICE_CACHE_MS };
        return llmHealth;
    }

    /**
     * Check Stripe API health
     */
    private async checkStripe(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.stripeHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            if (!process.env.STRIPE_SECRET_KEY) {
                this.stripeHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    STRIPE_SERVICE_CACHE_MS,
                    { error: "Stripe secret key not configured" },
                );
                return this.stripeHealthCache.health;
            }

            const stripe = new Stripe(process.env.STRIPE_SECRET_KEY, {
                apiVersion: "2022-11-15",
            });

            // Make a simple API call that doesn't affect any data
            await stripe.balance.retrieve();

            this.stripeHealthCache = this.createServiceCache(
                ServiceStatus.Operational,
                STRIPE_SERVICE_CACHE_MS,
            );
            return this.stripeHealthCache.health;
        } catch (error) {
            this.stripeHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                STRIPE_SERVICE_CACHE_MS,
                { error: "Failed to connect to Stripe API" },
            );
            return this.stripeHealthCache.health;
        }
    }

    /**
     * Check SSL certificate health by attempting an HTTPS connection
     */
    private async checkSslCertificate(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.sslHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            const url = new URL(SERVER_URL);
            if (!url.protocol.startsWith("https")) {
                this.sslHealthCache = this.createServiceCache(
                    ServiceStatus.Operational,
                    SSL_SERVICE_CACHE_MS,
                    { info: "SSL check skipped - not using HTTPS" },
                );
                return this.sslHealthCache.health;
            }

            const response = await fetch(SERVER_URL, {
                method: "HEAD",  // Only get headers, don't download body
                // Don't follow redirects to ensure we're checking the actual certificate
                redirect: "manual",
            });

            this.sslHealthCache = this.createServiceCache(
                response.ok ? ServiceStatus.Operational : ServiceStatus.Down,
                SSL_SERVICE_CACHE_MS,
                { protocol: url.protocol, host: url.host, statusCode: response.status },
            );
            return this.sslHealthCache.health;
        } catch (error) {
            this.sslHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                SSL_SERVICE_CACHE_MS,
                { error: "Failed to verify HTTPS connection" },
            );
            return this.sslHealthCache.health;
        }
    }

    /**
     * Check MCP service health by performing an end-to-end transport + RPC test
     */
    private async checkMcp(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.mcpHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        let transportHealthy = false;
        let builtInHealthy = false;
        let routineHealthy = false;
        let transportDetails: Record<string, unknown> = {};
        let builtInDetails: Record<string, unknown> = {};
        let routineDetails: Record<string, unknown> = {};

        try {
            // 1. Check Transport Layer via internal TransportManager (bypass HTTP/runtime permissions)
            const mcpApp = getMcpServer();
            const transportInfo = mcpApp.getTransportManager().getHealthInfo();
            transportHealthy = typeof transportInfo.activeConnections === "number" && transportInfo.activeConnections >= 0;
            transportDetails = transportInfo;

            // 2. Built-in tools via registry
            const toolRegistry = mcpApp.getToolRegistry();
            const findResult = await toolRegistry.execute(McpToolName.FindResources, { resource_type: "Routine" });
            builtInHealthy = Array.isArray(findResult.content) && findResult.content.length > 0;
            builtInDetails = { result: findResult };

            // 3. Routine execution via registry
            const runResult = await toolRegistry.execute(McpToolName.RunRoutine, { id: MCP_TEST_ROUTINE_ID, replacement: "healthcheck" });
            routineHealthy = Array.isArray(runResult.content) && runResult.content.length > 0;
            routineDetails = { result: runResult };
        } catch (err) {
            transportHealthy = false;
            transportDetails = { error: (err as Error).message };
            builtInHealthy = false;
            builtInDetails = { error: "Registry execution skipped" };
            routineHealthy = false;
            routineDetails = { error: "Registry execution skipped" };
        }

        // Determine the overall MCP status
        let overallStatus: ServiceStatus;
        if (transportHealthy && builtInHealthy && routineHealthy) {
            overallStatus = ServiceStatus.Operational;
        } else if (transportHealthy) {
            overallStatus = ServiceStatus.Degraded;
        } else {
            overallStatus = ServiceStatus.Down;
        }

        this.mcpHealthCache = this.createServiceCache(
            overallStatus === ServiceStatus.Operational ? ServiceStatus.Operational : ServiceStatus.Down,
            DEFAULT_SERVICE_CACHE_MS,
            {
                transport: { healthy: transportHealthy, ...transportDetails },
                builtInTools: { healthy: builtInHealthy, ...builtInDetails },
                routineTools: { healthy: routineHealthy, ...routineDetails },
            },
        );
        return this.mcpHealthCache.health;
    }

    /**
     * Check API health by testing core endpoints
     */
    private async checkApi(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.apiHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
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
                    const url = `${API_URL}/${SERVER_VERSION}${endpoint.endpoint}`;
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
            let apiStatus = ServiceStatus.Operational;
            if (workingEndpoints === 0) {
                apiStatus = ServiceStatus.Down;
            } else if (workingEndpoints < totalEndpoints) {
                apiStatus = ServiceStatus.Degraded;
            }

            this.apiHealthCache = this.createServiceCache(
                apiStatus,
                API_SERVICE_CACHE_MS,
                { endpoints: results, working: workingEndpoints, total: totalEndpoints },
            );
            return this.apiHealthCache.health;
        } catch (error) {
            console.error(`[HealthCheck] Error in checkApi: ${(error as Error).message}`);
            this.apiHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                API_SERVICE_CACHE_MS,
                { error: "Failed to check API health" },
            );
            return this.apiHealthCache.health;
        }
    }

    /**
     * Check WebSocket server health using SocketService
     */
    private checkWebSocket(): ServiceHealth {
        const cachedHealth = this.getCachedHealthIfValid(this.websocketHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            // Get the singleton instance
            const socketService = SocketService.get();
            // Get health details from the service
            const details = socketService.getHealthDetails();

            if (details) {
                this.websocketHealthCache = this.createServiceCache(
                    ServiceStatus.Operational,
                    DEFAULT_SERVICE_CACHE_MS,
                    details,
                );
                return this.websocketHealthCache.health;
            } else {
                this.websocketHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "WebSocket service component not fully initialized." },
                );
                return this.websocketHealthCache.health;
            }
        } catch (error) {
            this.websocketHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: (error as Error).message || "Failed to check WebSocket server status" },
            );
            return this.websocketHealthCache.health;
        }
    }

    /**
     * Check cron job health
     * 
     * This method uses Redis to track cron job execution history.
     * Each cron job records its success/failure to Redis.
     */
    private async checkCronJobs(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.cronJobsHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        const now = Date.now();
        try {
            const redis = await initializeRedis();
            if (!redis) {
                this.cronJobsHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "Redis client not initialized" },
                );
                return this.cronJobsHealthCache.health;
            }

            // Get all registered cron jobs
            const jobNames = await redis.sMembers("cron:jobs");

            if (!jobNames.length) {
                this.cronJobsHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "No cron jobs registered in Redis" },
                );
                return this.cronJobsHealthCache.health;
            }

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
            const cronStatus = hasCriticalFailures ? ServiceStatus.Down :
                hasFailures ? ServiceStatus.Degraded : ServiceStatus.Operational;

            this.cronJobsHealthCache = this.createServiceCache(
                cronStatus,
                DEFAULT_SERVICE_CACHE_MS,
                { jobs: jobDetails, total: jobNames.length },
            );
            return this.cronJobsHealthCache.health;
        } catch (error) {
            this.cronJobsHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: "Failed to check cron job health" },
            );
            return this.cronJobsHealthCache.health;
        }
    }

    /**
     * Check i18next initialization
     */
    private checkI18n(): ServiceHealth {
        const cachedHealth = this.getCachedHealthIfValid(this.i18nHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            // Check if i18next is initialized
            if (!i18next.isInitialized) {
                this.i18nHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "i18next is not initialized" },
                );
                return this.i18nHealthCache.health;
            }

            // Check if we can translate a basic key
            try {
                const translation = i18next.t("common:Yes");
                const isTranslated = translation !== "common:Yes" && typeof translation === "string" && translation.length > 0;

                if (!isTranslated) {
                    this.i18nHealthCache = this.createServiceCache(
                        ServiceStatus.Degraded,
                        DEFAULT_SERVICE_CACHE_MS,
                        { error: "i18next is initialized but translations are not working properly" },
                    );
                    return this.i18nHealthCache.health;
                }

                this.i18nHealthCache = this.createServiceCache(
                    ServiceStatus.Operational,
                    DEFAULT_SERVICE_CACHE_MS,
                    { language: i18next.language, languages: i18next.languages, namespaces: i18next.options.ns },
                );
                return this.i18nHealthCache.health;
            } catch (error) {
                this.i18nHealthCache = this.createServiceCache(
                    ServiceStatus.Degraded,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: (error as Error).message || "i18next is initialized but failed to translate" },
                );
                return this.i18nHealthCache.health;
            }
        } catch (error) {
            this.i18nHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: (error as Error).message || "Failed to check i18next" },
            );
            return this.i18nHealthCache.health;
        }
    }

    /**
     * Check image storage health (S3 and NSFW detection)
     */
    private async checkImageStorage(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.imageStorageHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        let s3Healthy = false;
        let nsfwDetectionHealthy = false;
        let s3Details: Record<string, unknown> = {};
        let nsfwDetails: Record<string, unknown> = {};
        let s3Client: S3Client | undefined;

        // 1. Check S3 Client and Bucket Access
        try {
            s3Client = getS3Client(); // Make sure getS3Client is exported from fileStorage.ts
            if (!s3Client) {
                this.imageStorageHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "S3 client not initialized" },
                );
                return this.imageStorageHealthCache.health;
            }
            // Simple check: attempt to get bucket metadata (requires ListBucket permissions)
            // Replace 'vrooli-bucket' if the actual bucket name is different or stored in env vars
            const bucketName = process.env.S3_BUCKET_NAME || "vrooli-bucket";
            const command = new HeadBucketCommand({ Bucket: bucketName });
            await s3Client.send(command);
            s3Healthy = true;
            s3Details = { initialized: true, bucketAccess: true, bucket: bucketName };
        } catch (error) {
            s3Healthy = false;
            s3Details = {
                initialized: !!s3Client,
                bucketAccess: false,
                error: `S3 check failed: ${(error as Error).message}`,
            };
            logger.error("S3 health check failed during HeadBucketCommand", { trace: "health-s3-fail", error });
        }

        // 2. Check NSFW Detection Service functionality by sending a safe image
        try {
            // Correct way to get directory path in ESM
            const currentFilePath = fileURLToPath(import.meta.url);
            const currentDirPath = path.dirname(currentFilePath);
            const safeImagePath = path.resolve(currentDirPath, "..", "..", "test-assets", "safe-test-image.webp");

            // Read the safe image file
            const safeImageBuffer = await fs.readFile(safeImagePath);

            // Call the checkNSFW function (imported from fileStorage.ts)
            const isNsfwResult = await checkNSFW(safeImageBuffer, "safe-test-image.png");

            // Health check passes if the function returns a boolean without error
            if (typeof isNsfwResult === "boolean") {
                nsfwDetectionHealthy = true;
                nsfwDetails = { serviceCalled: true, result: isNsfwResult };
            } else {
                this.imageStorageHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: `checkNSFW returned non-boolean value: ${typeof isNsfwResult}` },
                );
                return this.imageStorageHealthCache.health;
            }
        } catch (error) {
            // Handle file read errors or checkNSFW call errors
            nsfwDetectionHealthy = false;
            nsfwDetails = {
                serviceCalled: false,
                error: `NSFW detection check failed: ${(error as Error).message}`,
            };
            logger.warning("NSFW detection health check failed", { trace: "health-nsfw-fail", details: nsfwDetails });
        }

        // Determine overall status
        let overallStatus: ServiceStatus;
        if (s3Healthy && nsfwDetectionHealthy) {
            overallStatus = ServiceStatus.Operational;
        } else if (s3Healthy || nsfwDetectionHealthy) {
            // If only one is down, consider it degraded
            overallStatus = ServiceStatus.Degraded;
        } else {
            overallStatus = ServiceStatus.Down;
        }

        this.imageStorageHealthCache = this.createServiceCache(
            overallStatus,
            DEFAULT_SERVICE_CACHE_MS,
            {
                s3: { healthy: s3Healthy, ...s3Details },
                nsfwDetection: { healthy: nsfwDetectionHealthy, ...nsfwDetails },
            },
        );
        return this.imageStorageHealthCache.health;
    }

    /**
     * Check Embedding Service health
     */
    private async checkEmbeddingService(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.embeddingServiceHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }

        try {
            const health = await EmbeddingService.get().checkHealth();
            this.embeddingServiceHealthCache = this.createServiceCache(
                health.status,
                EMBEDDING_SERVICE_CACHE_MS,
                health.details,
            );
            return this.embeddingServiceHealthCache.health;
        } catch (error) {
            this.embeddingServiceHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                EMBEDDING_SERVICE_CACHE_MS,
                { error: `Failed to check Embedding Service: ${(error instanceof Error) ? error.message : String(error)}` },
            );
            return this.embeddingServiceHealthCache.health;
        }
    }

    /**
     * Get overall system health status
     */
    public async getHealth(): Promise<SystemHealth> {

        // Perform health checks in parallel
        const results = await Promise.allSettled([
            this.checkApi(),                   // 0
            this.checkBus(),                   // 1
            this.checkCronJobs(),              // 2
            this.checkDatabase(),              // 3
            this.checkI18n(),                  // 4
            this.checkLlmServices(),           // 5
            this.checkMcp(),                   // 6
            this.checkMemory(),                // 7
            this.checkQueues(),                // 8
            this.checkRedis(),                 // 9
            this.checkSslCertificate(),        // 10
            this.checkStripe(),                // 11
            this.checkSystem(),                // 12
            this.checkWebSocket(),             // 13
            this.checkImageStorage(),          // 14
            this.checkEmbeddingService(),      // 15
        ]);

        // Helper to create a default 'Down' status for failed checks
        function createDownStatus(serviceName: string, error: unknown): ServiceHealth {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: `Failed to check ${serviceName}: ${(error instanceof Error) ? error.message : String(error)}` },
            };
        }

        // Process results, providing defaults for rejected promises
        const apiHealth = results[0].status === "fulfilled" ? results[0].value : createDownStatus("API", results[0].reason);
        const busHealth = results[1].status === "fulfilled" ? results[1].value : createDownStatus("Bus", results[1].reason);
        const cronJobsHealth = results[2].status === "fulfilled" ? results[2].value : createDownStatus("Cron Jobs", results[2].reason);
        const dbHealth = results[3].status === "fulfilled" ? results[3].value : createDownStatus("Database", results[3].reason);
        const i18nHealth = results[4].status === "fulfilled" ? results[4].value : createDownStatus("i18n", results[4].reason);
        const llmHealth = results[5].status === "fulfilled" ? results[5].value : {}; // Handled differently as it's an object
        if (results[5].status === "rejected") logger.error("LLM health check failed", { trace: "health-llm-fail", error: results[5].reason });
        const mcpHealth = results[6].status === "fulfilled" ? results[6].value : createDownStatus("MCP", results[6].reason);
        const memoryHealth = results[7].status === "fulfilled" ? results[7].value : createDownStatus("Memory", results[7].reason);
        const queueHealths = results[8].status === "fulfilled" ? results[8].value : {}; // Handled differently as it's an object
        if (results[8].status === "rejected") logger.error("Queue health check failed", { trace: "health-queue-fail", error: results[8].reason });
        const redisHealth = results[9].status === "fulfilled" ? results[9].value : createDownStatus("Redis", results[9].reason);
        const sslHealth = results[10].status === "fulfilled" ? results[10].value : createDownStatus("SSL", results[10].reason);
        const stripeHealth = results[11].status === "fulfilled" ? results[11].value : createDownStatus("Stripe", results[11].reason);
        const systemHealth = results[12].status === "fulfilled" ? results[12].value : createDownStatus("System", results[12].reason);
        const websocketHealth = results[13].status === "fulfilled" ? results[13].value : createDownStatus("WebSocket", results[13].reason);
        const imageStorageHealth = results[14].status === "fulfilled" ? results[14].value : createDownStatus("Image Storage", results[14].reason);
        const embeddingServiceHealth = results[15].status === "fulfilled" ? results[15].value : createDownStatus("Embedding Service", results[15].reason);


        // Determine overall status
        const allServices = [
            apiHealth,
            busHealth,
            cronJobsHealth,
            dbHealth,
            i18nHealth,
            mcpHealth,
            memoryHealth,
            redisHealth,
            sslHealth,
            stripeHealth,
            systemHealth,
            websocketHealth,
            imageStorageHealth,
            embeddingServiceHealth,
            ...Object.values(queueHealths),
            ...Object.values(llmHealth),
        ];
        const overallStatus = allServices.every(s => s?.status === ServiceStatus.Operational) ? ServiceStatus.Operational :
            allServices.some(s => s?.status === ServiceStatus.Down) ? ServiceStatus.Down : ServiceStatus.Degraded;

        const health: SystemHealth = {
            status: overallStatus,
            version: process.env.npm_package_version || "unknown",
            services: {
                api: apiHealth,
                bus: busHealth,
                cronJobs: cronJobsHealth,
                database: dbHealth,
                i18n: i18nHealth,
                llm: llmHealth,
                mcp: mcpHealth,
                memory: memoryHealth,
                queues: queueHealths,
                redis: redisHealth,
                ssl: sslHealth,
                stripe: stripeHealth,
                system: systemHealth,
                websocket: websocketHealth,
                imageStorage: imageStorageHealth,
                embeddingService: embeddingServiceHealth,
            },
            timestamp: Date.now(),
        };

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

    if (debug) {
        // Typicaly this would be a POST request, but it's easier to call this as a GET request 
        // since you can do it from the browser.
        app.get("/healthcheck/retry-seeding", async (req, res) => {
            try {
                // Call the force retry method
                const success = DbProvider.forceRetrySeeding();

                if (success) {
                    res.status(HttpStatus.Ok).json({
                        success: true,
                        message: "Database seeding retry triggered successfully",
                        timestamp: Date.now(),
                    });
                } else {
                    res.status(HttpStatus.BadRequest).json({
                        success: false,
                        message: "Failed to trigger database seeding retry",
                        timestamp: Date.now(),
                    });
                }
            } catch (error) {
                res.status(HttpStatus.InternalServerError).json({
                    success: false,
                    message: "Error triggering database seeding retry",
                    error: (error as Error).message,
                    timestamp: Date.now(),
                });
            }
        });

        // Development endpoint to clear Bull queue job data
        app.get("/healthcheck/clear-queue-data", async (req, res) => {
            try {
                const queueName = req.query.queue as string;
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

                // Check if a specific queue was requested
                if (queueName && queues[queueName as keyof typeof queues]) {
                    const queue = queues[queueName as keyof typeof queues];
                    await queue.getQueue().clean(0, "completed");
                    await queue.getQueue().clean(0, "failed");
                    await queue.getQueue().clean(0, "delayed");
                    await queue.getQueue().clean(0, "wait");
                    await queue.getQueue().clean(0, "active");

                    res.status(HttpStatus.Ok).json({
                        success: true,
                        message: `Queue '${queueName}' job data cleared successfully`,
                        timestamp: Date.now(),
                    });
                } else if (queueName) {
                    // Invalid queue name provided
                    res.status(HttpStatus.BadRequest).json({
                        success: false,
                        message: `Invalid queue name: ${queueName}`,
                        validQueues: Object.keys(queues),
                        timestamp: Date.now(),
                    });
                } else {
                    // No queue specified, clean all queues
                    const results: Record<string, boolean> = {};

                    for (const [name, queue] of Object.entries(queues)) {
                        try {
                            await queue.getQueue().clean(0, "completed");
                            await queue.getQueue().clean(0, "failed");
                            await queue.getQueue().clean(0, "delayed");
                            await queue.getQueue().clean(0, "wait");
                            await queue.getQueue().clean(0, "active");
                            results[name] = true;
                        } catch (error) {
                            results[name] = false;
                        }
                    }

                    res.status(HttpStatus.Ok).json({
                        success: true,
                        message: "Queue job data cleared",
                        results,
                        timestamp: Date.now(),
                    });
                }
            } catch (error) {
                res.status(HttpStatus.InternalServerError).json({
                    success: false,
                    message: "Error clearing queue job data",
                    error: (error as Error).message,
                    timestamp: Date.now(),
                });
            }
        });

        // Development endpoint to send a test notification
        // Example: http://localhost:5329/healthcheck/send-test-notification?userId=3f038f3b-f8f9-4f9b-8f9b-c8f4b8f9b8d2
        app.get("/healthcheck/send-test-notification", async (req, res) => {
            const userId = req.query.userId as string;

            if (!userId) {
                return res.status(HttpStatus.BadRequest).json({
                    success: false,
                    message: "Missing 'userId' query parameter",
                    timestamp: Date.now(),
                });
            }

            try {
                // Fetch user languages to ensure proper notification translation
                const user = await DbProvider.get().user.findUnique({
                    where: { id: BigInt(userId) },
                    select: { languages: true },
                });

                if (!user) {
                    return res.status(HttpStatus.NotFound).json({
                        success: false,
                        message: `User with ID '${userId}' not found`,
                        timestamp: Date.now(),
                    });
                }

                // Use a generic notification type or adapt as needed
                await Notify(user.languages)
                    .pushNewDeviceSignIn() // Using an existing simple notification for testing
                    .toUser(userId);

                res.status(HttpStatus.Ok).json({
                    success: true,
                    message: `Test notification sent to user '${userId}'`,
                    timestamp: Date.now(),
                });
            } catch (error) {
                console.error(`[HealthCheck] Error sending test notification to user '${userId}':`, error);
                res.status(HttpStatus.InternalServerError).json({
                    success: false,
                    message: "Error sending test notification",
                    error: (error as Error).message,
                    timestamp: Date.now(),
                });
            }
        });
    }
}
