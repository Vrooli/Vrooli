import { HeadBucketCommand, type S3Client } from "@aws-sdk/client-s3";
import { ApiKeyPermission, DAYS_1_S, GB_1_BYTES, HOURS_1_MS, HttpStatus, MINUTES_15_MS, MINUTES_1_MS, MINUTES_2_S, MINUTES_30_MS, ResourceSubType, SECONDS_1_MS, SERVER_VERSION, endpointsAuth, endpointsFeed } from "@local/shared";
import { exec as execCb } from "child_process";
import type { Request, Response } from "express";
import { type Express } from "express";
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
import { CacheService } from "../redisConn.js";
import { API_URL, SERVER_URL } from "../server.js";
import { SocketService } from "../sockets/io.js";
import { QueueStatus } from "../tasks/queueFactory.js";
import { QueueService } from "../tasks/queues.js";
import { checkNSFW, getS3Client } from "../utils/fileStorage.js";
import { BusService, type RedisStreamBus } from "./bus.js";
import { AIServiceRegistry, AIServiceState } from "./conversation/registry.js";
import { EmbeddingService } from "./embedding.js";
import { runWithMcpContext } from "./mcp/context.js";
import { getMcpServer } from "./mcp/index.js";
import { McpToolName } from "./types/tools.js";

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

// Per-service cache durations
const DEFAULT_SERVICE_CACHE_MS = MINUTES_2_S * SECONDS_1_MS; // 2 minutes
const EMBEDDING_SERVICE_CACHE_MS = HOURS_1_MS; // 1 hour
const STRIPE_SERVICE_CACHE_MS = MINUTES_30_MS; // 30 minutes
const LLM_SERVICE_CACHE_MS = MINUTES_15_MS; // 15 minutes
const API_SERVICE_CACHE_MS = MINUTES_1_MS; // 1 minute
// eslint-disable-next-line no-magic-numbers
const SSL_SERVICE_CACHE_MS = HOURS_1_MS * 6; // 6 hours

