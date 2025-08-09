// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-08-05
import { HeadBucketCommand, type S3Client } from "@aws-sdk/client-s3";
import { ApiKeyPermission, DAYS_1_S, GB_1_BYTES, HOURS_1_MS, HttpStatus, MINUTES_15_MS, MINUTES_1_MS, MINUTES_2_S, MINUTES_30_MS, McpToolName, ResourceSubType, SECONDS_1_MS, SECONDS_5_MS } from "@vrooli/shared";
import { exec as execCb } from "child_process";
import type { Request, Response } from "express";
import { type Express } from "express";
import fs from "fs/promises";
import i18next from "i18next";
import os from "os";
import * as path from "path";
import Stripe from "stripe";
import { fileURLToPath } from "url";
import { promisify } from "util";
import { MigrationService } from "../db/migrations.js";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { Notify } from "../notify/notify.js";
import { CacheService } from "../redisConn.js";
import { API_URL, SERVER_URL } from "../server.js";
import { SocketService } from "../sockets/io.js";
import { QueueStatus } from "../tasks/queueFactory.js";
import { QueueService } from "../tasks/queues.js";
import { getExecutionMode } from "../utils/executionMode.js";
import { checkNSFW, getS3Client } from "../utils/fileStorage.js";
import { getImageProcessingStatus } from "../utils/sharpWrapper.js";
import { BusService } from "./bus.js";
import { EmbeddingService } from "./embedding.js";
import { runWithMcpContext } from "./mcp/context.js";
import { getMcpServer } from "./mcp/index.js";
import { AIServiceRegistry, AIServiceState } from "./response/registry.js";
import { ResourceRegistry } from "./resources/ResourceRegistry.js";

const exec = promisify(execCb);

// Dynamic debug check to allow runtime NODE_ENV changes
function isDebugMode() {
    return process.env.NODE_ENV === "development";
}

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
        resources: ServiceHealth;
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
const _HTTP_STATUS_OK_MIN = 200;
// eslint-disable-next-line no-magic-numbers  
const _HTTP_STATUS_OK_MAX = 300;

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

