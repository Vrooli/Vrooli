/**
 * Huginn Automation Resource Implementation
 * 
 * Provides infrastructure-level management for Huginn automation platform including:
 * - Service discovery and health monitoring
 * - Agent management and monitoring
 * - Scenario execution statistics
 * - Authentication status checking
 * - Integration with unified configuration system
 */

import { MINUTES_5_MS } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { AutomationMetadata, HuginnConfig } from "../typeRegistry.js";
import type { HealthCheckResult, IAutomationResource } from "../types.js";
import { DeploymentType, DiscoveryStatus, ResourceCategory } from "../types.js";

/**
 * Huginn agent information
 */
interface HuginnAgent {
    id: number;
    name: string;
    type: string;
    schedule?: string;
    disabled: boolean;
    working: boolean;
    last_check_at?: string;
    last_event_at?: string;
    last_error_log_at?: string;
    events_count: number;
}

/**
 * Huginn scenario information
 */
interface HuginnScenario {
    id: number;
    name: string;
    description?: string;
    public: boolean;
    source_url?: string;
    agent_ids: number[];
    created_at: string;
    updated_at: string;
}

/**
 * Huginn system statistics
 */
interface HuginnStats {
    agents_count: number;
    scenarios_count: number;
    events_count: number;
    active_agents_count: number;
    disabled_agents_count: number;
}

/**
 * Production Huginn resource implementation with enhanced monitoring
 */
@RegisterResource
export class HuginnResource extends ResourceProvider<"huginn", HuginnConfig> implements IAutomationResource {
    // Resource identification
    readonly id = "huginn";
    readonly category = ResourceCategory.Automation;
    readonly displayName = "Huginn";
    readonly description = "Multi-tenant web scraping and automation platform";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Agent and scenario caching
    private agents: Map<number, HuginnAgent> = new Map();
    private scenarios: Map<number, HuginnScenario> = new Map();
    private lastStatsRefresh = 0;
    private stats?: HuginnStats;
    private readonly CACHE_TTL_MS = MINUTES_5_MS;

