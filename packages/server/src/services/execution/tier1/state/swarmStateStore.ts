import { type Logger } from "winston";
import {
    type Swarm,
    SwarmState,
    type SwarmConfig,
    type TeamFormation,
    type SwarmAgent,
    type SwarmTeam,
} from "@vrooli/shared";
import {
    type BlackboardItem,
    type SwarmResource,
} from "@vrooli/shared";
import { InMemoryStore, type IStore } from "../../shared/BaseStore.js";

/**
 * Swarm state store interface
 */
export interface ISwarmStateStore {
    // Swarm lifecycle
    createSwarm(swarmId: string, swarm: Swarm): Promise<void>;
    getSwarm(swarmId: string): Promise<Swarm | null>;
    updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void>;
    deleteSwarm(swarmId: string): Promise<void>;
    
    // State management
    getSwarmState(swarmId: string): Promise<SwarmState>;
    updateSwarmState(swarmId: string, state: SwarmState): Promise<void>;
    
    // Team management
    getTeam(swarmId: string): Promise<TeamFormation | null>;
    updateTeam(swarmId: string, team: TeamFormation): Promise<void>;
    createTeam?(swarmId: string, team: SwarmTeam): Promise<void>;
    getTeamById?(swarmId: string, teamId: string): Promise<SwarmTeam | null>;
    updateTeamById?(swarmId: string, teamId: string, updates: Partial<SwarmTeam>): Promise<void>;
    deleteTeam?(swarmId: string, teamId: string): Promise<void>;
    listTeams?(swarmId: string): Promise<SwarmTeam[]>;
    