const PER_SERVICE_TIMEOUT_MS = SECONDS_5_MS;

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
    private resourcesHealthCache: { health: ServiceHealth, expiresAt: number } | null = null;

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

        // Add timeout for bus initialization
        const BUS_CHECK_TIMEOUT_MS = 5000; // 5 seconds
        const timeoutPromise = new Promise((_, reject) => {
            setTimeout(() => {
                logger.error("[Health] Bus check timeout after 5 seconds");
                reject(new Error("Bus health check timed out"));
            }, BUS_CHECK_TIMEOUT_MS);
        });

        try {
            // Ensure bus exists (lazy init is idempotent) with timeout
            await Promise.race([
                BusService.get().startEventBus(),
                timeoutPromise,
            ]);

            // Skip metrics collection due to Redis performance issues
            // The Bus is operational if we can get it and it's started
            const rawMetrics: any = {
                connected: true,
                consumerGroups: {},
                note: "Metrics collection skipped for performance",
            };

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

            // Check migration status
            let migrationStatus;
            try {
                migrationStatus = await MigrationService.checkStatus();
            } catch (migrationError) {
                // If we can't check migrations, log it but continue
                logger.warning("Failed to check migration status", { error: migrationError });
                migrationStatus = {
                    hasPending: false,
                    pendingCount: 0,
                    pendingMigrations: [],
                    appliedCount: 0,
                    appliedMigrations: [],
                    error: "Failed to check migration status",
                };
            }

            // Check if seeding was successful
            const seedingSuccessful = DbProvider.isSeedingSuccessful();
            const isRetrying = DbProvider.isRetryingSeeding();
            const retryCount = DbProvider.getSeedRetryCount();
            const attemptCount = DbProvider.getSeedAttemptCount();
            const maxRetries = DbProvider.getMaxRetries();

            // If migrations are pending, that's likely why seeding is failing
            if (migrationStatus.hasPending) {
                const migrationMessage = `${migrationStatus.pendingCount} pending migration(s): ${migrationStatus.pendingMigrations.join(", ")}`;
                this.databaseHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    {
                        connection: true,
                        migrations: {
                            hasPending: true,
                            pendingCount: migrationStatus.pendingCount,
                            pendingMigrations: migrationStatus.pendingMigrations,
                            appliedCount: migrationStatus.appliedCount,
                        },
                        seeding: false,
                        message: `Database has pending migrations. ${migrationMessage}. Run 'pnpm prisma migrate deploy' before starting the server.`,
                    });
                return this.databaseHealthCache.health;
            }

            // If connection works but seeding failed (and migrations are up to date)
            if (!seedingSuccessful) {
                this.databaseHealthCache = this.createServiceCache(
                    isRetrying ? ServiceStatus.Degraded : ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    {
                        connection: true,
                        migrations: {
                            hasPending: false,
                            appliedCount: migrationStatus.appliedCount,
                            ...(migrationStatus.error && { error: migrationStatus.error }),
                        },
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
                    migrations: {
                        hasPending: false,
                        appliedCount: migrationStatus.appliedCount,
                        message: `All ${migrationStatus.appliedCount} migrations applied`,
                        ...(migrationStatus.error && { error: migrationStatus.error }),
                    },
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
                    stack: isDebugMode() ? error.stack : "Stack trace available in development mode",
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
        const queueHealth: { [key: string]: ServiceHealth } = {};

        // Only check queues that are already initialized
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
    private async checkMemory(): Promise<ServiceHealth> {
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
    private async checkLlmServices(): Promise<{ [key: string]: ServiceHealth }> {
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
     * Check API health by testing the healthcheck endpoint itself
     */
    private async checkApi(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.apiHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            // Since we're already in the healthcheck, just verify the server is responding
            // by checking if we can bind to the port (server is listening)
            const serverResponding = SERVER_URL && API_URL;

            if (!serverResponding) {
                this.apiHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    API_SERVICE_CACHE_MS,
                    { error: "Server URLs not configured" },
                );
                return this.apiHealthCache.health;
            }

            // The fact that this healthcheck is running means the API is operational
            this.apiHealthCache = this.createServiceCache(
                ServiceStatus.Operational,
                API_SERVICE_CACHE_MS,
                {
                    serverUrl: SERVER_URL,
                    apiUrl: API_URL,
                    message: "API is operational (healthcheck endpoint is responding)",
                },
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
    private async checkWebSocket(): Promise<ServiceHealth> {
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
    private async checkI18n(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.i18nHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }
        try {
            // Check if i18next is initialized by attempting to use it
            // Modern i18next doesn't expose isInitialized property directly
            const hasCommonNamespace = i18next.hasLoadedNamespace("common");

            if (!hasCommonNamespace) {
                this.i18nHealthCache = this.createServiceCache(
                    ServiceStatus.Down,
                    DEFAULT_SERVICE_CACHE_MS,
                    { error: "i18next is not initialized - common namespace not loaded" },
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

                // Access language properties safely using type assertion
                // These properties exist at runtime but TypeScript doesn't know about them
                const i18nInstance = i18next as any;
                const language = i18nInstance.language || "unknown";
                const languages = i18nInstance.languages || [];

                // Try to get namespaces from options if available
                let namespaces: string[] = [];
                if (i18nInstance.options && i18nInstance.options.ns) {
                    namespaces = Array.isArray(i18nInstance.options.ns)
                        ? i18nInstance.options.ns
                        : [i18nInstance.options.ns];
                }

                this.i18nHealthCache = this.createServiceCache(
                    ServiceStatus.Operational,
                    DEFAULT_SERVICE_CACHE_MS,
                    {
                        language,
                        languages,
                        namespaces,
                        hasCommonNamespace,
                    },
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
        let imageProcessingHealthy = false;
        let s3Details: Record<string, unknown> = {};
        let nsfwDetails: Record<string, unknown> = {};
        let imageProcessingDetails: Record<string, unknown> = {};
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
        // Skip NSFW detection in local execution mode since the service isn't needed
        const executionMode = getExecutionMode();
        if (executionMode === "local") {
            nsfwDetectionHealthy = true;
            nsfwDetails = {
                status: "Disabled",
                reason: "NSFW detection is not required in local execution mode",
                executionMode,
            };
        } else {
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
        }

        // 3. Check image processing capabilities (Sharp)
        try {
            const imageProcessingStatus = getImageProcessingStatus();
            imageProcessingHealthy = imageProcessingStatus.available;
            imageProcessingDetails = {
                available: imageProcessingStatus.available,
                features: imageProcessingStatus.features,
                error: imageProcessingStatus.error,
            };
        } catch (error) {
            imageProcessingHealthy = false;
            imageProcessingDetails = {
                available: false,
                error: `Image processing check failed: ${(error as Error).message}`,
            };
        }

        // Determine overall status
        let overallStatus: ServiceStatus;
        if (s3Healthy && nsfwDetectionHealthy && imageProcessingHealthy) {
            overallStatus = ServiceStatus.Operational;
        } else if (s3Healthy && (nsfwDetectionHealthy || imageProcessingHealthy)) {
            // S3 is critical, but image processing can be degraded
            overallStatus = ServiceStatus.Degraded;
        } else if (!s3Healthy) {
            // S3 down means the whole service is down
            overallStatus = ServiceStatus.Down;
        } else {
            overallStatus = ServiceStatus.Degraded;
        }

        this.imageStorageHealthCache = this.createServiceCache(
            overallStatus,
            DEFAULT_SERVICE_CACHE_MS,
            {
                s3: { healthy: s3Healthy, ...s3Details },
                nsfwDetection: { healthy: nsfwDetectionHealthy, ...nsfwDetails },
                imageProcessing: { healthy: imageProcessingHealthy, ...imageProcessingDetails },
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
     * Check Resources health
     */
    private async checkResources(): Promise<ServiceHealth> {
        const cachedHealth = this.getCachedHealthIfValid(this.resourcesHealthCache);
        if (cachedHealth) {
            return cachedHealth;
        }

        try {
            const registry = ResourceRegistry.getInstance();
            const healthCheck = registry.getHealthCheck();

            // Map ResourceSystemHealth enum to ServiceStatus
            let status: ServiceStatus;
            switch (healthCheck.status) {
                case "Operational":
                    status = ServiceStatus.Operational;
                    break;
                case "Degraded":
                    status = ServiceStatus.Degraded;
                    break;
                case "Down":
                    status = ServiceStatus.Down;
                    break;
                default:
                    status = ServiceStatus.Down;
            }

            this.resourcesHealthCache = this.createServiceCache(
                status,
                DEFAULT_SERVICE_CACHE_MS,
                {
                    systemHealth: healthCheck.status,
                    totalSupported: healthCheck.stats.totalSupported,
                    totalRegistered: healthCheck.stats.totalRegistered,
                    totalEnabled: healthCheck.stats.totalEnabled,
                    totalActive: healthCheck.stats.totalActive,
                    categories: healthCheck.categories,
                    message: healthCheck.message,
                    missingImplementations: healthCheck.missingImplementations,
                    unavailableServices: healthCheck.unavailableServices,
                },
            );
            return this.resourcesHealthCache.health;
        } catch (error) {
            this.resourcesHealthCache = this.createServiceCache(
                ServiceStatus.Down,
                DEFAULT_SERVICE_CACHE_MS,
                { error: `Failed to check Resources: ${(error instanceof Error) ? error.message : String(error)}` },
            );
            return this.resourcesHealthCache.health;
        }
    }

    /**
     * Wrapper to run a health check with timeout
     */
    private async withTimeout<T>(check: () => Promise<T>, serviceName: string, defaultValue: T): Promise<T> {
        const timeout = new Promise<T>((_, reject) => {
            setTimeout(() => reject(new Error(`${serviceName} health check timed out`)), PER_SERVICE_TIMEOUT_MS);
        });

        try {
            return await Promise.race([check(), timeout]);
        } catch (error) {
            logger.warning(`${serviceName} health check timed out after ${PER_SERVICE_TIMEOUT_MS / SECONDS_1_MS} seconds`, { error });
            return defaultValue; // Return a default "Down" status or empty object for that service
        }
    }

    /**
     * Get overall system health status
     */
    public async getHealth(): Promise<SystemHealth> {

        const serviceChecks = [
            { name: "API", check: () => this.withTimeout(() => this.checkApi(), "API", createDownStatus("API", new Error("Timeout"))) },
            { name: "Bus", check: () => this.withTimeout(() => this.checkBus(), "Bus", createDownStatus("Bus", new Error("Timeout"))) },
            { name: "CronJobs", check: () => this.withTimeout(() => this.checkCronJobs(), "CronJobs", createDownStatus("CronJobs", new Error("Timeout"))) },
            { name: "Database", check: () => this.withTimeout(() => this.checkDatabase(), "Database", createDownStatus("Database", new Error("Timeout"))) },
            { name: "i18n", check: () => this.withTimeout(() => this.checkI18n(), "i18n", createDownStatus("i18n", new Error("Timeout"))) },
            { name: "LLMServices", check: () => this.withTimeout(() => this.checkLlmServices(), "LLMServices", {}) }, // Empty object for maps
            { name: "MCP", check: () => this.withTimeout(() => this.checkMcp(), "MCP", createDownStatus("MCP", new Error("Timeout"))) },
            { name: "Memory", check: () => this.withTimeout(() => this.checkMemory(), "Memory", createDownStatus("Memory", new Error("Timeout"))) },
            { name: "Queues", check: () => this.withTimeout(() => this.checkQueues(), "Queues", {}) }, // Empty object for maps
            { name: "Redis", check: () => this.withTimeout(() => this.checkRedis(), "Redis", createDownStatus("Redis", new Error("Timeout"))) },
            { name: "SSLCertificate", check: () => this.withTimeout(() => this.checkSslCertificate(), "SSLCertificate", createDownStatus("SSLCertificate", new Error("Timeout"))) },
            { name: "Stripe", check: () => this.withTimeout(() => this.checkStripe(), "Stripe", createDownStatus("Stripe", new Error("Timeout"))) },
            { name: "System", check: () => this.withTimeout(() => this.checkSystem(), "System", createDownStatus("System", new Error("Timeout"))) },
            { name: "WebSocket", check: () => this.withTimeout(() => this.checkWebSocket(), "WebSocket", createDownStatus("WebSocket", new Error("Timeout"))) },
            { name: "ImageStorage", check: () => this.withTimeout(() => this.checkImageStorage(), "ImageStorage", createDownStatus("ImageStorage", new Error("Timeout"))) },
            { name: "EmbeddingService", check: () => this.withTimeout(() => this.checkEmbeddingService(), "EmbeddingService", createDownStatus("EmbeddingService", new Error("Timeout"))) },
            { name: "Resources", check: () => this.withTimeout(() => this.checkResources(), "Resources", createDownStatus("Resources", new Error("Timeout"))) },
        ];

        const results: { name: string, status: "fulfilled" | "rejected", value?: any, reason?: any }[] = [];

        for (const { name, check } of serviceChecks) {
            try {
                const result = await check();
                results.push({ name, status: "fulfilled", value: result });
            } catch (error) {
                logger.error(`Health check failed for: ${name}`, { error });
                results.push({ name, status: "rejected", reason: error });
            }
        }

        // Helper to create a default 'Down' status for failed checks
        function createDownStatus(serviceName: string, error: unknown): ServiceHealth {
            // Aggressive error detail extraction - simplified for robust JSON serialization
            let serializableErrorDetails: object;
            if (error instanceof Error) {
                serializableErrorDetails = {
                    message: error.message,
                    name: error.name,
                    stack: isDebugMode() ? error.stack : "Stack trace available in development mode",
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
        const apiResult = results.find(r => r.name === "API");
        const apiHealth = apiResult?.status === "fulfilled" ? apiResult.value : createDownStatus("API", apiResult?.reason);
        const busResult = results.find(r => r.name === "Bus");
        const busHealth = busResult?.status === "fulfilled" ? busResult.value : createDownStatus("Bus", busResult?.reason);
        const cronJobsResult = results.find(r => r.name === "CronJobs");
        const cronJobsHealth = cronJobsResult?.status === "fulfilled" ? cronJobsResult.value : createDownStatus("Cron Jobs", cronJobsResult?.reason);
        const dbResult = results.find(r => r.name === "Database");
        const dbHealth = dbResult?.status === "fulfilled" ? dbResult.value : createDownStatus("Database", dbResult?.reason);
        const i18nResult = results.find(r => r.name === "i18n");
        const i18nHealth = i18nResult?.status === "fulfilled" ? i18nResult.value : createDownStatus("i18n", i18nResult?.reason);

        let llmHealth: { [key: string]: ServiceHealth } = {};
        const llmResult = results.find(r => r.name === "LLMServices");
        if (llmResult?.status === "fulfilled") {
            llmHealth = llmResult.value;
        } else if (llmResult?.status === "rejected") {
            logger.error("LLM health check failed", { trace: "health-llm-fail", error: llmResult.reason });
            // llmHealth remains empty or you could add a generic error entry
        }

        const mcpResult = results.find(r => r.name === "MCP");
        const mcpHealth = mcpResult?.status === "fulfilled" ? mcpResult.value : createDownStatus("MCP", mcpResult?.reason);
        const memoryResult = results.find(r => r.name === "Memory");
        const memoryHealth = memoryResult?.status === "fulfilled" ? memoryResult.value : createDownStatus("Memory", memoryResult?.reason);

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

        const redisResult = results.find(r => r.name === "Redis");
        const redisHealth = redisResult?.status === "fulfilled" ? redisResult.value : createDownStatus("Redis", redisResult?.reason);
        const sslResult = results.find(r => r.name === "SSLCertificate");
        const sslHealth = sslResult?.status === "fulfilled" ? sslResult.value : createDownStatus("SSL", sslResult?.reason);
        const stripeResult = results.find(r => r.name === "Stripe");
        const stripeHealth = stripeResult?.status === "fulfilled" ? stripeResult.value : createDownStatus("Stripe", stripeResult?.reason);
        const systemResult = results.find(r => r.name === "System");
        const systemHealth = systemResult?.status === "fulfilled" ? systemResult.value : createDownStatus("System", systemResult?.reason);
        const websocketResult = results.find(r => r.name === "WebSocket");
        const websocketHealth = websocketResult?.status === "fulfilled" ? websocketResult.value : createDownStatus("WebSocket", websocketResult?.reason);
        const imageStorageResult = results.find(r => r.name === "ImageStorage");
        const imageStorageHealth = imageStorageResult?.status === "fulfilled" ? imageStorageResult.value : createDownStatus("Image Storage", imageStorageResult?.reason);
        const embeddingServiceResult = results.find(r => r.name === "EmbeddingService");
        const embeddingServiceHealth = embeddingServiceResult?.status === "fulfilled" ? embeddingServiceResult.value : createDownStatus("Embedding Service", embeddingServiceResult?.reason);
        const resourcesResult = results.find(r => r.name === "Resources");
        const resourcesHealth = resourcesResult?.status === "fulfilled" ? resourcesResult.value : createDownStatus("Resources", resourcesResult?.reason);


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
            resourcesHealth,
            // Spread results from checks that return an object of ServiceHealth
            ...(Object.values(queueHealths) as ServiceHealth[]),
            ...(Object.values(llmHealth) as ServiceHealth[]),
        ].filter(s => s !== undefined && s !== null); // Filter out undefined/null if createDownStatus can return them or if a check isn't found

        const overallStatus = allServices.every(s => s?.status === ServiceStatus.Operational) ? ServiceStatus.Operational :
            allServices.some(s => s?.status === ServiceStatus.Down) ? ServiceStatus.Down : ServiceStatus.Degraded;

        // Health check completed

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
                resources: resourcesHealth,
            },
            timestamp: Date.now(),
        };

        return health;
    }
}

// Track ongoing health check to prevent duplicate execution
let ongoingHealthCheck: Promise<SystemHealth> | null = null;

/**
 * Setup the health check endpoint.
 */
export function setupHealthCheck(app: Express): void {
    app.get("/healthcheck", async (_req, res) => {
        const HEALTH_CHECK_TIMEOUT_MS = 30000; // 30 seconds

        // If a health check is already in progress, wait for it instead of starting a new one
        if (ongoingHealthCheck) {
            logger.info("Reusing ongoing health check instead of starting duplicate");
            try {
                const health = await ongoingHealthCheck;
                res.status(HttpStatus.Ok).json(health);
                return;
            } catch (error) {
                // If the ongoing check failed, continue with a new check
                logger.warn("Ongoing health check failed, starting new check", { error });
            }
        }

        const timeoutPromise = new Promise((_, reject) => {
            setTimeout(() => {
                reject(new Error("Health check timed out"));
            }, HEALTH_CHECK_TIMEOUT_MS);
        });

        try {
            // Start new health check and store the promise
            ongoingHealthCheck = Promise.race([
                HealthService.get().getHealth(),
                timeoutPromise,
            ]) as Promise<SystemHealth>;

            const health = await ongoingHealthCheck;

            // Clear the ongoing check after successful completion
            ongoingHealthCheck = null;

            res.status(HttpStatus.Ok).json(health);
        } catch (error) {
            // Clear the ongoing check on failure
            ongoingHealthCheck = null;

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

    if (isDebugMode()) {
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
        // Example: http://localhost:5329/healthcheck/send-test-notification?userId=1234567890123456789
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
