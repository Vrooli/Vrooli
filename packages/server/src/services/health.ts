import { HeadBucketCommand, S3Client } from "@aws-sdk/client-s3";
import { DAYS_1_S, GB_1_BYTES, HttpStatus, SECONDS_1_MS, SERVER_VERSION, endpointsAuth, endpointsFeed } from "@local/shared";
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
    status: ServiceStatus;
    lastChecked: number;
    details?: object;
}

interface SystemHealth {
    status: ServiceStatus;
    version: string;
    services: {
        api: ServiceHealth;
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

const CACHE_TIMEOUT_MS = 120_000;

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
            // First check if we're connected at all
            if (!DbProvider.isConnected()) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: {
                        connection: false,
                        message: "Database not connected",
                    },
                };
            }

            // Then try a simple query to verify connection is still active
            try {
                await DbProvider.get().$queryRaw`SELECT 1`;
            } catch (queryError) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: {
                        connection: false,
                        message: "Database connection lost",
                        error: (queryError as Error).message,
                    },
                };
            }

            // Check if seeding was successful
            const seedingSuccessful = DbProvider.isSeedingSuccessful();
            const isRetrying = DbProvider.isRetryingSeeding();
            const retryCount = DbProvider.getSeedRetryCount();
            const attemptCount = DbProvider.getSeedAttemptCount();
            const maxRetries = DbProvider.getMaxRetries();

            // If connection works but seeding failed
            if (!seedingSuccessful) {
                return {
                    healthy: false,
                    status: isRetrying ? ServiceStatus.Degraded : ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: {
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
                    },
                };
            }

            return {
                healthy: true,
                status: ServiceStatus.Operational,
                lastChecked: Date.now(),
                details: {
                    connection: true,
                    seeding: true,
                    attemptCount,
                },
            };
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    connection: false,
                    message: "Failed to check database health",
                    error: (error as Error).message,
                },
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
            const healthDetails = await queue.checkHealth();
            queueHealth[name] = {
                healthy: healthDetails.status === QueueStatus.Healthy,
                status: healthDetails.status === QueueStatus.Healthy ? ServiceStatus.Operational :
                    healthDetails.status === QueueStatus.Degraded ? ServiceStatus.Degraded : ServiceStatus.Down,
                lastChecked: Date.now(),
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

        return queueHealth;
    }

    /**
     * Check memory usage health
     */
    private checkMemory(): ServiceHealth {
        const used = process.memoryUsage();
        const heapUsedPercent = (used.heapUsed / used.heapTotal) * 100;
        const memoryStatus = used.heapUsed > MEMORY_CRITICAL_THRESHOLD ? ServiceStatus.Down :
            used.heapUsed > MEMORY_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                ServiceStatus.Operational;

        return {
            healthy: memoryStatus === ServiceStatus.Operational,
            status: memoryStatus,
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
            const systemStatus = cpuUsage > CPU_CRITICAL_THRESHOLD || diskUsagePercent > DISK_CRITICAL_THRESHOLD ? ServiceStatus.Down :
                cpuUsage > CPU_WARNING_THRESHOLD || diskUsagePercent > DISK_WARNING_THRESHOLD ? ServiceStatus.Degraded :
                    ServiceStatus.Operational;

            return {
                healthy: systemStatus === ServiceStatus.Operational,
                status: systemStatus,
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
     * Check MCP service health by performing an end-to-end transport + RPC test
     */
    private async checkMcp(): Promise<ServiceHealth> {
        const checkTimestamp = Date.now();
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

        return {
            healthy: overallStatus === ServiceStatus.Operational,
            status: overallStatus,
            lastChecked: checkTimestamp,
            details: {
                transport: { healthy: transportHealthy, ...transportDetails },
                builtInTools: { healthy: builtInHealthy, ...builtInDetails },
                routineTools: { healthy: routineHealthy, ...routineDetails },
            },
        };
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

            return {
                healthy: apiStatus === ServiceStatus.Operational,
                status: apiStatus,
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
     * Check WebSocket server health using SocketService
     */
    private checkWebSocket(): ServiceHealth {
        try {
            // Get the singleton instance
            const socketService = SocketService.get();
            // Get health details from the service
            const details = socketService.getHealthDetails();

            if (details) {
                return {
                    healthy: true,
                    status: ServiceStatus.Operational,
                    lastChecked: Date.now(),
                    details,
                };
            } else {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: { error: "WebSocket service component not fully initialized." },
                };
            }
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: { error: (error as Error).message || "Failed to check WebSocket server status" },
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
            const cronStatus = hasCriticalFailures ? ServiceStatus.Down :
                hasFailures ? ServiceStatus.Degraded : ServiceStatus.Operational;

            return {
                healthy: cronStatus === ServiceStatus.Operational,
                status: cronStatus,
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
     * Check i18next initialization
     */
    private checkI18n(): ServiceHealth {
        try {
            // Check if i18next is initialized
            if (!i18next.isInitialized) {
                return {
                    healthy: false,
                    status: ServiceStatus.Down,
                    lastChecked: Date.now(),
                    details: {
                        message: "i18next is not initialized",
                    },
                };
            }

            // Check if we can translate a basic key
            try {
                const translation = i18next.t("common:Yes");
                const isTranslated = translation !== "common:Yes" && typeof translation === "string" && translation.length > 0;

                if (!isTranslated) {
                    return {
                        healthy: false,
                        status: ServiceStatus.Degraded,
                        lastChecked: Date.now(),
                        details: {
                            message: "i18next is initialized but translations are not working properly",
                        },
                    };
                }

                return {
                    healthy: true,
                    status: ServiceStatus.Operational,
                    lastChecked: Date.now(),
                    details: {
                        language: i18next.language,
                        languages: i18next.languages,
                        namespaces: i18next.options.ns,
                    },
                };
            } catch (error) {
                return {
                    healthy: false,
                    status: ServiceStatus.Degraded,
                    lastChecked: Date.now(),
                    details: {
                        message: "i18next is initialized but failed to translate",
                        error: (error as Error).message,
                    },
                };
            }
        } catch (error) {
            return {
                healthy: false,
                status: ServiceStatus.Down,
                lastChecked: Date.now(),
                details: {
                    message: "Failed to check i18next",
                    error: (error as Error).message,
                },
            };
        }
    }

    /**
     * Check image storage health (S3 and NSFW detection)
     */
    private async checkImageStorage(): Promise<ServiceHealth> {
        const checkTimestamp = Date.now();
        let s3Healthy = false;
        let nsfwDetectionHealthy = false;
        let s3Details: Record<string, unknown> = {};
        let nsfwDetails: Record<string, unknown> = {};
        let s3Client: S3Client | undefined;

        // 1. Check S3 Client and Bucket Access
        try {
            s3Client = getS3Client(); // Make sure getS3Client is exported from fileStorage.ts
            if (!s3Client) {
                throw new Error("S3 client not initialized");
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
                throw new Error(`checkNSFW returned non-boolean value: ${typeof isNsfwResult}`);
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

        return {
            healthy: overallStatus === ServiceStatus.Operational,
            status: overallStatus,
            lastChecked: checkTimestamp,
            details: {
                s3: { healthy: s3Healthy, ...s3Details },
                nsfwDetection: { healthy: nsfwDetectionHealthy, ...nsfwDetails },
            },
        };
    }

    /**
     * Get overall system health status
     */
    public async getHealth(): Promise<SystemHealth> {
        // Return cached result if valid
        if (this.isCacheValid() && this.lastCheck) {
            return this.lastCheck;
        }

        // Perform health checks in parallel
        const results = await Promise.allSettled([
            this.checkApi(),                   // 0
            this.checkCronJobs(),              // 1
            this.checkDatabase(),              // 2
            this.checkI18n(),                  // 3
            this.checkLlmServices(),           // 4
            this.checkMcp(),                   // 5
            this.checkMemory(),                // 6
            this.checkQueues(),                // 7
            this.checkRedis(),                 // 8
            this.checkSslCertificate(),        // 9
            this.checkStripe(),                // 10
            this.checkSystem(),                // 11
            this.checkWebSocket(),             // 12
            this.checkImageStorage(),          // 13
            EmbeddingService.get().checkHealth(), // 14
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
        const cronJobsHealth = results[1].status === "fulfilled" ? results[1].value : createDownStatus("Cron Jobs", results[1].reason);
        const dbHealth = results[2].status === "fulfilled" ? results[2].value : createDownStatus("Database", results[2].reason);
        const i18nHealth = results[3].status === "fulfilled" ? results[3].value : createDownStatus("i18n", results[3].reason);
        const llmHealth = results[4].status === "fulfilled" ? results[4].value : {}; // Handled differently as it's an object
        if (results[4].status === "rejected") logger.error("LLM health check failed", { trace: "health-llm-fail", error: results[4].reason });
        const mcpHealth = results[5].status === "fulfilled" ? results[5].value : createDownStatus("MCP", results[5].reason);
        const memoryHealth = results[6].status === "fulfilled" ? results[6].value : createDownStatus("Memory", results[6].reason);
        const queueHealths = results[7].status === "fulfilled" ? results[7].value : {}; // Handled differently as it's an object
        if (results[7].status === "rejected") logger.error("Queue health check failed", { trace: "health-queue-fail", error: results[7].reason });
        const redisHealth = results[8].status === "fulfilled" ? results[8].value : createDownStatus("Redis", results[8].reason);
        const sslHealth = results[9].status === "fulfilled" ? results[9].value : createDownStatus("SSL", results[9].reason);
        const stripeHealth = results[10].status === "fulfilled" ? results[10].value : createDownStatus("Stripe", results[10].reason);
        const systemHealth = results[11].status === "fulfilled" ? results[11].value : createDownStatus("System", results[11].reason);
        const websocketHealth = results[12].status === "fulfilled" ? results[12].value : createDownStatus("WebSocket", results[12].reason);
        const imageStorageHealth = results[13].status === "fulfilled" ? results[13].value : createDownStatus("Image Storage", results[13].reason);
        const embeddingServiceHealth = results[14].status === "fulfilled" ? results[14].value : createDownStatus("Embedding Service", results[14].reason);


        // Determine overall status
        const allServices = [
            apiHealth,
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
