/**
 * Puppeteer Browser Automation Resource Implementation
 * 
 * Provides infrastructure-level management for headless browser automation including:
 * - Browser instance lifecycle management
 * - Resource limits and instance tracking
 * - Health monitoring with performance metrics
 * - Integration with Puppeteer HTTP service wrapper
 * - Support for screenshots, PDFs, web scraping, and custom automation
 */

import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus, ResourceEvent } from "../types.js";
import type { HealthCheckResult, IAgentResource, ResourceEventData } from "../types.js";
import type { PuppeteerConfig, AgentMetadata } from "../typeRegistry.js";
import { logger } from "../../../events/logger.js";

/**
 * Health response from Puppeteer service
 */
interface PuppeteerHealthResponse {
    status: string;
    timestamp: string;
    uptime: number;
    cluster: {
        idle: boolean;
        closed: boolean;
        activeInstances: number;
        maxInstances: number;
        totalInstances: number;
    };
    metrics: {
        totalRequests: number;
        failedRequests: number;
        successRate: string;
    };
}

/**
 * Info response from Puppeteer service
 */
interface PuppeteerInfoResponse {
    service: string;
    version: string;
    capabilities: string[];
    config: {
        maxConcurrency: number;
        headless: boolean;
        timeout: number;
    };
}

/**
 * Browser action types supported by the service
 */
export enum BrowserAction {
    Screenshot = "screenshot",
    PDF = "pdf",
    Scrape = "scrape",
    Execute = "execute",
}

/**
 * Options for screenshot capture
 */
export interface ScreenshotOptions {
    url: string;
    options?: {
        type?: "png" | "jpeg";
        quality?: number;
        fullPage?: boolean;
        clip?: {
            x: number;
            y: number;
            width: number;
            height: number;
        };
    };
}

/**
 * Options for PDF generation
 */
export interface PDFOptions {
    url?: string;
    html?: string;
    options?: {
        format?: "A4" | "Letter" | "Legal";
        printBackground?: boolean;
        margin?: {
            top?: string;
            bottom?: string;
            left?: string;
            right?: string;
        };
    };
}

/**
 * Options for web scraping
 */
export interface ScrapeOptions {
    url: string;
    selector?: string;
    script?: string;
    waitFor?: string | number;
}

/**
 * Step for custom browser automation
 */
export interface AutomationStep {
    action: "click" | "type" | "select" | "waitForSelector" | "evaluate";
    selector?: string;
    value?: string;
    script?: string;
}

/**
 * Options for custom automation execution
 */
export interface ExecuteOptions {
    url: string;
    steps: AutomationStep[];
}

/**
 * Production Puppeteer resource implementation with comprehensive browser management
 */
@RegisterResource
export class PuppeteerResource extends ResourceProvider<"puppeteer", PuppeteerConfig> implements IAgentResource {
    // Resource identification
    readonly id = "puppeteer";
    readonly category = ResourceCategory.Agents;
    readonly displayName = "Puppeteer";
    readonly description = "Headless browser automation with Chrome/Chromium";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;
    
    // Instance tracking
    private activeInstances = 0;
    private maxInstances = 5;
    private totalInstancesCreated = 0;
    private lastInstanceCheck = Date.now();
    
    // Service information
    private serviceInfo?: PuppeteerInfoResponse;
    private serviceVersion?: string;
    
    // Performance metrics
    private metrics = {
        totalRequests: 0,
        failedRequests: 0,
        averageResponseTime: 0,
        lastResponseTimes: [] as number[],
    };
    
