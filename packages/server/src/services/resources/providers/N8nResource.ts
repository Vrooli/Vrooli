/**
 * n8n Workflow Automation Resource Implementation
 * 
 * Provides infrastructure-level management for n8n automation platform including:
 * - Service discovery and health monitoring
 * - Workflow statistics and monitoring
 * - Webhook URL configuration and validation
 * - Authentication status checking
 * - Integration with unified configuration system
 */

import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import { ResourceCategory, DeploymentType, ResourceHealth, DiscoveryStatus, ResourceEvent } from "../types.js";
import type { HealthCheckResult, IAutomationResource, ResourceEventData } from "../types.js";
import type { N8nConfig, AutomationMetadata } from "../typeRegistry.js";
import { logger } from "../../../events/logger.js";

/**
 * Workflow information from n8n API
 */
interface N8nWorkflow {
    id: string;
    name: string;
    active: boolean;
    createdAt: string;
    updatedAt: string;
    tags?: Array<{ id: string; name: string }>;
    nodes?: Array<{ type: string; name: string }>;
}

/**
 * n8n version and instance information
 */
interface N8nVersionInfo {
    version: string;
    versionCli: string;
    platform: string;
    nodeVersion: string;
}

/**
 * n8n instance settings
 */
interface N8nSettings {
    allowedModules?: string[];
    communityNodesEnabled?: boolean;
    defaultLocale?: string;
    executionMode?: string;
    instanceId?: string;
    n8n?: {
        version: string;
        versionCli: string;
    };
    templates?: {
        enabled: boolean;
        host: string;
    };
    telemetry?: {
        enabled: boolean;
    };
}

/**
 * Production n8n resource implementation with enhanced monitoring and workflow management
 */
@RegisterResource
export class N8nResource extends ResourceProvider<"n8n", N8nConfig> implements IAutomationResource {
    // Resource identification
    readonly id = "n8n";
    readonly category = ResourceCategory.Automation;
    readonly displayName = "n8n";
    readonly description = "Workflow automation platform with extensive integrations";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;
    
    // Enhanced workflow caching with metadata
    private workflows: Map<string, N8nWorkflow> = new Map();
    private lastWorkflowRefresh = 0;
    private instanceInfo?: N8nVersionInfo;
    private instanceSettings?: N8nSettings;
    private readonly WORKFLOW_CACHE_TTL_MS = 2 * 60 * 1000; // 2 minutes
    
    // Authentication state
    private authEnabled = false;
    private authChecked = false;
    
    /**
     * Enhanced discovery with n8n-specific checks
     */
    protected async performDiscovery(): Promise<boolean> {
        try {
            logger.debug("[N8nResource] Starting service discovery");
            
            // First check basic health endpoint
            const healthResult = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/healthz`,
                method: "GET",
                timeout: 5000,
            });
            
            if (!healthResult.success) {
                logger.debug(`[N8nResource] Health endpoint failed: ${healthResult.error}`);
                return false;
            }

            // Try to get version information (may require auth)
            try {
                const versionResult = await this.httpClient!.makeRequest({
                    url: `${this.config!.baseUrl}/rest/version`,
                    method: "GET",
                    timeout: 3000,
                });
                
                if (versionResult.success && versionResult.data) {
                    this.instanceInfo = versionResult.data;
                    logger.debug("[N8nResource] Retrieved version information", this.instanceInfo);
                }
            } catch (error) {
                logger.debug("[N8nResource] Version endpoint not accessible (may require auth)");
            }

            // Check authentication requirement
            await this.checkAuthenticationStatus();
            
            // Try to get workflow count if accessible
            await this.refreshWorkflowCache();
            
            logger.info("[N8nResource] Discovered n8n service", {
                version: this.instanceInfo?.version,
                baseUrl: this.config!.baseUrl,
                authRequired: this.authEnabled,
                workflowCount: this.workflows.size,
            });

            // Emit discovery event
            this.emitResourceEvent(ResourceEvent.Discovered, {
                version: this.instanceInfo?.version,
                authEnabled: this.authEnabled,
                workflowCount: this.workflows.size,
            });
            
            return true;
        } catch (error) {
            logger.debug(`[N8nResource] Discovery failed: ${error}`);
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
                url: `${this.config!.baseUrl}/healthz`,
                method: "GET",
                timeout: this.config!.healthCheck?.timeoutMs || 5000,
            });
            
            const responseTime = Date.now() - startTime;
            
            if (result.success) {
                // Refresh workflow cache if stale
                if (Date.now() - this.lastWorkflowRefresh > this.WORKFLOW_CACHE_TTL_MS) {
                    await this.refreshWorkflowCache();
                }
                
                const activeWorkflows = Array.from(this.workflows.values()).filter(w => w.active).length;
                const details = {
                    workflowCount: this.workflows.size,
                    activeWorkflows,
                    authEnabled: this.authEnabled,
                    version: this.instanceInfo?.version,
                    responseTime,
                    baseUrl: this.config!.baseUrl,
                    webhookUrl: this.config!.webhookUrl || `${this.config!.baseUrl}/webhook`,
                    lastWorkflowRefresh: new Date(this.lastWorkflowRefresh),
                };

                logger.debug("[N8nResource] Health check passed", details);

                return {
                    healthy: true,
                    message: `n8n is healthy with ${this.workflows.size} workflows (${activeWorkflows} active, ${responseTime}ms response)`,
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

            logger.warn("[N8nResource] Health check failed", errorDetails);
            
            return {
                healthy: false,
                message: `n8n health check failed: ${result.error}`,
                details: errorDetails,
                timestamp: new Date(),
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : "Unknown health check error";
            
            logger.error(`[N8nResource] Health check error: ${errorMessage}`, error);
            
            return {
                healthy: false,
                message: errorMessage,
                details: { error: errorMessage },
                timestamp: new Date(),
            };
        }
    }
    
    /**
     * Check if authentication is required
     */
    private async checkAuthenticationStatus(): Promise<void> {
        if (this.authChecked) return;
        
        try {
            // Try to access a protected endpoint
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/rest/workflows`,
                method: "GET",
                timeout: 3000,
            });
            