    // Agent management
    registerAgent?(swarmId: string, agent: SwarmAgent): Promise<void>;
    getAgent?(swarmId: string, agentId: string): Promise<SwarmAgent | null>;
    updateAgent?(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void>;
    unregisterAgent?(swarmId: string, agentId: string): Promise<void>;
    listAgents?(swarmId: string): Promise<SwarmAgent[]>;
    
    // Blackboard operations
    addBlackboardItem?(swarmId: string, item: BlackboardItem): Promise<void>;
    getBlackboardItems?(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]>;
    updateBlackboardItem?(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void>;
    removeBlackboardItem?(swarmId: string, itemId: string): Promise<void>;
    
    // Resource tracking
    allocateResource?(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void>;
    releaseResource?(swarmId: string, resourceId: string, consumerId: string): Promise<void>;
    getResourceAllocation?(swarmId: string, resourceId: string): Promise<string[]>;
    
    // Query operations
    listActiveSwarms(): Promise<string[]>;
    getSwarmsByState(state: SwarmState): Promise<string[]>;
    getSwarmsByUser(userId: string): Promise<string[]>;
}

/**
 * In-memory swarm state store (for development)
 * Now extends InMemoryStore for the main swarm storage
 */
export class InMemorySwarmStateStore extends InMemoryStore<Swarm> implements ISwarmStateStore {
    // Use composition for subsidiary stores
    private readonly teamStore: InMemoryStore<Map<string, SwarmTeam>>;
    private readonly agentStore: InMemoryStore<Map<string, SwarmAgent>>;
    private readonly blackboardStore: InMemoryStore<Map<string, BlackboardItem>>;
    private readonly resourceStore: InMemoryStore<Map<string, Set<string>>>;

    constructor(logger: Logger) {
        super(logger);
        this.teamStore = new InMemoryStore<Map<string, SwarmTeam>>(logger);
        this.agentStore = new InMemoryStore<Map<string, SwarmAgent>>(logger);
        this.blackboardStore = new InMemoryStore<Map<string, BlackboardItem>>(logger);
        this.resourceStore = new InMemoryStore<Map<string, Set<string>>>(logger);
    }

    async createSwarm(swarmId: string, swarm: Swarm): Promise<void> {
        await this.set(swarmId, swarm);
        this.logger.debug("[SwarmStateStore] Created swarm", { swarmId });
    }

    async getSwarm(swarmId: string): Promise<Swarm | null> {
        return await this.get(swarmId);
    }

    async updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void> {
        const swarm = await this.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        const updatedSwarm = { ...swarm, ...updates, updatedAt: new Date() };
        await this.set(swarmId, updatedSwarm);
        this.logger.debug("[SwarmStateStore] Updated swarm", { swarmId });
    }

    async deleteSwarm(swarmId: string): Promise<void> {
        await this.delete(swarmId);
        // Clean up subsidiary data
        await this.teamStore.delete(swarmId);
        await this.agentStore.delete(swarmId);
        await this.blackboardStore.delete(swarmId);
        await this.resourceStore.delete(swarmId);
        this.logger.debug("[SwarmStateStore] Deleted swarm", { swarmId });
    }

    async getSwarmState(swarmId: string): Promise<SwarmState> {
        const swarm = await this.getSwarm(swarmId);
        return swarm?.state || SwarmState.UNINITIALIZED;
    }

    async updateSwarmState(swarmId: string, state: SwarmState): Promise<void> {
        await this.updateSwarm(swarmId, { state });
    }

    async getTeam(swarmId: string): Promise<TeamFormation | null> {
        const swarm = await this.getSwarm(swarmId);
        return swarm?.team || null;
    }

    async updateTeam(swarmId: string, team: TeamFormation): Promise<void> {
        await this.updateSwarm(swarmId, { team });
    }

    async listActiveSwarms(): Promise<string[]> {
        const active: string[] = [];
        const allSwarms = await this.getAll();
        for (const [id, swarm] of allSwarms) {
            if (![SwarmState.STOPPED, SwarmState.FAILED, SwarmState.TERMINATED].includes(swarm.state)) {
                active.push(id);
            }
        }
        return active;
    }

    async getSwarmsByState(state: SwarmState): Promise<string[]> {
        const matches: string[] = [];
        const allSwarms = await this.getAll();
        for (const [id, swarm] of allSwarms) {
            if (swarm.state === state) {
                matches.push(id);
            }
        }
        return matches;
    }

    async getSwarmsByUser(userId: string): Promise<string[]> {
        const matches: string[] = [];
        const allSwarms = await this.getAll();
        for (const [id, swarm] of allSwarms) {
            if (swarm.metadata.userId === userId) {
                matches.push(id);
            }
        }
        return matches;
    }

    // Team management - Enhanced
    async createTeam(swarmId: string, team: SwarmTeam): Promise<void> {
        let teams = await this.teamStore.get(swarmId);
        if (!teams) {
            teams = new Map();
        }
        teams.set(team.id, team);
        await this.teamStore.set(swarmId, teams);
        this.logger.debug("[SwarmStateStore] Created team", { swarmId, teamId: team.id });
    }

    async getTeamById(swarmId: string, teamId: string): Promise<SwarmTeam | null> {
        const teams = await this.teamStore.get(swarmId);
        return teams?.get(teamId) || null;
    }

    async updateTeamById(swarmId: string, teamId: string, updates: Partial<SwarmTeam>): Promise<void> {
        const teams = await this.teamStore.get(swarmId);
        if (!teams) {
            throw new Error(`No teams found for swarm ${swarmId}`);
        }
        
        const team = teams.get(teamId);
        if (!team) {
            throw new Error(`Team ${teamId} not found in swarm ${swarmId}`);
        }
        
        teams.set(teamId, { ...team, ...updates });
        await this.teamStore.set(swarmId, teams);
        this.logger.debug("[SwarmStateStore] Updated team", { swarmId, teamId });
    }

    async deleteTeam(swarmId: string, teamId: string): Promise<void> {
        const teams = await this.teamStore.get(swarmId);
        if (teams) {
            teams.delete(teamId);
            await this.teamStore.set(swarmId, teams);
        }
        this.logger.debug("[SwarmStateStore] Deleted team", { swarmId, teamId });
    }

    async listTeams(swarmId: string): Promise<SwarmTeam[]> {
        const teams = await this.teamStore.get(swarmId);
        return teams ? Array.from(teams.values()) : [];
    }

    // Agent management
    async registerAgent(swarmId: string, agent: SwarmAgent): Promise<void> {
        let agents = await this.agentStore.get(swarmId);
        if (!agents) {
            agents = new Map();
        }
        agents.set(agent.id, agent);
        await this.agentStore.set(swarmId, agents);
        this.logger.debug("[SwarmStateStore] Registered agent", { swarmId, agentId: agent.id });
    }

    async getAgent(swarmId: string, agentId: string): Promise<SwarmAgent | null> {
        const agents = await this.agentStore.get(swarmId);
        return agents?.get(agentId) || null;
    }

    async updateAgent(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void> {
        const agents = await this.agentStore.get(swarmId);
        if (!agents) {
            throw new Error(`No agents found for swarm ${swarmId}`);
        }
        
        const agent = agents.get(agentId);
        if (!agent) {
            throw new Error(`Agent ${agentId} not found in swarm ${swarmId}`);
        }
        
        agents.set(agentId, { ...agent, ...updates });
        await this.agentStore.set(swarmId, agents);
        this.logger.debug("[SwarmStateStore] Updated agent", { swarmId, agentId });
    }

    async unregisterAgent(swarmId: string, agentId: string): Promise<void> {
        const agents = await this.agentStore.get(swarmId);
        if (agents) {
            agents.delete(agentId);
            await this.agentStore.set(swarmId, agents);
        }
        this.logger.debug("[SwarmStateStore] Unregistered agent", { swarmId, agentId });
    }

    async listAgents(swarmId: string): Promise<SwarmAgent[]> {
        const agents = await this.agentStore.get(swarmId);
        return agents ? Array.from(agents.values()) : [];
    }

    // Blackboard operations
    async addBlackboardItem(swarmId: string, item: BlackboardItem): Promise<void> {
        let blackboard = await this.blackboardStore.get(swarmId);
        if (!blackboard) {
            blackboard = new Map();
        }
        blackboard.set(item.id, item);
        await this.blackboardStore.set(swarmId, blackboard);
        this.logger.debug("[SwarmStateStore] Added blackboard item", { swarmId, itemId: item.id });
    }

    async getBlackboardItems(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]> {
        const blackboard = await this.blackboardStore.get(swarmId);
        if (!blackboard) {
            return [];
        }
        
        const items = Array.from(blackboard.values());
        return filter ? items.filter(filter) : items;
    }

    async updateBlackboardItem(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void> {
        const blackboard = await this.blackboardStore.get(swarmId);
        if (!blackboard) {
            throw new Error(`No blackboard found for swarm ${swarmId}`);
        }
        
        const item = blackboard.get(itemId);
        if (!item) {
            throw new Error(`Blackboard item ${itemId} not found`);
        }
        
        blackboard.set(itemId, { ...item, ...updates, updatedAt: new Date() });
        await this.blackboardStore.set(swarmId, blackboard);
        this.logger.debug("[SwarmStateStore] Updated blackboard item", { swarmId, itemId });
    }

    async removeBlackboardItem(swarmId: string, itemId: string): Promise<void> {
        const blackboard = await this.blackboardStore.get(swarmId);
        if (blackboard) {
            blackboard.delete(itemId);
            await this.blackboardStore.set(swarmId, blackboard);
        }
        this.logger.debug("[SwarmStateStore] Removed blackboard item", { swarmId, itemId });
    }

    // Resource tracking
    async allocateResource(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void> {
        let resources = await this.resourceStore.get(swarmId);
        if (!resources) {
            resources = new Map();
        }
        
        if (!resources.has(resource.id)) {
            resources.set(resource.id, new Set());
        }
        resources.get(resource.id)!.add(consumerId);
        
        await this.resourceStore.set(swarmId, resources);
        
        this.logger.debug("[SwarmStateStore] Allocated resource", { 
            swarmId, 
            resourceId: resource.id, 
            consumerId,
        });
    }

    async releaseResource(swarmId: string, resourceId: string, consumerId: string): Promise<void> {
        const resources = await this.resourceStore.get(swarmId);
        if (resources) {
            const consumers = resources.get(resourceId);
            if (consumers) {
                consumers.delete(consumerId);
                if (consumers.size === 0) {
                    resources.delete(resourceId);
                }
                await this.resourceStore.set(swarmId, resources);
            }
        }
        this.logger.debug("[SwarmStateStore] Released resource", { 
            swarmId, 
            resourceId, 
            consumerId,
        });
    }

    async getResourceAllocation(swarmId: string, resourceId: string): Promise<string[]> {
        const resources = await this.resourceStore.get(swarmId);
        const allocations = resources?.get(resourceId);
        return allocations ? Array.from(allocations) : [];
    }
}