    /**
     * Enhanced discovery with Puppeteer-specific checks
     */
    protected async performDiscovery(): Promise<boolean> {
        try {
            logger.debug("[PuppeteerResource] Starting service discovery");
            
            // First check if service is responding
            const healthResult = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/health`,
                method: "GET",
                timeout: 5000,
            });
            
            if (!healthResult.success) {
                logger.debug(`[PuppeteerResource] Health endpoint failed: ${healthResult.error}`);
                return false;
            }
            
            // Parse health response
            const healthData = healthResult.data as PuppeteerHealthResponse;
            this.activeInstances = healthData.cluster.activeInstances;
            this.maxInstances = healthData.cluster.maxInstances || this.config!.maxInstances || 5;
            this.totalInstancesCreated = healthData.cluster.totalInstances;
            
            // Get service info
            try {
                const infoResult = await this.httpClient!.makeRequest({
                    url: `${this.config!.baseUrl}/info`,
                    method: "GET",
                    timeout: 3000,
                });
                
                if (infoResult.success && infoResult.data) {
                    this.serviceInfo = infoResult.data as PuppeteerInfoResponse;
                    this.serviceVersion = this.serviceInfo.version;
                    logger.debug("[PuppeteerResource] Retrieved service information", this.serviceInfo);
                }
            } catch (error) {
                logger.debug("[PuppeteerResource] Info endpoint not accessible", error);
            }
            
            logger.info("[PuppeteerResource] Discovered Puppeteer service", {
                version: this.serviceVersion,
                baseUrl: this.config!.baseUrl,
                activeInstances: this.activeInstances,
                maxInstances: this.maxInstances,
            });
            
            // Emit discovery event
            this.emitResourceEvent(ResourceEvent.Discovered, {
                version: this.serviceVersion,
                activeInstances: this.activeInstances,
                maxInstances: this.maxInstances,
                capabilities: this.serviceInfo?.capabilities,
            });
            
            return true;
        } catch (error) {
            logger.debug(`[PuppeteerResource] Discovery failed: ${error}`);
            return false;
        }
    }
    
    /**
     * Enhanced health check with detailed metrics
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        try {
            const startTime = Date.now();
            
            // Check health endpoint
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/health`,
                method: "GET",
                timeout: this.config!.healthCheck?.timeoutMs || 5000,
            });
            
            const responseTime = Date.now() - startTime;
            this.updateResponseTimeMetrics(responseTime);
            
            if (result.success) {
                const healthData = result.data as PuppeteerHealthResponse;
                
                // Update instance tracking
                this.activeInstances = healthData.cluster.activeInstances;
                this.totalInstancesCreated = healthData.cluster.totalInstances;
                this.lastInstanceCheck = Date.now();
                
                // Parse metrics
                const requestMetrics = healthData.metrics;
                
                const details = {
                    activeInstances: this.activeInstances,
                    maxInstances: this.maxInstances,
                    totalInstances: this.totalInstancesCreated,
                    idle: healthData.cluster.idle,
                    uptime: healthData.uptime,
                    totalRequests: requestMetrics.totalRequests,
                    failedRequests: requestMetrics.failedRequests,
                    successRate: requestMetrics.successRate,
                    responseTime,
                    averageResponseTime: this.metrics.averageResponseTime,
                    baseUrl: this.config!.baseUrl,
                    headless: this.config!.headless !== false,
                };
                
                logger.debug("[PuppeteerResource] Health check passed", details);
                
                return {
                    healthy: true,
                    message: `Puppeteer is healthy (${this.activeInstances}/${this.maxInstances} instances active, ${responseTime}ms response)`,
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
            
            logger.warn("[PuppeteerResource] Health check failed", errorDetails);
            
            return {
                healthy: false,
                message: `Puppeteer health check failed: ${result.error}`,
                details: errorDetails,
                timestamp: new Date(),
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : "Unknown health check error";
            
            logger.error(`[PuppeteerResource] Health check error: ${errorMessage}`, error);
            
            return {
                healthy: false,
                message: errorMessage,
                details: { error: errorMessage },
                timestamp: new Date(),
            };
        }
    }
    
    /**
     * Get current number of active browser instances - required by IAgentResource
     */
    getActiveInstances(): number {
        // Return cached value if recent
        if (Date.now() - this.lastInstanceCheck < 5000) {
            return this.activeInstances;
        }
        
        // Otherwise return last known value (will be updated on next health check)
        return this.activeInstances;
    }
    
    /**
     * Check if can spawn new browser instance - required by IAgentResource
     */
    canSpawnInstance(): boolean {
        return this._status === DiscoveryStatus.Available && 
               this.activeInstances < this.maxInstances;
    }
    
    /**
     * Take a screenshot of a webpage
     */
    async screenshot(options: ScreenshotOptions): Promise<Buffer> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Puppeteer service is not available");
        }
        
        if (!this.canSpawnInstance()) {
            throw new Error(`Instance limit reached (${this.activeInstances}/${this.maxInstances})`);
        }
        