            // If we get 401/403, auth is required
            this.authEnabled = result.status === 401 || result.status === 403;
            this.authChecked = true;
            
            logger.debug(`[N8nResource] Authentication ${this.authEnabled ? "required" : "not required"}`);
        } catch (error) {
            logger.debug("[N8nResource] Could not determine authentication status", error);
            this.authEnabled = false;
        }
    }
    
    /**
     * Refresh workflow cache
     */
    private async refreshWorkflowCache(): Promise<void> {
        // Skip if auth is required and no credentials provided
        if (this.authEnabled && !this.getAuthConfig()) {
            logger.debug("[N8nResource] Skipping workflow refresh - authentication required");
            return;
        }
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/rest/workflows`,
                method: "GET",
                timeout: 10000,
                auth: this.getAuthConfig(),
            });
            
            if (result.success && result.data?.data) {
                this.workflows.clear();
                
                for (const workflow of result.data.data) {
                    this.workflows.set(workflow.id, workflow);
                }
                
                this.lastWorkflowRefresh = Date.now();
                
                logger.debug(`[N8nResource] Refreshed workflow cache with ${this.workflows.size} workflows`);
            }
        } catch (error) {
            logger.debug("[N8nResource] Failed to refresh workflow cache", error);
        }
    }
    
    /**
     * List workflows - required by IAutomationResource
     */
    async listWorkflows(): Promise<string[]> {
        if (this._status !== DiscoveryStatus.Available) {
            logger.debug("[N8nResource] Service not available, returning empty workflow list");
            return [];
        }
        
        // Refresh cache if stale
        if (Date.now() - this.lastWorkflowRefresh > this.WORKFLOW_CACHE_TTL_MS) {
            await this.refreshWorkflowCache();
        }
        
        return Array.from(this.workflows.values()).map(w => w.name);
    }
    
    /**
     * Get active workflow count - required by IAutomationResource
     */
    async getActiveWorkflowCount(): Promise<number> {
        await this.listWorkflows(); // Ensure cache is fresh
        return Array.from(this.workflows.values()).filter(w => w.active).length;
    }
    
    /**
     * Get capabilities of this automation service - required by IAutomationResource
     */
    getCapabilities(): string[] {
        return [
            "workflow-automation",
            "webhooks",
            "scheduled-workflows",
            "manual-triggers",
            "error-workflows",
            "sub-workflows",
            "credential-management",
            "rest-api",
            "400+ integrations",
            "custom-code-nodes",
            "conditional-logic",
            "loops-and-iterations",
            "data-transformation",
            "file-processing",
            "http-requests",
        ];
    }
    
    /**
     * Check if service can execute a specific routine - required by IAutomationResource
     */
    async canExecuteRoutine(routineId: string): Promise<boolean> {
        // For n8n, we check if the workflow exists and is active
        await this.listWorkflows(); // Ensure cache is fresh
        
        const workflow = Array.from(this.workflows.values()).find(
            w => w.id === routineId || w.name === routineId,
        );
        
        return workflow?.active || false;
    }
    
    /**
     * Get workflow statistics
     */
    async getWorkflowStats(): Promise<{
        total: number;
        active: number;
        inactive: number;
        byTag: Record<string, number>;
        byNodeType: Record<string, number>;
    }> {
        await this.listWorkflows(); // Ensure cache is fresh
        
        const workflows = Array.from(this.workflows.values());
        const active = workflows.filter(w => w.active).length;
        
        // Count by tags
        const byTag: Record<string, number> = {};
        const byNodeType: Record<string, number> = {};
        
        for (const workflow of workflows) {
            // Count tags
            if (workflow.tags) {
                for (const tag of workflow.tags) {
                    byTag[tag.name] = (byTag[tag.name] || 0) + 1;
                }
            }
            
            // Count node types
            if (workflow.nodes) {
                for (const node of workflow.nodes) {
                    byNodeType[node.type] = (byNodeType[node.type] || 0) + 1;
                }
            }
        }
        
        return {
            total: workflows.length,
            active,
            inactive: workflows.length - active,
            byTag,
            byNodeType,
        };
    }
    
    /**
     * Execute a workflow (if API allows)
     */
    async executeWorkflow(workflowId: string, data?: any): Promise<{
        success: boolean;
        executionId?: string;
        error?: string;
    }> {
        if (this._status !== DiscoveryStatus.Available) {
            return { success: false, error: "n8n is not available" };
        }
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config!.baseUrl}/rest/workflows/${workflowId}/run`,
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: { data },
                timeout: 30000,
                auth: this.getAuthConfig(),
            });
            
            if (result.success && result.data?.data?.id) {
                return {
                    success: true,
                    executionId: result.data.data.id,
                };
            }
            
            return {
                success: false,
                error: result.error || "Failed to execute workflow",
            };
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error(`[N8nResource] Failed to execute workflow ${workflowId}`, error);
            return { success: false, error: errorMessage };
        }
    }
    
    /**
     * Test webhook endpoint
     */
    async testWebhook(path = "/webhook-test/test"): Promise<boolean> {
        const webhookUrl = this.config!.webhookUrl || `${this.config!.baseUrl}/webhook`;
        
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${webhookUrl}${path}`,
                method: "GET",
                timeout: 5000,
            });
            
            // n8n webhooks typically return 404 if the path doesn't exist
            // but the webhook system itself is working
            return result.status !== 500 && result.status !== 502 && result.status !== 503;
        } catch (error) {
            logger.debug(`[N8nResource] Webhook test failed for ${path}`, error);
            return false;
        }
    }
    
    /**
     * Provide enhanced metadata about the resource
     */
    protected getMetadata(): AutomationMetadata {
        const workflowList = Array.from(this.workflows.values());
        const activeCount = workflowList.filter(w => w.active).length;
        
        return {
            version: this.instanceInfo?.version || "unknown",
            capabilities: [
                "workflow-automation",
                "webhooks",
                "rest-api",
                "scheduled-workflows",
                "manual-triggers",
                "error-workflows",
                "sub-workflows",
                "credential-management",
            ],
            workflowCount: this.workflows.size,
            activeWorkflowCount: activeCount,
            supportedTriggers: [
                "webhook",
                "schedule",
                "manual",
                "error",
                "workflow",
            ],
            integrationCount: 400, // n8n has 400+ integrations
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            // n8n-specific metadata
            nodeVersion: this.instanceInfo?.nodeVersion,
            platform: this.instanceInfo?.platform,
            executionMode: this.instanceSettings?.executionMode || "regular",
            communityNodesEnabled: this.instanceSettings?.communityNodesEnabled || false,
            templatesEnabled: this.instanceSettings?.templates?.enabled || false,
            authRequired: this.authEnabled,
            webhookUrl: this.config?.webhookUrl || `${this.config?.baseUrl}/webhook`,
        };
    }
    
    /**
     * Get authentication configuration from resource config
     * n8n supports basic auth primarily
     */
    protected getAuthConfig() {
        if (!this.config || !this.authEnabled) return undefined;
        
        // Check for basic auth credentials
        if (this.config.username && this.config.password) {
            return {
                type: "basic" as const,
                username: this.config.username,
                password: this.config.password,
            };
        }
        
        // Check for API key (n8n supports this in newer versions)
        if (this.config.apiKey) {
            return {
                type: "apikey" as const,
                token: this.config.apiKey,
                headerName: "X-N8N-API-KEY",
            };
        }
        
        return undefined;
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
        logger.info("[N8nResource] Shutting down n8n resource");
        
        // Clear caches
        this.workflows.clear();
        this.lastWorkflowRefresh = 0;
        this.instanceInfo = undefined;
        this.instanceSettings = undefined;
        this.authEnabled = false;
        this.authChecked = false;
        
        // Call parent shutdown
        await super.shutdown();
        
        logger.info("[N8nResource] n8n resource shutdown complete");
    }
}
