/**
 * Browserless Browser Automation Resource Implementation
 * 
 * Provides infrastructure-level management for browserless Chrome automation including:
 * - Service discovery and health monitoring
 * - Browser instance management and availability
 * - Connection configuration and validation
 * - Performance metrics and resource usage
 * - Integration with unified configuration system
 */

import { SECONDS_30_MS, SECONDS_5_MS } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { AgentMetadata, BrowserlessConfig } from "../typeRegistry.js";
import type { HealthCheckResult, IAgentResource, ResourceEventData } from "../types.js";
import { DeploymentType, DiscoveryStatus, ResourceCategory, ResourceEvent } from "../types.js";

// Default maximum concurrent browser instances
const DEFAULT_MAX_CONCURRENT = 10;

/**
 * Browserless stats information
 */
interface BrowserlessStats {
    date: number;
    successful: number;
    error: number;
    queued: number;
    rejected: number;
    timedout: number;
    totalTime: number;
    meanTime: number;
    maxTime: number;
    minTime: number;
    maxConcurrent: number;
    sessionTimes: number[];
    cpu: number;
    memory: number;
}

/**
 * Browserless pressure information
 */
interface BrowserlessPressure {
    date: number;
    running: number;
    queued: number;
    maxConcurrent: number;
    maxQueued: number;
    cpu: number;
    memory: number;
    reason: string;
    message?: string[];
}

/**
 * Browserless config information
 */
interface BrowserlessConfigInfo {
    allowCors: boolean;
    allowGetCalls: boolean;
    concurrent: number;
    timeout: number;
    queued: number;
    retries: number;
    hooks: any;
}

/**
 * Production Browserless resource implementation with enhanced monitoring and browser management
 */
@RegisterResource
export class BrowserlessResource extends ResourceProvider<"browserless", BrowserlessConfig> implements IAgentResource {
    // Resource identification
    readonly id = "browserless";
    readonly category = ResourceCategory.Agents;
    readonly displayName = "Browserless";
    readonly description = "Chrome automation service for headless browser operations";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Browser instance tracking
    private activeInstances = 0;
    private queuedInstances = 0;
    private lastStatsRefresh = 0;
    private stats?: BrowserlessStats;
    private pressure?: BrowserlessPressure;
    private configInfo?: BrowserlessConfigInfo;
    private readonly STATS_CACHE_TTL_MS = SECONDS_30_MS;

    /**
     * Enhanced discovery with browserless-specific checks
     */
    protected async performDiscovery(): Promise<boolean> {
        try {
            logger.debug("[BrowserlessResource] Starting service discovery");

            // First check if browserless is responding
            const pressureResult = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/pressure`,
                method: "GET",
                timeout: SECONDS_5_MS,
                auth: this.getAuthConfig(),
            });

            if (!pressureResult.success) {
                logger.debug(`[BrowserlessResource] Pressure endpoint failed: ${pressureResult.error}`);
                return false;
            }

            // Store pressure info
            this.pressure = pressureResult.data;
            this.activeInstances = this.pressure?.running || 0;
            this.queuedInstances = this.pressure?.queued || 0;

            // Try to get config information
            try {
                const configResult = await this.httpClient!.makeRequest({
                    url: `${this.config!.baseUrl}/config`,
                    method: "GET",
                    timeout: SECONDS_5_MS,
                    auth: this.getAuthConfig(),
                });

                if (configResult.success && configResult.data) {
                    this.configInfo = configResult.data;
                    logger.debug("[BrowserlessResource] Retrieved config information", this.configInfo);
                }
            } catch (error) {
                logger.debug("[BrowserlessResource] Config endpoint not accessible");
            }

            // Try to get stats
            await this.refreshStats();

            logger.info("[BrowserlessResource] Discovered browserless service", {
                baseUrl: this.config!.baseUrl,
                activeInstances: this.activeInstances,
                queuedInstances: this.queuedInstances,
                maxConcurrent: this.configInfo?.concurrent || this.config!.concurrent,
            });

            // Emit discovery event
            this.emitResourceEvent(ResourceEvent.Discovered, {
                activeInstances: this.activeInstances,
                queuedInstances: this.queuedInstances,
                maxConcurrent: this.configInfo?.concurrent || this.config!.concurrent,
            });

            return true;
        } catch (error) {
            logger.debug(`[BrowserlessResource] Discovery failed: ${error}`);
            return false;
        }
    }

    /**
     * Enhanced health check with detailed metrics
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        try {
            const startTime = Date.now();

            // Check pressure endpoint for current status
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/pressure`,
                method: "GET",
                timeout: this.config!.healthCheck?.timeoutMs || SECONDS_5_MS,
                auth: this.getAuthConfig(),
            });

            const responseTime = Date.now() - startTime;

