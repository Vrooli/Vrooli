import { type Logger } from "winston";
import { getIntelligentEventBus, type IntelligentEvent, type AgentResponse, type AgentSubscription } from "../events/eventBus.js";
import { EmergentAgent, type AgentInsights } from "./emergentAgent.js";

/**
 * Agent Deployment Service - Infrastructure for deploying goal-driven emergent agents
 * 
 * This service provides the infrastructure for teams to deploy agents with specific goals
 * and routines. Agents learn from events and propose routine improvements, enabling
 * emergent capabilities to develop naturally.
 */
export class AgentDeploymentService {
    private readonly logger: Logger;
    private readonly eventBus: any;
    private deployedAgents = new Map<string, DeployedEmergentAgent>();
    private agentSwarms = new Map<string, EmergentSwarm>();
    
    // Service configuration
    private readonly HEALTH_CHECK_INTERVAL = 30000; // 30 seconds
    private readonly SWARM_COORDINATION_INTERVAL = 60000; // 1 minute
    
    constructor(logger?: Logger) {
        this.logger = logger || console as any;
        this.eventBus = getIntelligentEventBus();
        
        // Start health monitoring
        this.startHealthMonitoring();
    }

    /**
     * Deploy an emergent agent with a specific goal and initial routine
     */
    async deployAgent(config: EmergentAgentConfig): Promise<string> {
        try {
            this.validateAgentConfig(config);
            
            // Create emergent agent instance
            const agent = new EmergentAgent(
                config.agentId,
                config.goal,
                config.initialRoutine,
                this.logger,
            );
            
            // Create subscription based on goal and subscriptions
            const subscription = this.createGoalBasedSubscription(config, agent);
            
            // Register with event bus
            await this.eventBus.subscribeAgent(subscription);
            
            // Store deployment
            const deployment: DeployedEmergentAgent = {
                config,
                agent,
                subscription,
                status: "active",
                deployedAt: new Date(),
                lastHealthCheck: new Date(),
                swarmId: config.swarmId,
            };
            
            this.deployedAgents.set(config.agentId, deployment);
            
            // Add to swarm if specified
            if (config.swarmId) {
                await this.addAgentToSwarm(config.agentId, config.swarmId);
            }
            
            this.logger.info("[AgentDeploymentService] Emergent agent deployed", {
                agentId: config.agentId,
                goal: config.goal,
                initialRoutine: config.initialRoutine,
                subscriptions: config.subscriptions,
            });
            
            return config.agentId;
            
        } catch (error) {
            this.logger.error("[AgentDeploymentService] Failed to deploy agent", {
                agentId: config.agentId,
                goal: config.goal,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Deploy a swarm of coordinated emergent agents
     */
    async deploySwarm(config: EmergentSwarmConfig): Promise<string> {
        try {
            const swarmId = config.swarmId || `swarm_${Date.now()}`;
            
            // Create swarm
            const swarm: EmergentSwarm = {
                id: swarmId,
                name: config.name,
                description: config.description,
                agents: new Map(),
                coordination: config.coordination || {
                    sharedLearning: true,
                    collaborativeProposals: false,
                    crossAgentInsights: true,
                },
                status: "deploying",
                deployedAt: new Date(),
                emergentCapabilities: [],
            };
            
            this.agentSwarms.set(swarmId, swarm);
            
            // Deploy individual agents
            const deploymentPromises: Promise<string>[] = [];
            
            for (const agentConfig of config.agents) {
                const fullConfig: EmergentAgentConfig = {
                    ...agentConfig,
                    agentId: agentConfig.agentId || `${swarmId}_${agentConfig.goal.replace(/\s+/g, "_").toLowerCase()}_${Date.now()}`,
                    swarmId,
                };
                
                deploymentPromises.push(this.deployAgent(fullConfig));
            }
            
            const deployedAgentIds = await Promise.all(deploymentPromises);
            
            // Update swarm status
            swarm.status = "active";
            
            this.logger.info("[AgentDeploymentService] Emergent swarm deployed", {
                swarmId,
                agentCount: deployedAgentIds.length,
                coordination: swarm.coordination,
            });
            
            return swarmId;
            
        } catch (error) {
            this.logger.error("[AgentDeploymentService] Failed to deploy swarm", {
                swarmId: config.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get agent insights including learning data and proposals
     */
    async getAgentInsights(agentId: string): Promise<AgentInsights | null> {
        const deployment = this.deployedAgents.get(agentId);
        if (!deployment) {
            return null;
        }
        
        return deployment.agent.getInsights();
    }

    /**
     * Get swarm insights showing emergent capabilities and coordination
     */
    async getSwarmInsights(swarmId: string): Promise<SwarmInsights | null> {
        const swarm = this.agentSwarms.get(swarmId);
        if (!swarm) {
            return null;
        }
        
        // Collect insights from all agents in swarm
        const agentInsights: AgentInsights[] = [];
        for (const [agentId] of swarm.agents) {
            const insights = await this.getAgentInsights(agentId);
            if (insights) {
                agentInsights.push(insights);
            }
        }
        
        // Identify emergent capabilities from agent collaboration
        const emergentCapabilities = this.identifyEmergentCapabilities(agentInsights);
        
        return {
            swarmId,
            name: swarm.name,
            description: swarm.description,
            status: swarm.status,
            deployedAt: swarm.deployedAt,
            coordination: swarm.coordination,
            agentCount: swarm.agents.size,
            agentInsights,
            emergentCapabilities,
            swarmMetrics: this.calculateSwarmMetrics(agentInsights),
        };
    }

    /**
     * Accept a routine improvement proposal from an agent
     */
    async acceptProposal(agentId: string, proposalId: string): Promise<boolean> {
        const deployment = this.deployedAgents.get(agentId);
        if (!deployment) {
            throw new Error(`Agent ${agentId} not found`);
        }
        
        const success = await deployment.agent.acceptProposal(proposalId);
        
        if (success) {
            this.logger.info("[AgentDeploymentService] Proposal accepted", {
                agentId,
                proposalId,
            });
            
            // Update swarm emergent capabilities if agent is in a swarm
            if (deployment.swarmId) {
                await this.updateSwarmCapabilities(deployment.swarmId);
            }
        }
        
        return success;
    }

    /**
     * Reject a routine improvement proposal from an agent
     */
    async rejectProposal(agentId: string, proposalId: string, reason: string): Promise<boolean> {
        const deployment = this.deployedAgents.get(agentId);
        if (!deployment) {
            throw new Error(`Agent ${agentId} not found`);
        }
        
        return await deployment.agent.rejectProposal(proposalId, reason);
    }

    /**
     * Undeploy an agent
     */
    async undeployAgent(agentId: string): Promise<void> {
        const deployment = this.deployedAgents.get(agentId);
        if (!deployment) {
            throw new Error(`Agent ${agentId} not found`);
        }
        
        try {
            // Unsubscribe from event bus
            await this.eventBus.unsubscribeAgent(agentId);
            
            // Remove from swarm if applicable
            if (deployment.swarmId) {
                const swarm = this.agentSwarms.get(deployment.swarmId);
                if (swarm) {
                    swarm.agents.delete(agentId);
                    await this.updateSwarmCapabilities(deployment.swarmId);
                }
            }
            
            // Clean up
            this.deployedAgents.delete(agentId);
            
            this.logger.info("[AgentDeploymentService] Agent undeployed", { agentId });
            
        } catch (error) {
            this.logger.error("[AgentDeploymentService] Failed to undeploy agent", {
                agentId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Undeploy an entire swarm
     */
    async undeploySwarm(swarmId: string): Promise<void> {
        const swarm = this.agentSwarms.get(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        try {
            // Undeploy all agents in swarm
            const undeployPromises: Promise<void>[] = [];
            for (const [agentId] of swarm.agents) {
                undeployPromises.push(this.undeployAgent(agentId));
            }
            
            await Promise.all(undeployPromises);
            
            // Clean up swarm
            this.agentSwarms.delete(swarmId);
            
            this.logger.info("[AgentDeploymentService] Swarm undeployed", { swarmId });
            
        } catch (error) {
            this.logger.error("[AgentDeploymentService] Failed to undeploy swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * List all deployed agents and swarms
     */
    getDeploymentStatus(): DeploymentStatus {
        const agents: AgentDeploymentInfo[] = [];
        const swarms: SwarmDeploymentInfo[] = [];
        
        // Collect agent status
        for (const [agentId, deployment] of this.deployedAgents) {
            agents.push({
                agentId,
                goal: deployment.config.goal,
                initialRoutine: deployment.config.initialRoutine,
                status: deployment.status,
                swarmId: deployment.swarmId,
                deployedAt: deployment.deployedAt,
            });
        }
        
        // Collect swarm status
        for (const [swarmId, swarm] of this.agentSwarms) {
            swarms.push({
                swarmId,
                name: swarm.name,
                description: swarm.description,
                status: swarm.status,
                agentCount: swarm.agents.size,
                deployedAt: swarm.deployedAt,
                emergentCapabilities: swarm.emergentCapabilities.length,
            });
        }
        
        return {
            agents,
            swarms,
            summary: {
                totalAgents: this.deployedAgents.size,
                totalSwarms: this.agentSwarms.size,
                activeAgents: agents.filter(a => a.status === "active").length,
                activeSwarms: swarms.filter(s => s.status === "active").length,
            },
        };
    }

    /**
     * Private helper methods
     */
    private validateAgentConfig(config: EmergentAgentConfig): void {
        if (!config.agentId || !config.goal || !config.initialRoutine) {
            throw new Error("Agent configuration missing required fields: agentId, goal, initialRoutine");
        }
        
        if (!config.subscriptions || config.subscriptions.length === 0) {
            throw new Error("Agent must have at least one event subscription");
        }
        
        if (this.deployedAgents.has(config.agentId)) {
            throw new Error(`Agent with id ${config.agentId} already deployed`);
        }
    }

    private createGoalBasedSubscription(config: EmergentAgentConfig, agent: EmergentAgent): AgentSubscription {
        return {
            agentId: config.agentId,
            eventPatterns: config.subscriptions,
            capabilities: [config.goal], // Goal becomes the primary capability
            priority: config.priority || 5,
            handler: async (event: IntelligentEvent) => {
                return await agent.processEvent(event);
            },
            learningEnabled: true, // Always enable learning for emergent agents
            filterPredicate: config.filterPredicate,
        };
    }

    private async addAgentToSwarm(agentId: string, swarmId: string): Promise<void> {
        const swarm = this.agentSwarms.get(swarmId);
        const deployment = this.deployedAgents.get(agentId);
        
        if (swarm && deployment) {
            swarm.agents.set(agentId, deployment);
            await this.updateSwarmCapabilities(swarmId);
        }
    }

    private identifyEmergentCapabilities(agentInsights: AgentInsights[]): EmergentCapability[] {
        const capabilities: EmergentCapability[] = [];
        
        // Analyze patterns across agents to identify emergent capabilities
        
        // 1. Cross-agent pattern recognition
        const commonPatterns = this.findCommonPatterns(agentInsights);
        if (commonPatterns.length > 0) {
            capabilities.push({
                name: "Cross-Agent Pattern Recognition",
                description: "Ability to recognize patterns across multiple agent domains",
                confidence: 0.8,
                contributingAgents: agentInsights.map(a => a.agentId),
                emergentFrom: commonPatterns,
            });
        }
        
        // 2. Collaborative optimization (if multiple agents have optimization proposals)
        const optimizationAgents = agentInsights.filter(a => 
            a.goal.toLowerCase().includes("optimize") || 
            a.activeProposals.some(p => p.improvementType === "optimization"),
        );
        
        if (optimizationAgents.length > 1) {
            capabilities.push({
                name: "Collaborative Optimization",
                description: "Coordinated optimization across multiple domains",
                confidence: 0.7,
                contributingAgents: optimizationAgents.map(a => a.agentId),
                emergentFrom: ["performance optimization", "cost reduction"],
            });
        }
        
        // 3. Quality-Security Integration
        const qualityAgents = agentInsights.filter(a => a.goal.toLowerCase().includes("quality"));
        const securityAgents = agentInsights.filter(a => a.goal.toLowerCase().includes("security"));
        
        if (qualityAgents.length > 0 && securityAgents.length > 0) {
            capabilities.push({
                name: "Security-Quality Integration",
                description: "Integrated approach to quality assurance and security compliance",
                confidence: 0.9,
                contributingAgents: [...qualityAgents.map(a => a.agentId), ...securityAgents.map(a => a.agentId)],
                emergentFrom: ["quality assurance", "security monitoring"],
            });
        }
        
        // 4. Domain expertise emergence (agents with many successful proposals)
        const expertAgents = agentInsights.filter(a => a.successfulProposals.length > 3);
        for (const agent of expertAgents) {
            const domain = this.extractDomainFromGoal(agent.goal);
            capabilities.push({
                name: `${domain} Expertise`,
                description: `Specialized expertise in ${domain} through iterative learning`,
                confidence: Math.min(0.95, 0.5 + (agent.successfulProposals.length * 0.1)),
                contributingAgents: [agent.agentId],
                emergentFrom: [agent.goal, ...agent.emergentCapabilities],
            });
        }
        
        return capabilities;
    }

    private findCommonPatterns(agentInsights: AgentInsights[]): string[] {
        const patternMap = new Map<string, number>();
        
        for (const agent of agentInsights) {
            for (const pattern of agent.patternsLearned) {
                const count = patternMap.get(pattern.pattern) || 0;
                patternMap.set(pattern.pattern, count + 1);
            }
        }
        
        // Return patterns seen by multiple agents
        return Array.from(patternMap.entries())
            .filter(([, count]) => count > 1)
            .map(([pattern]) => pattern);
    }

    private extractDomainFromGoal(goal: string): string {
        const goalLower = goal.toLowerCase();
        
        if (goalLower.includes("security")) return "Security";
        if (goalLower.includes("performance") || goalLower.includes("optimize")) return "Performance";
        if (goalLower.includes("quality")) return "Quality";
        if (goalLower.includes("cost")) return "Cost Management";
        if (goalLower.includes("monitor")) return "Monitoring";
        
        return "General Intelligence";
    }

    private calculateSwarmMetrics(agentInsights: AgentInsights[]): SwarmMetrics {
        const totalProposals = agentInsights.reduce((sum, a) => sum + a.activeProposals.length, 0);
        const totalAccepted = agentInsights.reduce((sum, a) => sum + a.successfulProposals.length, 0);
        const totalPatterns = agentInsights.reduce((sum, a) => sum + a.patternsLearned.length, 0);
        
        return {
            totalProposals,
            totalAccepted,
            totalPatterns,
            acceptanceRate: totalProposals > 0 ? totalAccepted / totalProposals : 0,
            avgConfidence: agentInsights.reduce((sum, a) => sum + a.performance.averageConfidence, 0) / agentInsights.length,
            lastActivity: new Date(Math.max(...agentInsights.map(a => a.lastActivity.getTime()))),
        };
    }

    private async updateSwarmCapabilities(swarmId: string): Promise<void> {
        const swarm = this.agentSwarms.get(swarmId);
        if (!swarm) return;
        
        // Collect agent insights
        const agentInsights: AgentInsights[] = [];
        for (const [agentId] of swarm.agents) {
            const insights = await this.getAgentInsights(agentId);
            if (insights) {
                agentInsights.push(insights);
            }
        }
        
        // Update emergent capabilities
        swarm.emergentCapabilities = this.identifyEmergentCapabilities(agentInsights);
    }

    private startHealthMonitoring(): void {
        setInterval(() => {
            this.performHealthChecks().catch(error => {
                this.logger.error("[AgentDeploymentService] Health check failed", { error });
            });
        }, this.HEALTH_CHECK_INTERVAL);
    }

    private async performHealthChecks(): Promise<void> {
        const now = new Date();
        
        for (const [agentId, deployment] of this.deployedAgents) {
            // For emergent agents, health is based on learning activity
            const insights = await this.getAgentInsights(agentId);
            if (insights) {
                const timeSinceActivity = now.getTime() - insights.lastActivity.getTime();
                
                if (timeSinceActivity > 10 * 60 * 1000) { // 10 minutes
                    deployment.status = "inactive";
                } else if (deployment.status === "inactive") {
                    deployment.status = "active";
                }
            }
            
            deployment.lastHealthCheck = now;
        }
    }
}

/**
 * Supporting interfaces for emergent agent deployment
 */
export interface EmergentAgentConfig {
    agentId: string;
    goal: string;
    initialRoutine: string;
    subscriptions: string[];
    priority?: number;
    swarmId?: string;
    filterPredicate?: (event: IntelligentEvent) => boolean;
}

export interface EmergentSwarmConfig {
    swarmId?: string;
    name: string;
    description: string;
    agents: Omit<EmergentAgentConfig, "swarmId">[];
    coordination?: {
        sharedLearning?: boolean;
        collaborativeProposals?: boolean;
        crossAgentInsights?: boolean;
    };
}

interface DeployedEmergentAgent {
    config: EmergentAgentConfig;
    agent: EmergentAgent;
    subscription: AgentSubscription;
    status: "active" | "inactive" | "error";
    deployedAt: Date;
    lastHealthCheck: Date;
    swarmId?: string;
}

interface EmergentSwarm {
    id: string;
    name: string;
    description: string;
    agents: Map<string, DeployedEmergentAgent>;
    coordination: {
        sharedLearning: boolean;
        collaborativeProposals: boolean;
        crossAgentInsights: boolean;
    };
    status: "deploying" | "active" | "inactive" | "error";
    deployedAt: Date;
    emergentCapabilities: EmergentCapability[];
}

interface EmergentCapability {
    name: string;
    description: string;
    confidence: number;
    contributingAgents: string[];
    emergentFrom: string[];
}

interface SwarmInsights {
    swarmId: string;
    name: string;
    description: string;
    status: string;
    deployedAt: Date;
    coordination: EmergentSwarm["coordination"];
    agentCount: number;
    agentInsights: AgentInsights[];
    emergentCapabilities: EmergentCapability[];
    swarmMetrics: SwarmMetrics;
}

interface SwarmMetrics {
    totalProposals: number;
    totalAccepted: number;
    totalPatterns: number;
    acceptanceRate: number;
    avgConfidence: number;
    lastActivity: Date;
}

interface AgentDeploymentInfo {
    agentId: string;
    goal: string;
    initialRoutine: string;
    status: string;
    swarmId?: string;
    deployedAt: Date;
}

interface SwarmDeploymentInfo {
    swarmId: string;
    name: string;
    description: string;
    status: string;
    agentCount: number;
    deployedAt: Date;
    emergentCapabilities: number;
}

interface DeploymentStatus {
    agents: AgentDeploymentInfo[];
    swarms: SwarmDeploymentInfo[];
    summary: {
        totalAgents: number;
        totalSwarms: number;
        activeAgents: number;
        activeSwarms: number;
    };
}
