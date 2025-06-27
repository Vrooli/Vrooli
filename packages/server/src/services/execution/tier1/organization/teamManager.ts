// AI_CHECK: STARTUP_ERRORS=2 | LAST: 2025-06-25
import { type Logger } from "winston";
import {
    type TeamFormation,
    type SwarmAgent,
    SwarmEventType as SwarmEventTypeEnum,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { BaseComponent } from "../../shared/BaseComponent.js";

/**
 * MINIMAL TEAM MANAGER
 * 
 * Provides ONLY the basic infrastructure for team coordination.
 * All intelligence, optimization, and formation strategies come from emergent agents.
 * 
 * WHAT THIS DOES:
 * - Store and track team formations
 * - Emit team coordination events
 * - Basic team state management
 * 
 * WHAT THIS DOES NOT DO (EMERGENT CAPABILITIES):
 * - Agent selection algorithms (coordination agents develop these)
 * - Team formation strategies (organizational agents optimize these)
 * - Performance analysis (monitoring agents provide this)
 * - Conflict resolution protocols (mediation agents handle this)
 * - Consensus mechanisms (decision agents create these)
 * - Capability matching (intelligence agents learn optimal matching)
 */
export class TeamManager extends BaseComponent {
    private readonly teams: Map<string, TeamFormation> = new Map();
    private readonly agentRegistry: Map<string, SwarmAgent> = new Map();

    constructor(logger: Logger, eventBus: EventBus) {
        super(logger, eventBus, "TeamManager");
        this.subscribeToBasicEvents();
    }

    /**
     * Forms a new team with basic validation
     */
    async formTeam(formation: TeamFormation): Promise<void> {
        this.logger.info("[TeamManager] Forming new team", {
            teamId: formation.id,
            swarmId: formation.swarmId,
            agentCount: formation.agents.length,
        });

        // Basic validation - agents can develop sophisticated validation
        if (!formation.swarmId || !formation.agents || formation.agents.length === 0) {
            throw new Error("Invalid team formation: missing required fields");
        }

        // Register agents in simple registry
        for (const agent of formation.agents) {
            this.agentRegistry.set(agent.id, agent);
        }

        // Store team
        this.teams.set(formation.swarmId, formation);

        // Emit team formed event for agents to analyze
        await this.publishUnifiedEvent("swarm.team.formed", {
            type: SwarmEventTypeEnum.TEAM_FORMED,
            swarmId: formation.swarmId,
            teamId: formation.id,
            agentCount: formation.agents.length,
            agents: formation.agents.map(a => ({
                id: a.id,
                role: a.role,
                capabilities: a.capabilities,
            })),
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            tags: ["team", "formation", "coordination"],
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
     * Updates team composition with basic change tracking
     */
    async updateTeam(
        swarmId: string,
        updates: Partial<TeamFormation>,
    ): Promise<void> {
        const team = await this.getTeam(swarmId);
        const originalAgentCount = team.agents.length;
        
        // Apply updates
        Object.assign(team, updates);

        // Update agent registry if agents changed
        if (updates.agents) {
            for (const agent of updates.agents) {
                this.agentRegistry.set(agent.id, agent);
            }
        }

        this.logger.info("[TeamManager] Team updated", {
            swarmId,
            updates: Object.keys(updates),
            agentCountChange: team.agents.length - originalAgentCount,
        });

        // Emit update event for agents to analyze
        await this.publishUnifiedEvent("swarm.team.updated", {
            type: SwarmEventTypeEnum.TEAM_UPDATED,
            swarmId,
            teamId: team.id,
            changes: Object.keys(updates),
            agentCount: team.agents.length,
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            tags: ["team", "update", "coordination"],
        });
    }

    /**
     * Removes a team
     */
    async removeTeam(swarmId: string): Promise<void> {
        const team = this.teams.get(swarmId);
        if (!team) {
            return; // Already removed
        }

        // Remove from tracking
        this.teams.delete(swarmId);

        // Remove agents from registry (simple cleanup)
        for (const agent of team.agents) {
            this.agentRegistry.delete(agent.id);
        }

        this.logger.info("[TeamManager] Team removed", {
            swarmId,
            teamId: team.id,
        });

        // Emit removal event for agents to analyze
        await this.publishUnifiedEvent("swarm.team.dissolved", {
            type: SwarmEventTypeEnum.TEAM_DISSOLVED,
            swarmId,
            teamId: team.id,
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            tags: ["team", "dissolution", "coordination"],
        });
    }

    /**
     * Gets agent from registry
     */
    getAgent(agentId: string): SwarmAgent | undefined {
        return this.agentRegistry.get(agentId);
    }

    /**
     * Lists all teams
     */
    listTeams(): TeamFormation[] {
        return Array.from(this.teams.values());
    }

    /**
     * Gets basic team statistics for agents to analyze
     */
    getTeamStats(swarmId: string): Record<string, unknown> {
        const team = this.teams.get(swarmId);
        if (!team) {
            return {};
        }

        return {
            agentCount: team.agents.length,
            roles: team.agents.map(a => a.role),
            capabilities: team.agents.flatMap(a => a.capabilities),
            createdAt: team.createdAt,
        };
    }

    /**
     * Subscribe to basic coordination events
     */
    private subscribeToBasicEvents(): void {
        // Subscribe to basic swarm events that affect team coordination
        // Agents can subscribe to more sophisticated event patterns
        this.eventBus.on("swarm.events", async (event) => {
            if (event.type === SwarmEventTypeEnum.SWARM_DISSOLVED) {
                await this.removeTeam(event.swarmId);
            }
        });
    }

    /**
     * Cleanup resources
     */
    async cleanup(): Promise<void> {
        this.teams.clear();
        this.agentRegistry.clear();
        
        await this.publishUnifiedEvent("swarm.team.manager.shutdown", {
            type: SwarmEventTypeEnum.TEAM_MANAGER_SHUTDOWN,
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "reliable",
            priority: "high",
            tags: ["team", "shutdown", "lifecycle"],
        });
    }
}
