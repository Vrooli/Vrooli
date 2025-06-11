import { type Logger } from "winston";
import {
    type Swarm,
    type SwarmState,
    type SwarmConfig,
    type TeamFormation,
    type BlackboardItem,
    type SwarmAgent,
    type SwarmTeam,
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
 * Refactored to use BaseStore for consistent storage patterns
 */
export class InMemorySwarmStateStore implements ISwarmStateStore {
    private readonly swarmStore: IStore<Swarm>;
    private readonly teamStore: IStore<Map<string, SwarmTeam>>;
    private readonly agentStore: IStore<Map<string, SwarmAgent>>;
    private readonly blackboardStore: IStore<Map<string, BlackboardItem>>;
    private readonly resourceStore: IStore<Map<string, Set<string>>>;
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
        this.swarmStore = new InMemoryStore<Swarm>(logger);
        this.teamStore = new InMemoryStore<Map<string, SwarmTeam>>(logger);
        this.agentStore = new InMemoryStore<Map<string, SwarmAgent>>(logger);
        this.blackboardStore = new InMemoryStore<Map<string, BlackboardItem>>(logger);
        this.resourceStore = new InMemoryStore<Map<string, Set<string>>>(logger);
    }

    async createSwarm(swarmId: string, swarm: Swarm): Promise<void> {
        await this.swarmStore.set(swarmId, swarm);
        this.logger.debug("[SwarmStateStore] Created swarm", { swarmId });
    }

    async getSwarm(swarmId: string): Promise<Swarm | null> {
        return await this.swarmStore.get(swarmId);
    }

    async updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void> {
        const swarm = await this.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        const updatedSwarm = { ...swarm, ...updates, updatedAt: new Date() };
        await this.swarmStore.set(swarmId, updatedSwarm);
        this.logger.debug("[SwarmStateStore] Updated swarm", { swarmId });
    }

    async deleteSwarm(swarmId: string): Promise<void> {
        await this.swarmStore.delete(swarmId);
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
        const allSwarms = await this.swarmStore.getAll();
        for (const [id, swarm] of allSwarms) {
            if (![SwarmState.STOPPED, SwarmState.FAILED, SwarmState.TERMINATED].includes(swarm.state)) {
                active.push(id);
            }
        }
        return active;
    }

    async getSwarmsByState(state: SwarmState): Promise<string[]> {
        const matches: string[] = [];
        const allSwarms = await this.swarmStore.getAll();
        for (const [id, swarm] of allSwarms) {
            if (swarm.state === state) {
                matches.push(id);
            }
        }
        return matches;
    }

    async getSwarmsByUser(userId: string): Promise<string[]> {
        const matches: string[] = [];
        const allSwarms = await this.swarmStore.getAll();
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
        const team = await this.getTeamById(swarmId, teamId);
        if (!team) {
            throw new Error(`Team ${teamId} not found in swarm ${swarmId}`);
        }
        
        Object.assign(team, updates);
        this.logger.debug("[SwarmStateStore] Updated team", { swarmId, teamId });
    }

    async deleteTeam(swarmId: string, teamId: string): Promise<void> {
        this.teams.get(swarmId)?.delete(teamId);
        this.logger.debug("[SwarmStateStore] Deleted team", { swarmId, teamId });
    }

    async listTeams(swarmId: string): Promise<SwarmTeam[]> {
        const swarmTeams = this.teams.get(swarmId);
        return swarmTeams ? Array.from(swarmTeams.values()) : [];
    }

    // Agent management
    async registerAgent(swarmId: string, agent: SwarmAgent): Promise<void> {
        if (!this.agents.has(swarmId)) {
            this.agents.set(swarmId, new Map());
        }
        this.agents.get(swarmId)!.set(agent.id, agent);
        this.logger.debug("[SwarmStateStore] Registered agent", { swarmId, agentId: agent.id });
    }

    async getAgent(swarmId: string, agentId: string): Promise<SwarmAgent | null> {
        return this.agents.get(swarmId)?.get(agentId) || null;
    }

    async updateAgent(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void> {
        const agent = await this.getAgent(swarmId, agentId);
        if (!agent) {
            throw new Error(`Agent ${agentId} not found in swarm ${swarmId}`);
        }
        
        Object.assign(agent, updates);
        this.logger.debug("[SwarmStateStore] Updated agent", { swarmId, agentId });
    }

    async unregisterAgent(swarmId: string, agentId: string): Promise<void> {
        this.agents.get(swarmId)?.delete(agentId);
        this.logger.debug("[SwarmStateStore] Unregistered agent", { swarmId, agentId });
    }

    async listAgents(swarmId: string): Promise<SwarmAgent[]> {
        const swarmAgents = this.agents.get(swarmId);
        return swarmAgents ? Array.from(swarmAgents.values()) : [];
    }

    // Blackboard operations
    async addBlackboardItem(swarmId: string, item: BlackboardItem): Promise<void> {
        if (!this.blackboards.has(swarmId)) {
            this.blackboards.set(swarmId, new Map());
        }
        this.blackboards.get(swarmId)!.set(item.id, item);
        this.logger.debug("[SwarmStateStore] Added blackboard item", { swarmId, itemId: item.id });
    }

    async getBlackboardItems(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]> {
        const swarmBlackboard = this.blackboards.get(swarmId);
        if (!swarmBlackboard) {
            return [];
        }
        
        const items = Array.from(swarmBlackboard.values());
        return filter ? items.filter(filter) : items;
    }

    async updateBlackboardItem(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void> {
        const blackboard = this.blackboards.get(swarmId);
        const item = blackboard?.get(itemId);
        if (!item) {
            throw new Error(`Blackboard item ${itemId} not found`);
        }
        
        Object.assign(item, updates, { updatedAt: new Date() });
        this.logger.debug("[SwarmStateStore] Updated blackboard item", { swarmId, itemId });
    }

    async removeBlackboardItem(swarmId: string, itemId: string): Promise<void> {
        this.blackboards.get(swarmId)?.delete(itemId);
        this.logger.debug("[SwarmStateStore] Removed blackboard item", { swarmId, itemId });
    }

    // Resource tracking
    async allocateResource(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void> {
        if (!this.resourceAllocations.has(swarmId)) {
            this.resourceAllocations.set(swarmId, new Map());
        }
        const swarmResources = this.resourceAllocations.get(swarmId)!;
        
        if (!swarmResources.has(resource.id)) {
            swarmResources.set(resource.id, new Set());
        }
        swarmResources.get(resource.id)!.add(consumerId);
        
        this.logger.debug("[SwarmStateStore] Allocated resource", { 
            swarmId, 
            resourceId: resource.id, 
            consumerId,
        });
    }

    async releaseResource(swarmId: string, resourceId: string, consumerId: string): Promise<void> {
        this.resourceAllocations.get(swarmId)?.get(resourceId)?.delete(consumerId);
        this.logger.debug("[SwarmStateStore] Released resource", { 
            swarmId, 
            resourceId, 
            consumerId,
        });
    }

    async getResourceAllocation(swarmId: string, resourceId: string): Promise<string[]> {
        const allocations = this.resourceAllocations.get(swarmId)?.get(resourceId);
        return allocations ? Array.from(allocations) : [];
    }
}