            if (result.success && result.data) {
                this.pressure = result.data;
                this.activeInstances = this.pressure?.running || 0;
                this.queuedInstances = this.pressure?.queued || 0;

                // Refresh stats if stale
                if (Date.now() - this.lastStatsRefresh > this.STATS_CACHE_TTL_MS) {
                    await this.refreshStats();
                }

                const isUnderPressure = this.pressure?.reason !== "OK";
                const maxConcurrent = this.configInfo?.concurrent || this.config!.concurrent || DEFAULT_MAX_CONCURRENT;
                const utilization = (this.activeInstances / maxConcurrent) * 100;

                const details = {
                    activeInstances: this.activeInstances,
                    queuedInstances: this.queuedInstances,
                    maxConcurrent,
                    utilization: `${utilization.toFixed(1)}%`,
                    cpu: `${this.pressure?.cpu || 0}%`,
                    memory: `${this.pressure?.memory || 0}%`,
                    reason: this.pressure?.reason || "Unknown",
                    messages: this.pressure?.message,
                    responseTime,
                    baseUrl: this.config!.baseUrl,
                    stats: this.stats ? {
                        successful: this.stats.successful,
                        error: this.stats.error,
                        timedout: this.stats.timedout,
                        meanTime: this.stats.meanTime,
                    } : undefined,
                };

                logger.debug("[BrowserlessResource] Health check passed", details);

                return {
                    healthy: !isUnderPressure || this.pressure?.reason === "OK",
                    message: isUnderPressure
                        ? `Browserless under pressure: ${this.pressure?.reason || "Unknown"} (${this.activeInstances}/${maxConcurrent} active)`
                        : `Browserless healthy (${this.activeInstances}/${maxConcurrent} active, ${responseTime}ms response)`,
                    details,
                    timestamp: new Date(),
                };
            }

            const errorDetails = {
                status: result.status,
                error: result.error,
                responseTime,
                baseUrl: this.config!.baseUrl,
            };

            logger.warn("[BrowserlessResource] Health check failed", errorDetails);

