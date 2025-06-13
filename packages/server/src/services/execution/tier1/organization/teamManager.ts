import { type Logger } from "winston";
import {
    type TeamFormation,
    type SwarmAgent,
    type AgentRole,
    type AgentCapability,
    type ConsensusRequest,
    type ConsensusResult,
    type TeamConstraints,
    SwarmEventType as SwarmEventTypeEnum,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { BaseComponent } from "../../shared/BaseComponent.js";

/**
 * Team performance metrics
 */
export interface TeamPerformance {
    cohesion: number;
    productivity: number;
    communicationEfficiency: number;
    conflictRate: number;
    adaptability: number;
}

/**
 * Agent selection criteria
 */
export interface AgentSelectionCriteria {
    requiredCapabilities: AgentCapability[];
    preferredCapabilities?: AgentCapability[];
    minPerformance?: number;
    maxWorkload?: number;
    specificAgents?: string[];
    excludeAgents?: string[];
}

/**
 * Team recommendation
 */
export interface TeamRecommendation {
    agents: SwarmAgent[];
    score: number;
    rationale: string;
    alternativeTeams?: TeamRecommendation[];
}

/**
 * TeamManager - Dynamic team formation and coordination
 * 
 * This component manages the formation, coordination, and adaptation of
 * agent teams within a swarm. It implements:
 * 
 * - Capability-based agent selection
 * - Dynamic team formation based on task requirements
 * - Consensus mechanisms for group decisions
 * - Performance monitoring and team optimization
 * - Conflict resolution and coordination protocols
 * 
 * The TeamManager ensures that the right agents work together effectively
 * to achieve swarm goals.
 */
export class TeamManager extends BaseComponent {
    private readonly teams: Map<string, TeamFormation> = new Map();
    private readonly agentRegistry: Map<string, SwarmAgent> = new Map();
    private readonly teamPerformance: Map<string, TeamPerformance> = new Map();

    constructor(logger: Logger, eventBus: EventBus) {
        super(logger, eventBus, "TeamManager");
        this.initializeAgentRegistry();
        this.subscribeToEvents();
    }

    /**
     * Forms a new team
     */
    async formTeam(formation: TeamFormation): Promise<void> {
        this.logger.info("[TeamManager] Forming new team", {
            teamId: formation.id,
            swarmId: formation.swarmId,
            agentCount: formation.agents.length,
        });

        // Validate team formation
        if (formation.constraints) {
            await this.validateTeamConstraints(formation);
        }

        // Register agents
        for (const agent of formation.agents) {
            this.agentRegistry.set(agent.id, agent);
        }

        // Store team
        this.teams.set(formation.swarmId, formation);

        // Initialize performance tracking
        this.teamPerformance.set(formation.swarmId, {
            cohesion: 1.0,
            productivity: 1.0,
            communicationEfficiency: 1.0,
            conflictRate: 0.0,
            adaptability: 1.0,
        });

        // Emit team formed event
        await this.publishEvent("swarm.events", {
            type: SwarmEventTypeEnum.TEAM_FORMED,
            swarmId: formation.swarmId,
            teamId: formation.id,
            agentCount: formation.agents.length,
        });
    }

    /**
     * Gets team for a swarm
     */
    async getTeam(swarmId: string): Promise<TeamFormation> {
        const team = this.teams.get(swarmId);
        if (!team) {
            throw new Error(`No team found for swarm ${swarmId}`);
        }
        return team;
    }

    /**
     * Updates team composition
     */
    async updateTeam(
        swarmId: string,
        updates: Partial<TeamFormation>,
    ): Promise<void> {
        const team = await this.getTeam(swarmId);
        
        // Apply updates
        Object.assign(team, updates);
        
        // Re-validate if constraints changed
        if (updates.constraints) {
            await this.validateTeamConstraints(team);
        }

        this.logger.info("[TeamManager] Team updated", {
            swarmId,
            updates: Object.keys(updates),
        });

        // Emit update event
        await this.publishEvent("swarm.events", {
            type: SwarmEventTypeEnum.TEAM_UPDATED,
            swarmId,
            updates,
        });
    }

    /**
     * Adds agent to team
     */
    async addAgent(
        swarmId: string,
        agent: SwarmAgent,
    ): Promise<void> {
        const team = await this.getTeam(swarmId);
        
        // Check constraints
        if (team.constraints?.maxSize && team.agents.length >= team.constraints.maxSize) {
            throw new Error("Team is at maximum size");
        }

        // Add agent
        team.agents.push(agent);
        this.agentRegistry.set(agent.id, agent);

        this.logger.info("[TeamManager] Agent added to team", {
            swarmId,
            agentId: agent.id,
            role: agent.role,
        });

        // Emit event
        await this.publishEvent("swarm.events", {
            type: SwarmEventTypeEnum.AGENT_JOINED,
            swarmId,
            agent,
        });
    }

    /**
     * Removes agent from team
     */
    async removeAgent(
        swarmId: string,
        agentId: string,
    ): Promise<void> {
        const team = await this.getTeam(swarmId);
        
        // Remove agent
        team.agents = team.agents.filter(a => a.id !== agentId);
        
        // Check constraints
        if (team.constraints?.minSize && team.agents.length < team.constraints.minSize) {
            this.logger.warn("[TeamManager] Team below minimum size", {
                swarmId,
                currentSize: team.agents.length,
                minSize: team.constraints.minSize,
            });
        }

        this.logger.info("[TeamManager] Agent removed from team", {
            swarmId,
            agentId,
        });

        // Emit event
        await this.publishEvent("swarm.events", {
            type: SwarmEventTypeEnum.AGENT_LEFT,
            swarmId,
            agentId,
        });
    }

    /**
     * Gets consensus from team
     */
    async getConsensus(
        swarmId: string,
        propositions: string[],
        timeout = 30000,
    ): Promise<ConsensusResult> {
        const team = await this.getTeam(swarmId);
        const request: ConsensusRequest = {
            id: generatePk(),
            swarmId,
            propositions,
            timeout,
            requiredThreshold: 0.7, // 70% default
        };

        this.logger.info("[TeamManager] Requesting consensus", {
            swarmId,
            propositionCount: propositions.length,
            timeout,
        });

        // Simulate consensus process
        // In production, this would involve actual agent voting
        const results = await this.simulateConsensus(team, propositions);

        const consensusResult: ConsensusResult = {
            requestId: request.id,
            results,
            threshold: request.requiredThreshold,
            reached: results.some(r => r >= request.requiredThreshold),
            participantCount: team.agents.length,
        };

        if (consensusResult.reached) {
            await this.publishEvent("swarm.events", {
                type: SwarmEventTypeEnum.CONSENSUS_REACHED,
                swarmId,
                consensusResult,
            });
        }

        return consensusResult;
    }

    /**
     * Recommends optimal team composition
     */
    async recommendTeam(
        swarmId: string,
        criteria: AgentSelectionCriteria,
    ): Promise<TeamRecommendation> {
        this.logger.debug("[TeamManager] Generating team recommendation", {
            swarmId,
            requiredCapabilities: criteria.requiredCapabilities,
        });

        // Get available agents
        const availableAgents = await this.getAvailableAgents(criteria);

        // Score agent combinations
        const teamCombinations = this.generateTeamCombinations(
            availableAgents,
            criteria.requiredCapabilities,
        );

        // Select best team
        const bestTeam = this.selectBestTeam(teamCombinations, criteria);

        return {
            agents: bestTeam.agents,
            score: bestTeam.score,
            rationale: bestTeam.rationale,
            alternativeTeams: teamCombinations.slice(1, 4), // Top 3 alternatives
        };
    }

    /**
     * Analyzes team performance
     */
    async analyzeTeamPerformance(swarmId: string): Promise<TeamPerformance> {
        const performance = this.teamPerformance.get(swarmId);
        if (!performance) {
            throw new Error(`No performance data for swarm ${swarmId}`);
        }

        // Update performance metrics based on recent events
        // This is simplified - real implementation would use actual metrics
        const team = await this.getTeam(swarmId);
        const activeAgents = team.agents.filter(a => a.status === "active").length;
        
        performance.productivity = activeAgents / team.agents.length;
        performance.cohesion = Math.max(0.5, performance.cohesion - (performance.conflictRate * 0.1));
        
        this.logger.debug("[TeamManager] Team performance analyzed", {
            swarmId,
            performance,
        });

        return performance;
    }

    /**
     * Resolves conflicts within team
     */
    async resolveConflict(
        swarmId: string,
        conflictType: string,
        involvedAgents: string[],
    ): Promise<void> {
        this.logger.warn("[TeamManager] Resolving team conflict", {
            swarmId,
            conflictType,
            involvedAgents,
        });

        const performance = this.teamPerformance.get(swarmId);
        if (performance) {
            performance.conflictRate = Math.min(1.0, performance.conflictRate + 0.1);
        }

        // Emit conflict event
        await this.publishEvent("swarm.events", {
            type: SwarmEventTypeEnum.CONFLICT_DETECTED,
            swarmId,
            conflictType,
            involvedAgents,
            resolution: "mediated", // Simplified
        });
    }

    /**
     * Private helper methods
     */
    private async validateTeamConstraints(formation: TeamFormation): Promise<void> {
        const constraints = formation.constraints;
        if (!constraints) return;

        // Check size constraints
        if (constraints.minSize && formation.agents.length < constraints.minSize) {
            throw new Error(`Team has ${formation.agents.length} agents, minimum is ${constraints.minSize}`);
        }
        
        if (constraints.maxSize && formation.agents.length > constraints.maxSize) {
            throw new Error(`Team has ${formation.agents.length} agents, maximum is ${constraints.maxSize}`);
        }

        // Check required capabilities
        if (constraints.requiredCapabilities) {
            const teamCapabilities = new Set<AgentCapability>();
            for (const agent of formation.agents) {
                for (const capability of agent.capabilities) {
                    teamCapabilities.add(capability as AgentCapability);
                }
            }

            for (const required of constraints.requiredCapabilities) {
                if (!teamCapabilities.has(required)) {
                    throw new Error(`Team missing required capability: ${required}`);
                }
            }
        }
    }

    private async simulateConsensus(
        team: TeamFormation,
        propositions: string[],
    ): Promise<number[]> {
        // Simulate agent voting
        // In production, this would involve actual agent deliberation
        const results: number[] = [];
        
        for (const proposition of propositions) {
            // Simple simulation based on team cohesion
            const performance = this.teamPerformance.get(team.swarmId) || {
                cohesion: 0.7,
            };
            
            // Add some randomness
            const baseScore = performance.cohesion;
            const variance = (Math.random() - 0.5) * 0.3;
            const score = Math.max(0, Math.min(1, baseScore + variance));
            
            results.push(score);
        }

        return results;
    }

    private async getAvailableAgents(
        criteria: AgentSelectionCriteria,
    ): Promise<SwarmAgent[]> {
        const agents: SwarmAgent[] = [];
        
        for (const [id, agent] of this.agentRegistry) {
            // Check if agent is available
            if (agent.status !== "active" && agent.status !== "idle") {
                continue;
            }

            // Check exclusions
            if (criteria.excludeAgents?.includes(id)) {
                continue;
            }

            // Check specific agents
            if (criteria.specificAgents && !criteria.specificAgents.includes(id)) {
                continue;
            }

            // Check capabilities
            const hasRequired = criteria.requiredCapabilities.every(cap =>
                agent.capabilities.includes(cap),
            );
            
            if (!hasRequired) {
                continue;
            }

            // Check performance
            if (criteria.minPerformance && agent.performance) {
                if (agent.performance.successRate < criteria.minPerformance) {
                    continue;
                }
            }

            agents.push(agent);
        }

        return agents;
    }

    private generateTeamCombinations(
        agents: SwarmAgent[],
        requiredCapabilities: AgentCapability[],
    ): TeamRecommendation[] {
        // Simplified team generation
        // In production, this would use combinatorial optimization
        const teams: TeamRecommendation[] = [];
        
        // Create a team with minimal agents that cover all capabilities
        const selectedAgents: SwarmAgent[] = [];
        const coveredCapabilities = new Set<AgentCapability>();
        
        for (const capability of requiredCapabilities) {
            const agent = agents.find(a => 
                a.capabilities.includes(capability) &&
                !selectedAgents.includes(a),
            );
            
            if (agent) {
                selectedAgents.push(agent);
                for (const cap of agent.capabilities) {
                    coveredCapabilities.add(cap as AgentCapability);
                }
            }
        }

        // Calculate team score
        const score = this.calculateTeamScore(selectedAgents, requiredCapabilities);
        
        teams.push({
            agents: selectedAgents,
            score,
            rationale: `Minimal team covering all ${requiredCapabilities.length} required capabilities`,
        });

        return teams;
    }

    private selectBestTeam(
        teams: TeamRecommendation[],
        criteria: AgentSelectionCriteria,
    ): TeamRecommendation {
        // Sort by score
        teams.sort((a, b) => b.score - a.score);
        return teams[0] || {
            agents: [],
            score: 0,
            rationale: "No suitable team found",
        };
    }

    private calculateTeamScore(
        agents: SwarmAgent[],
        requiredCapabilities: AgentCapability[],
    ): number {
        if (agents.length === 0) return 0;
        
        // Calculate coverage
        const teamCapabilities = new Set<AgentCapability>();
        let totalPerformance = 0;
        
        for (const agent of agents) {
            for (const cap of agent.capabilities) {
                teamCapabilities.add(cap as AgentCapability);
            }
            if (agent.performance) {
                totalPerformance += agent.performance.successRate;
            }
        }
        
        const coverage = requiredCapabilities.filter(cap => 
            teamCapabilities.has(cap),
        ).length / requiredCapabilities.length;
        
        const avgPerformance = totalPerformance / agents.length;
        const efficiency = 1 / agents.length; // Prefer smaller teams
        
        // Weighted score
        return (coverage * 0.5) + (avgPerformance * 0.3) + (efficiency * 0.2);
    }

    private initializeAgentRegistry(): void {
        // Initialize with some default agents
        // In production, this would load from a database or service
        const defaultAgents: SwarmAgent[] = [
            {
                id: "agent-1",
                role: "coordinator" as AgentRole,
                capabilities: ["planning", "communication", "resource_management"] as AgentCapability[],
                status: "active",
                performance: {
                    tasksCompleted: 10,
                    successRate: 0.9,
                    averageExecutionTime: 5000,
                    resourceEfficiency: 0.85,
                    collaborationScore: 0.95,
                },
            },
            {
                id: "agent-2",
                role: "executor" as AgentRole,
                capabilities: ["execution", "monitoring"] as AgentCapability[],
                status: "active",
                performance: {
                    tasksCompleted: 25,
                    successRate: 0.85,
                    averageExecutionTime: 3000,
                    resourceEfficiency: 0.9,
                    collaborationScore: 0.8,
                },
            },
        ];

        for (const agent of defaultAgents) {
            this.agentRegistry.set(agent.id, agent);
        }
    }

    private subscribeToEvents(): void {
        // Subscribe to agent performance updates
        this.eventBus.subscribe("agent.performance", async (event) => {
            const agent = this.agentRegistry.get(event.agentId);
            if (agent && event.performance) {
                agent.performance = event.performance;
            }
        });

        // Subscribe to agent status changes
        this.eventBus.subscribe("agent.status", async (event) => {
            const agent = this.agentRegistry.get(event.agentId);
            if (agent) {
                agent.status = event.status;
            }
        });
    }
}
