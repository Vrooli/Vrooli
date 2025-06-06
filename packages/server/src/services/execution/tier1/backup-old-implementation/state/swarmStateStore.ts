/**
 * Swarm state store
 * Manages persistent state for swarm coordination
 */

import { Redis } from "ioredis";
import { 
    SwarmState, 
    ChatConfigObject,
    SwarmSubTask,
    SwarmResource,
    BlackboardItem,
    SwarmAgent,
    SwarmTeam,
    MOISEOrganization
} from "@local/shared";
import { getRedisConnection } from "../../../../redisConn.js";
import { logger } from "../../../../events/logger.js";

/**
 * Swarm state store interface
 */
export interface ISwarmStateStore {
    // Swarm lifecycle
    createSwarm(swarmId: string, config: ChatConfigObject): Promise<void>;
    getSwarm(swarmId: string): Promise<ChatConfigObject | null>;
    updateSwarm(swarmId: string, updates: Partial<ChatConfigObject>): Promise<void>;
    deleteSwarm(swarmId: string): Promise<void>;
    
    // State management
    getSwarmState(swarmId: string): Promise<SwarmState>;
    updateSwarmState(swarmId: string, state: SwarmState): Promise<void>;
    
    // Team management
    createTeam(swarmId: string, team: SwarmTeam): Promise<void>;
    getTeam(swarmId: string, teamId: string): Promise<SwarmTeam | null>;
    updateTeam(swarmId: string, teamId: string, updates: Partial<SwarmTeam>): Promise<void>;
    deleteTeam(swarmId: string, teamId: string): Promise<void>;
    listTeams(swarmId: string): Promise<SwarmTeam[]>;
    
    // Agent management
    registerAgent(swarmId: string, agent: SwarmAgent): Promise<void>;
    getAgent(swarmId: string, agentId: string): Promise<SwarmAgent | null>;
    updateAgent(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void>;
    unregisterAgent(swarmId: string, agentId: string): Promise<void>;
    listAgents(swarmId: string): Promise<SwarmAgent[]>;
    
    // Blackboard operations
    addBlackboardItem(swarmId: string, item: BlackboardItem): Promise<void>;
    getBlackboardItems(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]>;
    updateBlackboardItem(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void>;
    removeBlackboardItem(swarmId: string, itemId: string): Promise<void>;
    
    // Resource tracking
    allocateResource(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void>;
    releaseResource(swarmId: string, resourceId: string, consumerId: string): Promise<void>;
    getResourceAllocation(swarmId: string, resourceId: string): Promise<string[]>;
    
    // Query operations
    listActiveSwarms(): Promise<string[]>;
    getSwarmsByState(state: SwarmState): Promise<string[]>;
}

/**
 * Redis-based swarm state store
 */
export class RedisSwarmStateStore implements ISwarmStateStore {
    private client: Redis | null = null;
    private readonly prefix = "swarm:state:";
    private readonly ttl = 86400; // 24 hours
    
    async initialize(): Promise<void> {
        this.client = await getRedisConnection();
    }
    
    private getKey(swarmId: string, ...parts: string[]): string {
        return [this.prefix, swarmId, ...parts].join(":");
    }
    
    async createSwarm(swarmId: string, config: ChatConfigObject): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "config");
        await this.client.setex(key, this.ttl, JSON.stringify(config));
        
        // Initialize state
        await this.updateSwarmState(swarmId, SwarmState.INITIALIZED);
        
        // Add to active swarms set
        await this.client.sadd(this.prefix + "active", swarmId);
        
        logger.info("Swarm created", { swarmId });
    }
    
    async getSwarm(swarmId: string): Promise<ChatConfigObject | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "config");
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateSwarm(swarmId: string, updates: Partial<ChatConfigObject>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const current = await this.getSwarm(swarmId);
        if (!current) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        const updated = {
            ...current,
            ...updates,
            updatedAt: new Date(),
        };
        
        const key = this.getKey(swarmId, "config");
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async deleteSwarm(swarmId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        // Delete all swarm-related keys
        const pattern = this.getKey(swarmId, "*");
        const keys = await this.client.keys(pattern);
        
        if (keys.length > 0) {
            await this.client.del(...keys);
        }
        
        // Remove from active swarms
        await this.client.srem(this.prefix + "active", swarmId);
        
        logger.info("Swarm deleted", { swarmId });
    }
    
    async getSwarmState(swarmId: string): Promise<SwarmState> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "state");
        const state = await this.client.get(key);
        