            return {
                healthy: false,
                message: `Browserless health check failed: ${result.error}`,
                details: errorDetails,
                timestamp: new Date(),
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : "Unknown health check error";

            logger.error(`[BrowserlessResource] Health check error: ${errorMessage}`, error);

            return {
                healthy: false,
                message: errorMessage,
                details: { error: errorMessage },
                timestamp: new Date(),
            };
        }
    }

    /**
     * Refresh stats from browserless
     */
    private async refreshStats(): Promise<void> {
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/stats`,
                method: "GET",
                timeout: 5000,
                auth: this.getAuthConfig(),
            });

            if (result.success && result.data) {
                this.stats = result.data;
                this.lastStatsRefresh = Date.now();

                logger.debug("[BrowserlessResource] Refreshed stats", {
                    successful: this.stats?.successful || 0,
                    error: this.stats?.error || 0,
                    meanTime: this.stats?.meanTime || 0,
                });
            }
        } catch (error) {
            logger.debug("[BrowserlessResource] Failed to refresh stats", error);
        }
    }

    /**
     * Get browser endpoint - required by IAgentResource
     */
    getBrowserEndpoint(): string {
        return this.config?.browserEndpoint || `${this.config?.baseUrl}/chrome`;
    }

    /**
     * Get WebSocket endpoint for browser connections - required by IAgentResource
     */
    getWebSocketEndpoint(): string {
        const baseUrl = this.config?.baseUrl || "";
        // Convert http/https to ws/wss
        const wsUrl = baseUrl.replace(/^http/, "ws");
        return `${wsUrl}/chrome`;
    }

    /**
     * Get base URL for making HTTP requests
     */
    getBaseUrl(): string | undefined {
        return this.config?.baseUrl;
    }

    /**
     * Get authentication token
     */
    getToken(): string | undefined {
        return this.config?.token;
    }

    /**
     * Check if can accept new browser instance - required by IAgentResource
     */
    async canAcceptInstance(): Promise<boolean> {
        if (this._status !== DiscoveryStatus.Available) {
            return false;
        }

        const maxConcurrent = this.configInfo?.concurrent || this.config?.concurrent || 10;
        return this.activeInstances < maxConcurrent;
    }

    /**
     * Get current instance count - required by IAgentResource
     */
    async getInstanceCount(): Promise<number> {
        return this.activeInstances;
    }

    /**
     * Get current number of active instances - required by IAgentResource
     */
    getActiveInstances(): number {
        return this.activeInstances;
    }

    /**
     * Check if can spawn new instance - required by IAgentResource
     */
    canSpawnInstance(): boolean {
        if (this._status !== DiscoveryStatus.Available) {
            return false;
        }

        const maxConcurrent = this.configInfo?.concurrent || this.config?.concurrent || 10;
        return this.activeInstances < maxConcurrent;
    }

    /**
     * Create a browser instance
     */
    async createBrowserInstance(options?: {
        headless?: boolean;
        timeout?: number;
        args?: string[];
    }): Promise<{
        success: boolean;
        browserWSEndpoint?: string;
        error?: string;
    }> {
        if (this._status !== DiscoveryStatus.Available) {
            return { success: false, error: "Browserless is not available" };
        }

        try {
            const body = {
                headless: options?.headless ?? this.config?.headless ?? true,
                timeout: options?.timeout ?? this.config?.timeout ?? 30000,
                args: options?.args,
            };

            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/chrome`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body,
                timeout: (options?.timeout ?? this.config?.timeout ?? 30000) + 5000, // Add buffer
                auth: this.getAuthConfig(),
            });

            if (result.success && result.data?.browserWSEndpoint) {
                logger.info("[BrowserlessResource] Created browser instance");

                // Update active instances count
                this.activeInstances++;

                return {
                    success: true,
                    browserWSEndpoint: result.data.browserWSEndpoint,
                };
            }

            return {
                success: false,
                error: result.error || "Failed to create browser instance",
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error("[BrowserlessResource] Failed to create browser instance", error);
            return { success: false, error: errorMessage };
        }
    }

    /**
     * Execute a function in the browser context
     */
    async executeFunction(code: string, context?: any): Promise<{
        success: boolean;
        result?: any;
        error?: string;
    }> {
        if (this._status !== DiscoveryStatus.Available) {
            return { success: false, error: "Browserless is not available" };
        }

        try {
            const body = {
                code,
                context,
            };

            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/chrome/function`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body,
                timeout: this.config?.timeout ?? 30000,
                auth: this.getAuthConfig(),
            });

            if (result.success) {
                return {
                    success: true,
                    result: result.data,
                };
            }

            return {
                success: false,
                error: result.error || "Failed to execute function",
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error("[BrowserlessResource] Failed to execute function", error);
            return { success: false, error: errorMessage };
        }
    }

    /**
     * Take a screenshot of a URL
     */
    async screenshot(url: string, options?: {
        fullPage?: boolean;
        width?: number;
        height?: number;
        type?: "png" | "jpeg";
        quality?: number;
    }): Promise<{
        success: boolean;
        screenshot?: Buffer;
        error?: string;
    }> {
        if (this._status !== DiscoveryStatus.Available) {
            return { success: false, error: "Browserless is not available" };
        }

        try {
            const body = {
                url,
                options: {
                    fullPage: options?.fullPage,
                    viewport: options?.width && options?.height ? {
                        width: options.width,
                        height: options.height,
                    } : undefined,
                    type: options?.type,
                    quality: options?.quality,
                },
            };

            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/chrome/screenshot`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body,
                timeout: this.config?.timeout ?? 30000,
                auth: this.getAuthConfig(),
            });

            if (result.success && result.data) {
                return {
                    success: true,
                    screenshot: result.data,
                };
            }

            return {
                success: false,
                error: result.error || "Failed to take screenshot",
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error("[BrowserlessResource] Failed to take screenshot", error);
            return { success: false, error: errorMessage };
        }
    }

    /**
     * Provide enhanced metadata about the resource
     */
    protected getMetadata(): AgentMetadata {
        return {
            version: "2.0", // Browserless v2
            capabilities: [
                "headless-chrome",
                "screenshot",
                "pdf-generation",
                "function-execution",
                "web-scraping",
                "automation",
                "puppeteer-compatible",
                "playwright-compatible",
                "concurrent-sessions",
                "request-interception",
                "cookie-management",
                "proxy-support",
            ],
            activeInstances: this.activeInstances,
            maxInstances: this.configInfo?.concurrent || this.config?.concurrent || 10,
            supportedBrowsers: ["chromium", "chrome"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            // Browserless-specific metadata
            queuedInstances: this.queuedInstances,
            stats: this.stats ? {
                successful: this.stats.successful,
                error: this.stats.error,
                timedout: this.stats.timedout,
                meanTime: this.stats.meanTime,
                cpu: this.stats.cpu,
                memory: this.stats.memory,
            } : undefined,
            pressure: this.pressure ? {
                reason: this.pressure.reason,
                cpu: this.pressure.cpu,
                memory: this.pressure.memory,
            } : undefined,
            timeout: this.configInfo?.timeout || this.config?.timeout,
            baseUrl: this.config?.baseUrl,
        };
    }

    /**
     * Get authentication configuration
     */
    protected getAuthConfig() {
        if (!this.config?.token) return undefined;

        return {
            type: "bearer" as const,
            token: this.config.token,
        };
    }

    /**
     * Emit resource events with proper typing
     */
    private emitResourceEvent(event: ResourceEvent, details?: Record<string, any>): void {
        const eventData: ResourceEventData = {
            resourceId: this.id,
            category: this.category,
            event,
            timestamp: new Date(),
            details,
        };

        this.emit(event, eventData);
    }

    /**
     * Enhanced shutdown with cleanup
     */
    async shutdown(): Promise<void> {
        logger.info("[BrowserlessResource] Shutting down browserless resource");

        // Clear state
        this.activeInstances = 0;
        this.queuedInstances = 0;
        this.lastStatsRefresh = 0;
        this.stats = undefined;
        this.pressure = undefined;
        this.configInfo = undefined;

        // Call parent shutdown
        await super.shutdown();

        logger.info("[BrowserlessResource] Browserless resource shutdown complete");
    }
}