    /**
     * Perform service discovery by checking Huginn availability
     */
    protected async performDiscovery(): Promise<boolean> {
        if (!this.config?.baseUrl) {
            logger.debug(`[${this.id}] No baseUrl configured`);
            return false;
        }

        try {
            // Try to reach the login page or API (depending on configuration)
            const testUrl = this.config.username && this.config.password
                ? `${this.config.baseUrl}/api/v1/agents`
                : `${this.config.baseUrl}/users/sign_in`;

            const result = await this.httpClient!.makeRequest({
                url: testUrl,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                auth: this.getAuthConfig(),
            });

            // For API endpoint, expect success; for login page, expect HTML response
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

        if (!this.config.username || !this.config.password) {
            return {
                healthy: false,
                message: "Username and password required for Huginn access",
                timestamp: new Date(),
            };
        }

        try {
            // Check API endpoint with authentication
            const agentsResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/api/v1/agents`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                auth: this.getAuthConfig(),
            });

            if (!agentsResult.success) {
                // If API fails, try checking if the web interface is at least accessible
                const webResult = await this.httpClient!.makeRequest({
                    url: `${this.config.baseUrl}/users/sign_in`,
                    method: "GET",
                    timeout: this.config.healthCheck?.timeoutMs || 5000,
                });

                if (webResult.success) {
                    return {
                        healthy: false,
                        message: "Huginn web interface accessible but API authentication failed",
                        timestamp: new Date(),
                        details: {
                            webAccessible: true,
                            apiError: agentsResult.error,
                            suggestion: "Check username/password configuration",
                        },
                    };
                }

                return {
                    healthy: false,
                    message: "Huginn service not accessible",
                    timestamp: new Date(),
                    details: { error: agentsResult.error },
                };
            }

            // Parse agent data and refresh stats
            let healthDetails: any = {
                baseUrl: this.config.baseUrl,
                authenticated: true,
            };

            try {
                await this.refreshStats();
                if (this.stats) {
                    healthDetails = {
                        ...healthDetails,
                        agentsCount: this.stats.agents_count,
                        scenariosCount: this.stats.scenarios_count,
                        activeAgents: this.stats.active_agents_count,
                        eventsCount: this.stats.events_count,
                    };
                }
            } catch (error) {
                logger.debug(`[${this.id}] Could not fetch detailed stats:`, error);
                // Continue with basic health check
            }

            return {
                healthy: true,
                message: `Huginn is healthy (${this.stats?.agents_count || 0} agents, ${this.stats?.scenarios_count || 0} scenarios)`,
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
            version: "unknown", // Huginn doesn't easily expose version info
            capabilities: ["web-scraping", "scheduling", "workflows", "multi-tenant", "api-access"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            workflowCount: this.stats?.scenarios_count || 0,
            activeWorkflowCount: this.stats?.active_agents_count || 0,
            executionCount: this.stats?.events_count || 0,
            supportedTriggers: ["webhook", "rss", "email", "twitter", "schedule", "manual"],
            supportedActions: ["webhook", "email", "twitter", "slack", "file", "data-transform"],
        };
    }

    /**
     * Get capabilities of this automation service
     */
    getCapabilities(): string[] {
        return ["web-scraping", "scheduling", "workflows", "multi-tenant", "api-access"];
    }

    /**
     * Check if service can execute a specific routine
     */
    async canExecuteRoutine(_routineId: string): Promise<boolean> {
        // Huginn can potentially execute routines through custom agents and scenarios
        // This would require more specific analysis of the routine requirements
        return this._status === DiscoveryStatus.Available;
    }

    /**
     * List available workflows/scenarios
     */
    async listWorkflows(): Promise<string[]> {
        await this.refreshStats();
        return Array.from(this.scenarios.values()).map(s => s.name);
    }

    /**
     * Get count of active workflows (agents in Huginn context)
     */
    async getActiveWorkflowCount(): Promise<number> {
        await this.refreshStats();
        return this.stats?.active_agents_count || 0;
    }

    /**
     * Refresh statistics and agent information
     */
    private async refreshStats(): Promise<void> {
        const now = Date.now();
        if (now - this.lastStatsRefresh < this.CACHE_TTL_MS) {
            return; // Use cached data
        }

        if (!this.config?.baseUrl || !this.getAuthConfig()) {
            return;
        }

        try {
            // Get agents
            const agentsResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/api/v1/agents`,
                method: "GET",
                timeout: 5000,
                auth: this.getAuthConfig(),
            });

            let agentsCount = 0;
            let activeAgentsCount = 0;

            if (agentsResult.success && Array.isArray(agentsResult.data)) {
                this.agents.clear();
                for (const agent of agentsResult.data) {
                    this.agents.set(agent.id, agent as HuginnAgent);
                    agentsCount++;
                    if (!agent.disabled && agent.working) {
                        activeAgentsCount++;
                    }
                }
            }

            // Get scenarios
            const scenariosResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/api/v1/scenarios`,
                method: "GET",
                timeout: 5000,
                auth: this.getAuthConfig(),
            });

            let scenariosCount = 0;
            if (scenariosResult.success && Array.isArray(scenariosResult.data)) {
                this.scenarios.clear();
                for (const scenario of scenariosResult.data) {
                    this.scenarios.set(scenario.id, scenario as HuginnScenario);
                    scenariosCount++;
                }
            }

            // Calculate total events from agents
            const eventsCount = Array.from(this.agents.values())
                .reduce((total, agent) => total + (agent.events_count || 0), 0);

            this.stats = {
                agents_count: agentsCount,
                scenarios_count: scenariosCount,
                events_count: eventsCount,
                active_agents_count: activeAgentsCount,
                disabled_agents_count: agentsCount - activeAgentsCount,
            };

            this.lastStatsRefresh = now;
            logger.debug(`[${this.id}] Refreshed stats: ${agentsCount} agents, ${scenariosCount} scenarios`);

        } catch (error) {
            logger.debug(`[${this.id}] Failed to refresh stats:`, error);
            // Don't throw - this is optional functionality
        }
    }

    /**
     * Get authentication configuration for Huginn API
     */
    protected getAuthConfig() {
        if (!this.config?.username || !this.config?.password) {
            return undefined;
        }

        return {
            type: "basic" as const,
            username: this.config.username,
            password: this.config.password,
        };
    }
}