        this.metrics.totalRequests++;
        const startTime = Date.now();
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/screenshot`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: options,
                timeout: 30000,
            });
            
            if (result.success && result.data) {
                this.updateResponseTimeMetrics(Date.now() - startTime);
                return result.data as Buffer;
            }
            
            this.metrics.failedRequests++;
            throw new Error(result.error || "Screenshot failed");
        } catch (error) {
            this.metrics.failedRequests++;
            logger.error("[PuppeteerResource] Screenshot error", error);
            throw error;
        }
    }
    
    /**
     * Generate a PDF from a webpage or HTML
     */
    async generatePDF(options: PDFOptions): Promise<Buffer> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Puppeteer service is not available");
        }
        
        if (!this.canSpawnInstance()) {
            throw new Error(`Instance limit reached (${this.activeInstances}/${this.maxInstances})`);
        }
        
        this.metrics.totalRequests++;
        const startTime = Date.now();
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/pdf`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: options,
                timeout: 30000,
            });
            
            if (result.success && result.data) {
                this.updateResponseTimeMetrics(Date.now() - startTime);
                return result.data as Buffer;
            }
            
            this.metrics.failedRequests++;
            throw new Error(result.error || "PDF generation failed");
        } catch (error) {
            this.metrics.failedRequests++;
            logger.error("[PuppeteerResource] PDF generation error", error);
            throw error;
        }
    }
    
    /**
     * Scrape data from a webpage
     */
    async scrape(options: ScrapeOptions): Promise<any> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Puppeteer service is not available");
        }
        
        if (!this.canSpawnInstance()) {
            throw new Error(`Instance limit reached (${this.activeInstances}/${this.maxInstances})`);
        }
        
        this.metrics.totalRequests++;
        const startTime = Date.now();
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/scrape`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: options,
                timeout: 30000,
            });
            
            if (result.success && result.data) {
                this.updateResponseTimeMetrics(Date.now() - startTime);
                return result.data;
            }
            
            this.metrics.failedRequests++;
            throw new Error(result.error || "Scraping failed");
        } catch (error) {
            this.metrics.failedRequests++;
            logger.error("[PuppeteerResource] Scraping error", error);
            throw error;
        }
    }
    
    /**
     * Execute custom automation steps
     */
    async execute(options: ExecuteOptions): Promise<any> {
        if (this._status !== DiscoveryStatus.Available) {
            throw new Error("Puppeteer service is not available");
        }
        
        if (!this.canSpawnInstance()) {
            throw new Error(`Instance limit reached (${this.activeInstances}/${this.maxInstances})`);
        }
        
        this.metrics.totalRequests++;
        const startTime = Date.now();
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/execute`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: options,
                timeout: 60000, // Longer timeout for complex automations
            });
            
            if (result.success && result.data) {
                this.updateResponseTimeMetrics(Date.now() - startTime);
                return result.data;
            }
            
            this.metrics.failedRequests++;
            throw new Error(result.error || "Automation execution failed");
        } catch (error) {
            this.metrics.failedRequests++;
            logger.error("[PuppeteerResource] Automation execution error", error);
            throw error;
        }
    }
    
    /**
     * Get supported browser capabilities
     */
    getCapabilities(): string[] {
        return this.serviceInfo?.capabilities || [
            "screenshot",
            "pdf", 
            "scrape",
            "evaluate",
            "click",
            "type",
            "waitForSelector",
            "navigation",
        ];
    }
    
    /**
     * Update response time metrics
     */
    private updateResponseTimeMetrics(responseTime: number): void {
        this.metrics.lastResponseTimes.push(responseTime);
        
        // Keep only last 100 response times
        if (this.metrics.lastResponseTimes.length > 100) {
            this.metrics.lastResponseTimes.shift();
        }
        
        // Calculate average
        const sum = this.metrics.lastResponseTimes.reduce((a, b) => a + b, 0);
        this.metrics.averageResponseTime = Math.round(sum / this.metrics.lastResponseTimes.length);
    }
    
    /**
     * Provide enhanced metadata about the resource
     */
    protected getMetadata(): AgentMetadata {
        return {
            version: this.serviceVersion || "1.0.0",
            capabilities: this.getCapabilities(),
            activeInstances: this.activeInstances,
            maxInstances: this.maxInstances,
            supportedBrowsers: ["chromium"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
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
        logger.info("[PuppeteerResource] Shutting down Puppeteer resource");
        
        // Clear metrics
        this.metrics = {
            totalRequests: 0,
            failedRequests: 0,
            averageResponseTime: 0,
            lastResponseTimes: [],
        };
        
        // Reset instance tracking
        this.activeInstances = 0;
        this.totalInstancesCreated = 0;
        this.lastInstanceCheck = Date.now();
        
        // Clear service info
        this.serviceInfo = undefined;
        this.serviceVersion = undefined;
        
        // Call parent shutdown
        await super.shutdown();
        
        logger.info("[PuppeteerResource] Puppeteer resource shutdown complete");
    }
}