        return (state as SwarmState) || SwarmState.INITIALIZED;
    }
    
    async updateSwarmState(swarmId: string, state: SwarmState): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "state");
        await this.client.setex(key, this.ttl, state);
        
        // Update state index
        const stateKey = this.prefix + "by-state:" + state;
        await this.client.sadd(stateKey, swarmId);
        
        // Remove from other state indexes
        for (const s of Object.values(SwarmState)) {
            if (s !== state) {
                const oldStateKey = this.prefix + "by-state:" + s;
                await this.client.srem(oldStateKey, swarmId);
            }
        }
    }
    
    async createTeam(swarmId: string, team: SwarmTeam): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "teams", team.id);
        await this.client.setex(key, this.ttl, JSON.stringify(team));
        
        // Add to teams set
        await this.client.sadd(this.getKey(swarmId, "team-ids"), team.id);
    }
    
    async getTeam(swarmId: string, teamId: string): Promise<SwarmTeam | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "teams", teamId);
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateTeam(swarmId: string, teamId: string, updates: Partial<SwarmTeam>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const current = await this.getTeam(swarmId, teamId);
        if (!current) {
            throw new Error(`Team ${teamId} not found`);
        }
        
        const updated = { ...current, ...updates };
        const key = this.getKey(swarmId, "teams", teamId);
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async deleteTeam(swarmId: string, teamId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "teams", teamId);
        await this.client.del(key);
        await this.client.srem(this.getKey(swarmId, "team-ids"), teamId);
    }
    
    async listTeams(swarmId: string): Promise<SwarmTeam[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const teamIds = await this.client.smembers(this.getKey(swarmId, "team-ids"));
        const teams: SwarmTeam[] = [];
        
        for (const teamId of teamIds) {
            const team = await this.getTeam(swarmId, teamId);
            if (team) {
                teams.push(team);
            }
        }
        
        return teams;
    }
    
    async registerAgent(swarmId: string, agent: SwarmAgent): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "agents", agent.id);
        await this.client.setex(key, this.ttl, JSON.stringify(agent));
        
        // Add to agents set
        await this.client.sadd(this.getKey(swarmId, "agent-ids"), agent.id);
    }
    
    async getAgent(swarmId: string, agentId: string): Promise<SwarmAgent | null> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "agents", agentId);
        const data = await this.client.get(key);
        
        return data ? JSON.parse(data) : null;
    }
    
    async updateAgent(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const current = await this.getAgent(swarmId, agentId);
        if (!current) {
            throw new Error(`Agent ${agentId} not found`);
        }
        
        const updated = { ...current, ...updates };
        const key = this.getKey(swarmId, "agents", agentId);
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async unregisterAgent(swarmId: string, agentId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "agents", agentId);
        await this.client.del(key);
        await this.client.srem(this.getKey(swarmId, "agent-ids"), agentId);
    }
    
    async listAgents(swarmId: string): Promise<SwarmAgent[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const agentIds = await this.client.smembers(this.getKey(swarmId, "agent-ids"));
        const agents: SwarmAgent[] = [];
        
        for (const agentId of agentIds) {
            const agent = await this.getAgent(swarmId, agentId);
            if (agent) {
                agents.push(agent);
            }
        }
        
        return agents;
    }
    
    async addBlackboardItem(swarmId: string, item: BlackboardItem): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "blackboard", item.id);
        await this.client.setex(key, this.ttl, JSON.stringify(item));
        
        // Add to blackboard set
        await this.client.sadd(this.getKey(swarmId, "blackboard-ids"), item.id);
    }
    
    async getBlackboardItems(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const itemIds = await this.client.smembers(this.getKey(swarmId, "blackboard-ids"));
        const items: BlackboardItem[] = [];
        
        for (const itemId of itemIds) {
            const key = this.getKey(swarmId, "blackboard", itemId);
            const data = await this.client.get(key);
            if (data) {
                const item = JSON.parse(data);
                if (!filter || filter(item)) {
                    items.push(item);
                }
            }
        }
        
        return items;
    }
    
    async updateBlackboardItem(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "blackboard", itemId);
        const data = await this.client.get(key);
        if (!data) {
            throw new Error(`Blackboard item ${itemId} not found`);
        }
        
        const current = JSON.parse(data);
        const updated = { ...current, ...updates };
        await this.client.setex(key, this.ttl, JSON.stringify(updated));
    }
    
    async removeBlackboardItem(swarmId: string, itemId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "blackboard", itemId);
        await this.client.del(key);
        await this.client.srem(this.getKey(swarmId, "blackboard-ids"), itemId);
    }
    
    async allocateResource(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "resources", resource.id, "allocations");
        await this.client.sadd(key, consumerId);
        await this.client.expire(key, this.ttl);
    }
    
    async releaseResource(swarmId: string, resourceId: string, consumerId: string): Promise<void> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "resources", resourceId, "allocations");
        await this.client.srem(key, consumerId);
    }
    
    async getResourceAllocation(swarmId: string, resourceId: string): Promise<string[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.getKey(swarmId, "resources", resourceId, "allocations");
        return await this.client.smembers(key);
    }
    
    async listActiveSwarms(): Promise<string[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        return await this.client.smembers(this.prefix + "active");
    }
    
    async getSwarmsByState(state: SwarmState): Promise<string[]> {
        if (!this.client) throw new Error("Store not initialized");
        
        const key = this.prefix + "by-state:" + state;
        return await this.client.smembers(key);
    }
}

// Singleton instance
let swarmStateStore: RedisSwarmStateStore | null = null;

/**
 * Get the swarm state store instance
 */
export function getSwarmStateStore(): RedisSwarmStateStore {
    if (!swarmStateStore) {
        swarmStateStore = new RedisSwarmStateStore();
    }
    return swarmStateStore;
}