const BUS_BACKLOG_DEGRADED_THRESHOLD_MS = 5000; // Threshold for bus backlog to be considered degraded

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

            // Gather raw Redis‐stream metrics from the bus
            let rawMetrics: any = { connected: true, consumerGroups: {} };
            if (typeof (bus as any).metrics === "function") {
                // bus.metrics() returns a ServiceHealth; unwrap its .details
                const svcHealth = await (bus as RedisStreamBus).metrics();
                rawMetrics = (svcHealth.details ?? {}) as any;
            }

            // Sum pending messages across all consumer groups
            const consumerGroups = rawMetrics.consumerGroups ?? {};
            const backlog = Object.values(consumerGroups as Record<string, any>)
                .reduce((sum, grp: any) => sum + (grp.pending ?? 0), 0);

            // Determine connected status
            const connected = rawMetrics.connected === true;
            const status = !connected
                ? ServiceStatus.Down
                : backlog > BUS_BACKLOG_DEGRADED_THRESHOLD_MS
                    ? ServiceStatus.Degraded
                    : ServiceStatus.Operational;

            // Cache only the raw metrics details
            this.busHealthCache = this.createServiceCache(status, DEFAULT_SERVICE_CACHE_MS, rawMetrics);
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
            const redisClient = await CacheService.get().raw();
            await redisClient.ping();
            this.redisHealthCache = this.createServiceCache(
                ServiceStatus.Operational,
                DEFAULT_SERVICE_CACHE_MS,
            );
            return this.redisHealthCache.health;
        } catch (error) {
            // Aggressive error detail extraction - simplified for robust JSON serialization
            let serializableErrorDetails: object;
            if (error instanceof Error) {
                serializableErrorDetails = {
                    message: error.message,
                    name: error.name,
                    stack: debug ? error.stack : "Stack trace available in development mode",
                    type: "ErrorInstance",
                };
            } else {
                let messageString: string;
                try {
                    messageString = JSON.stringify(error);
                } catch (stringifyError) {
                    messageString = String(error);
                }
                serializableErrorDetails = {
                    message: messageString,
                    type: typeof error,
                    details: "Non-Error object thrown",
                };
            }

            this.redisHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                {
                    error: serializableErrorDetails,
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
        const queueSvc = QueueService.get();
        queueSvc.initializeAllQueues();   // forces lazy queues to materialise
        const queueHealth: { [key: string]: ServiceHealth } = {};

        for (const [name, q] of Object.entries(queueSvc.queues)) {
            const health = await q.checkHealth();

            queueHealth[name] = {
                healthy: health.status === QueueStatus.Healthy,
                status: health.status === QueueStatus.Healthy
                    ? ServiceStatus.Operational
                    : health.status === QueueStatus.Degraded
                        ? ServiceStatus.Degraded
                        : ServiceStatus.Down,
                lastChecked: now,
                details: {
                    counts: health.jobCounts,
                    activeJobs: health.activeJobs,
                    metrics: {
                        queueLength: health.jobCounts.waiting + health.jobCounts.delayed,
                        activeCount: health.jobCounts.active,
                        failedCount: health.jobCounts.failed,
                        completedCount: health.jobCounts.completed,
                        totalJobs: health.jobCounts.total,
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
        const registry = AIServiceRegistry.get();
        const services = registry["serviceStates"];
        const llmHealth: { [key: string]: ServiceHealth } = {};

        for (const [serviceId, state] of services.entries()) {
            llmHealth[serviceId] = {
                healthy: state.state === AIServiceState.Active,
                status: state.state === AIServiceState.Active ? ServiceStatus.Operational :
                    state.state === AIServiceState.Cooldown ? ServiceStatus.Degraded :
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
        const cached = this.getCachedHealthIfValid(this.mcpHealthCache);
        if (cached) {
            return cached;
        }
        let transportHealthy = false;
        let builtInHealthy = false;
        let transportDetails: Record<string, unknown> = {};
        let builtInDetails: Record<string, unknown> = {};

        // Mock request and response for context
        const mockReq = {
            session: {
                fromSafeOrigin: true,
                isLoggedIn: true, // Assuming internal calls are implicitly "logged in"
                // Grant all permissions for internal health check
                permissions: {
                    [ApiKeyPermission.ReadPublic]: true,
                    [ApiKeyPermission.ReadPrivate]: true,
                    [ApiKeyPermission.WritePrivate]: true,
                    [ApiKeyPermission.WriteAuth]: true,
                    [ApiKeyPermission.ReadAuth]: true,
                },
                userId: "system-healthcheck",
                apiToken: undefined,
                languages: ["en"],
            },
            ip: "127.0.0.1",
            headers: {},
            body: {},
            route: { path: "/healthcheck/mcp" },
            method: "GET",
        } as unknown as Request;
        const mockRes = {} as Response;

        try {
            // 1. Check Transport Layer
            const mcpApp = getMcpServer();
            const transportInfo = mcpApp.getTransportManager().getHealthInfo();
            transportHealthy = typeof transportInfo.activeConnections === "number" && transportInfo.activeConnections >= 0;
            transportDetails = transportInfo;

            // 2. Built-in tools via registry
            const toolRegistry = mcpApp.getToolRegistry();
            // Wrap in runWithMcpContext
            await runWithMcpContext(mockReq, mockRes, async () => {
                // Use DefineTool for a simple resource type to check basic tool functionality
                const defineToolResult = await toolRegistry.execute(McpToolName.DefineTool, { toolName: McpToolName.ResourceManage, variant: ResourceSubType.RoutineInformational, op: "find" });
                builtInHealthy = defineToolResult && typeof defineToolResult.content === "object" && !defineToolResult.isError;
                builtInDetails = { result: defineToolResult };
            });

        } catch (error) {
            logger.error("Error during MCP health check:", { trace: "mcp-health-error", error });
            transportHealthy = false; // Assume all failed if an error occurs during checks
            builtInHealthy = false;
            transportDetails = { error: (error as Error).message };
            builtInDetails = { error: (error as Error).message };
        }

        const currentMcpOverallStatus = (transportHealthy && builtInHealthy)
            ? ServiceStatus.Operational
            : ServiceStatus.Degraded;

        const healthToCache: ServiceHealth = {
            status: currentMcpOverallStatus,
            healthy: currentMcpOverallStatus === ServiceStatus.Operational,
            lastChecked: Date.now(),
            details: {
                transport: { healthy: transportHealthy, ...transportDetails },
                builtInTools: { healthy: builtInHealthy, ...builtInDetails },
            },
        };

        this.mcpHealthCache = {
            health: healthToCache,
            expiresAt: Date.now() + MINUTES_1_MS,
        };

        return healthToCache;
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
            const redis = await CacheService.get().raw();
            // Get all registered cron jobs
            const jobNames = await redis.smembers("cron:jobs");

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
        logger.info("HealthService.getHealth() started");

        const serviceChecks = [
            { name: "API", check: () => this.checkApi() },
            { name: "Bus", check: () => this.checkBus() },
            { name: "CronJobs", check: () => this.checkCronJobs() },
            { name: "Database", check: () => this.checkDatabase() },
            { name: "i18n", check: () => this.checkI18n() },
            { name: "LLMServices", check: () => this.checkLlmServices() },
            { name: "MCP", check: () => this.checkMcp() },
            { name: "Memory", check: () => this.checkMemory() },
            { name: "Queues", check: () => this.checkQueues() },
            { name: "Redis", check: () => this.checkRedis() },
            { name: "SSLCertificate", check: () => this.checkSslCertificate() },
            { name: "Stripe", check: () => this.checkStripe() },
            { name: "System", check: () => this.checkSystem() },
            { name: "WebSocket", check: () => this.checkWebSocket() },
            { name: "ImageStorage", check: () => this.checkImageStorage() },
            { name: "EmbeddingService", check: () => this.checkEmbeddingService() },
        ];

        const results: { name: string, status: "fulfilled" | "rejected", value?: any, reason?: any }[] = [];

        for (const { name, check } of serviceChecks) {
            logger.info(`Health check starting for: ${name}`);
            try {
                const result = await check();
                logger.info(`Health check completed for: ${name}`);
                results.push({ name, status: "fulfilled", value: result });
            } catch (error) {
                logger.error(`Health check failed for: ${name}`, { error });
                results.push({ name, status: "rejected", reason: error });
            }
        }

        logger.info("All individual health checks processed.");

        // Helper to create a default 'Down' status for failed checks
        function createDownStatus(serviceName: string, error: unknown): ServiceHealth {
            // Aggressive error detail extraction - simplified for robust JSON serialization
            let serializableErrorDetails: object;
            if (error instanceof Error) {
                serializableErrorDetails = {
                    message: error.message,
                    name: error.name,
                    stack: debug ? error.stack : "Stack trace available in development mode",
                    type: "ErrorInstance",
                };
            } else {
                let messageString: string;
                try {
                    messageString = JSON.stringify(error);
                } catch (stringifyError) {
                    messageString = String(error);
                }
                serializableErrorDetails = {
                    message: messageString,
                    type: typeof error,
                    details: "Non-Error object thrown",
                };
            }

            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    checkFailedFor: serviceName,
                    errorDetails: serializableErrorDetails,
                },
            };
        }

        // Process results, providing defaults for rejected promises
        // Match these indices to the order in serviceChecks array
        const apiHealth = results.find(r => r.name === "API")?.status === "fulfilled" ? results.find(r => r.name === "API")!.value : createDownStatus("API", results.find(r => r.name === "API")?.reason);
        const busHealth = results.find(r => r.name === "Bus")?.status === "fulfilled" ? results.find(r => r.name === "Bus")!.value : createDownStatus("Bus", results.find(r => r.name === "Bus")?.reason);
        const cronJobsHealth = results.find(r => r.name === "CronJobs")?.status === "fulfilled" ? results.find(r => r.name === "CronJobs")!.value : createDownStatus("Cron Jobs", results.find(r => r.name === "CronJobs")?.reason);
        const dbHealth = results.find(r => r.name === "Database")?.status === "fulfilled" ? results.find(r => r.name === "Database")!.value : createDownStatus("Database", results.find(r => r.name === "Database")?.reason);
        const i18nHealth = results.find(r => r.name === "i18n")?.status === "fulfilled" ? results.find(r => r.name === "i18n")!.value : createDownStatus("i18n", results.find(r => r.name === "i18n")?.reason);

        let llmHealth: { [key: string]: ServiceHealth } = {};
        const llmResult = results.find(r => r.name === "LLMServices");
        if (llmResult?.status === "fulfilled") {
            llmHealth = llmResult.value;
        } else if (llmResult?.status === "rejected") {
            logger.error("LLM health check failed", { trace: "health-llm-fail", error: llmResult.reason });
            // llmHealth remains empty or you could add a generic error entry
        }

        const mcpHealth = results.find(r => r.name === "MCP")?.status === "fulfilled" ? results.find(r => r.name === "MCP")!.value : createDownStatus("MCP", results.find(r => r.name === "MCP")?.reason);
        const memoryHealth = results.find(r => r.name === "Memory")?.status === "fulfilled" ? results.find(r => r.name === "Memory")!.value : createDownStatus("Memory", results.find(r => r.name === "Memory")?.reason);

        let queueHealths: { [key: string]: ServiceHealth } = {};
        const queueResult = results.find(r => r.name === "Queues");
        if (queueResult?.status === "fulfilled") {
            queueHealths = queueResult.value;
        } else if (queueResult?.status === "rejected") {
            const reason = queueResult.reason;
            const errorDetails = (reason instanceof Error)
                ? { message: reason.message, name: reason.name, stack: reason.stack, type: "ErrorInstance" }
                : { /* ... detailed error object construction ... */ };
            logger.error("Queue health check failed", { trace: "health-queue-fail", error: errorDetails });
            // queueHealths remains empty or you could add a generic error entry
        }

        const redisHealth = results.find(r => r.name === "Redis")?.status === "fulfilled" ? results.find(r => r.name === "Redis")!.value : createDownStatus("Redis", results.find(r => r.name === "Redis")?.reason);
        const sslHealth = results.find(r => r.name === "SSLCertificate")?.status === "fulfilled" ? results.find(r => r.name === "SSLCertificate")!.value : createDownStatus("SSL", results.find(r => r.name === "SSLCertificate")?.reason);
        const stripeHealth = results.find(r => r.name === "Stripe")?.status === "fulfilled" ? results.find(r => r.name === "Stripe")!.value : createDownStatus("Stripe", results.find(r => r.name === "Stripe")?.reason);
        const systemHealth = results.find(r => r.name === "System")?.status === "fulfilled" ? results.find(r => r.name === "System")!.value : createDownStatus("System", results.find(r => r.name === "System")?.reason);
        const websocketHealth = results.find(r => r.name === "WebSocket")?.status === "fulfilled" ? results.find(r => r.name === "WebSocket")!.value : createDownStatus("WebSocket", results.find(r => r.name === "WebSocket")?.reason);
        const imageStorageHealth = results.find(r => r.name === "ImageStorage")?.status === "fulfilled" ? results.find(r => r.name === "ImageStorage")!.value : createDownStatus("Image Storage", results.find(r => r.name === "ImageStorage")?.reason);
        const embeddingServiceHealth = results.find(r => r.name === "EmbeddingService")?.status === "fulfilled" ? results.find(r => r.name === "EmbeddingService")!.value : createDownStatus("Embedding Service", results.find(r => r.name === "EmbeddingService")?.reason);


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
            // Spread results from checks that return an object of ServiceHealth
            ...(Object.values(queueHealths) as ServiceHealth[]),
            ...(Object.values(llmHealth) as ServiceHealth[]),
        ].filter(s => s !== undefined && s !== null); // Filter out undefined/null if createDownStatus can return them or if a check isn't found

        const overallStatus = allServices.every(s => s?.status === ServiceStatus.Operational) ? ServiceStatus.Operational :
            allServices.some(s => s?.status === ServiceStatus.Down) ? ServiceStatus.Down : ServiceStatus.Degraded;

        logger.info(`HealthService.getHealth() completed. Overall status: ${overallStatus}`);

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
        console.log("here 1");
        const HEALTH_CHECK_TIMEOUT_MS = 30000; // 30 seconds

        const timeoutPromise = new Promise((_, reject) => {
            setTimeout(() => {
                reject(new Error("Health check timed out"));
            }, HEALTH_CHECK_TIMEOUT_MS);
        });

        try {
            const health = await Promise.race([
                HealthService.get().getHealth(),
                timeoutPromise,
            ]) as SystemHealth; // Added type assertion after Promise.race

            res.status(HttpStatus.Ok).json(health);
        } catch (error) {
            let errorMessage = "Health check failed";
            if (error instanceof Error && error.message === "Health check timed out") {
                errorMessage = "Health check timed out after " + (HEALTH_CHECK_TIMEOUT_MS / SECONDS_1_MS) + " seconds";
            } else if (error instanceof Error) {
                errorMessage = `Health check failed: ${error.message}`;
            }
            // Log the actual error for server-side diagnostics
            logger.error(`🚨 ${errorMessage}`, { trace: "healthcheck-error", error });

            res.status(HttpStatus.ServiceUnavailable).json({
                status: ServiceStatus.Down,
                version: process.env.npm_package_version || "unknown",
                error: errorMessage, // Send more specific error message
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
                const qs = QueueService.get();
                const queues = qs.queues;
                const cleanLimit = 10_000;

                // Check if a specific queue was requested
                if (queueName && queues[queueName as keyof typeof queues]) {
                    const queue = queues[queueName as keyof typeof queues];
                    await queue.queue.clean(0, cleanLimit, "completed");
                    await queue.queue.clean(0, cleanLimit, "failed");
                    await queue.queue.clean(0, cleanLimit, "delayed");
                    await queue.queue.clean(0, cleanLimit, "wait");
                    await queue.queue.clean(0, cleanLimit, "active");

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
                            await queue.queue.clean(0, cleanLimit, "completed");
                            await queue.queue.clean(0, cleanLimit, "failed");
                            await queue.queue.clean(0, cleanLimit, "delayed");
                            await queue.queue.clean(0, cleanLimit, "wait");
                            await queue.queue.clean(0, cleanLimit, "active");
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
