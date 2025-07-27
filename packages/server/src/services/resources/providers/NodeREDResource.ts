/**
 * Node-RED Workflow Automation Resource Implementation
 * 
 * Provides infrastructure-level management for Node-RED automation platform including:
 * - Service discovery and health monitoring
 * - Flow management and monitoring
 * - Runtime statistics and performance metrics
 * - Authentication status checking
 * - Integration with unified configuration system
 */

import { MINUTES_5_MS } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { AutomationMetadata, NodeREDConfig } from "../typeRegistry.js";
import type { HealthCheckResult, IAutomationResource } from "../types.js";
import { DeploymentType, DiscoveryStatus, ResourceCategory } from "../types.js";

/**
 * Node-RED flow information
 */
interface NodeREDFlow {
    id: string;
    label?: string;
    type: "tab" | "subflow" | "config";
    disabled?: boolean;
    info?: string;
    env?: Array<{ name: string; value: any; type: string }>;
}

/**
 * Node-RED node information
 */
interface NodeREDNode {
    id: string;
    type: string;
    name?: string;
    z?: string; // flow id
    wires?: string[][];
}

/**
 * Node-RED runtime settings
 */
interface NodeREDSettings {
    httpNodeRoot: string;
    version: string;
    user?: {
        username: string;
        permissions: string;
    };
    context?: {
        default: string;
        stores: string[];
    };
    nodes?: {
        count: number;
        missing: string[];
    };
}

/**
 * Production Node-RED resource implementation with enhanced monitoring
 */
@RegisterResource
export class NodeREDResource extends ResourceProvider<"node-red", NodeREDConfig> implements IAutomationResource {
    // Resource identification
    readonly id = "node-red";
    readonly category = ResourceCategory.Automation;
    readonly displayName = "Node-RED";
    readonly description = "Visual programming tool for wiring IoT devices, APIs and online services";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Flow and node caching
    private flows: Map<string, NodeREDFlow> = new Map();
    private nodes: Map<string, NodeREDNode> = new Map();
    private lastFlowRefresh = 0;
    private settings?: NodeREDSettings;
    private readonly CACHE_TTL_MS = MINUTES_5_MS;

    /**
     * Perform service discovery by checking Node-RED availability
     */
    protected async performDiscovery(): Promise<boolean> {
        if (!this.config?.baseUrl) {
            logger.debug(`[${this.id}] No baseUrl configured`);
            return false;
        }

        try {
            // Try to reach the admin API
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/settings`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                auth: this.getAuthConfig(),
            });

            return result.success;
        } catch (error) {
            logger.debug(`[${this.id}] Discovery failed:`, error);
            return false;
        }
    }

    /**
     * Perform comprehensive health check
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        if (!this.config?.baseUrl) {
            return {
                healthy: false,
                message: "No baseUrl configured",
                timestamp: new Date(),
            };
        }

        try {
            // Check settings endpoint (basic health check)
            const settingsResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/settings`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                auth: this.getAuthConfig(),
            });

            if (!settingsResult.success) {
                return {
                    healthy: false,
                    message: "Settings API not accessible",
                    timestamp: new Date(),
                    details: { error: settingsResult.error },
                };
            }

            // Parse settings for additional health information
            let healthDetails: any = {
                baseUrl: this.config.baseUrl,
                adminUrl: `${this.config.baseUrl}/admin`,
            };

            if (settingsResult.data) {
                this.settings = settingsResult.data as NodeREDSettings;
                healthDetails = {
                    ...healthDetails,
                    version: this.settings.version,
                    httpNodeRoot: this.settings.httpNodeRoot,
                    userAuthenticated: !!this.settings.user,
                };
            }

            // Try to get flow count (optional)
            try {
                await this.refreshFlows();
                healthDetails.flowCount = this.flows.size;
                healthDetails.nodeCount = this.nodes.size;
            } catch (error) {
                logger.debug(`[${this.id}] Could not fetch flow info:`, error);
                // Continue with basic health check
            }

            return {
                healthy: true,
                message: `Node-RED is healthy (v${this.settings?.version || "unknown"})`,
                timestamp: new Date(),
                details: healthDetails,
            };

        } catch (error) {
            logger.error(`[${this.id}] Health check failed:`, error);
            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
            };
        }
    }

    /**
     * Get resource-specific metadata
     */
    protected getMetadata(): AutomationMetadata {
        return {
            version: this.settings?.version || "unknown",
            capabilities: ["visual-workflow", "code-editor", "debugging", "templates", "custom-nodes"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            workflowCount: this.flows.size,
            activeWorkflowCount: Array.from(this.flows.values()).filter(f => !f.disabled).length,
            executionCount: undefined, // Would need additional API calls to get this
            supportedTriggers: ["inject", "http", "mqtt", "websocket", "cron", "file", "tcp"],
            supportedActions: ["http", "mqtt", "email", "file", "function", "template", "debug"],
        };
    }

    /**
     * Get capabilities of this automation service
     */
    getCapabilities(): string[] {
        return ["visual-workflow", "code-editor", "debugging", "templates", "custom-nodes"];
    }

    /**
     * Check if service can execute a specific routine
     */
    async canExecuteRoutine(_routineId: string): Promise<boolean> {
        // Node-RED can potentially execute any routine through custom flows
        // This would require more specific analysis of the routine requirements
        return this._status === DiscoveryStatus.Available;
    }

    /**
     * List available workflows/flows
     */
    async listWorkflows(): Promise<string[]> {
        await this.refreshFlows();
        return Array.from(this.flows.values())
            .filter(f => f.type === "tab")
            .map(f => f.label || f.id);
    }

    /**
     * Get count of active workflows
     */
    async getActiveWorkflowCount(): Promise<number> {
        await this.refreshFlows();
        return Array.from(this.flows.values())
            .filter(f => f.type === "tab" && !f.disabled).length;
    }

    /**
     * Refresh flows and nodes information
     */
    private async refreshFlows(): Promise<void> {
        const now = Date.now();
        if (now - this.lastFlowRefresh < this.CACHE_TTL_MS) {
            return; // Use cached data
        }

        if (!this.config?.baseUrl) {
            return;
        }

        try {
            // Get flows
            const flowsResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/flows`,
                method: "GET",
                timeout: 5000,
                auth: this.getAuthConfig(),
            });

            if (flowsResult.success && Array.isArray(flowsResult.data)) {
                this.flows.clear();
                this.nodes.clear();

                for (const item of flowsResult.data) {
                    if (item.type === "tab" || item.type === "subflow") {
                        // This is a flow
                        this.flows.set(item.id, item as NodeREDFlow);
                    } else {
                        // This is a node
                        this.nodes.set(item.id, item as NodeREDNode);
                    }
                }

                this.lastFlowRefresh = now;
                logger.debug(`[${this.id}] Refreshed ${this.flows.size} flows and ${this.nodes.size} nodes`);
            }
        } catch (error) {
            logger.debug(`[${this.id}] Failed to refresh flows:`, error);
            // Don't throw - this is optional functionality
        }
    }

    /**
     * Get authentication configuration for Node-RED API
     */
    protected getAuthConfig() {
        if (!this.config) return undefined;

        // Check for admin authentication
        if (this.config.adminAuth?.username && this.config.adminAuth?.password) {
            return {
                type: "basic" as const,
                username: this.config.adminAuth.username,
                password: this.config.adminAuth.password,
            };
        }

        return super.getAuthConfig();
    }
}
