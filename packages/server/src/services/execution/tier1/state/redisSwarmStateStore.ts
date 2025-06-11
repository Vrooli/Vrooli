import { type Logger } from "winston";
import {
    type Swarm,
    type SwarmState,
    type TeamFormation,
    type BlackboardItem,
    type SwarmAgent,
    type SwarmTeam,
    type SwarmResource,
    SwarmStatus,
} from "@vrooli/shared";
import { redis } from "../../../../redisConn.js";
import { type ISwarmStateStore } from "./swarmStateStore.js";

/**
 * Redis-based swarm state store for production use
 * 
 * This implementation provides persistent storage for swarm state
 * using Redis with proper key namespacing and TTL management.
 */
export class RedisSwarmStateStore implements ISwarmStateStore {
    private readonly logger: Logger;
    private readonly keyPrefix = "swarm:";
    private readonly indexPrefix = "swarm_index:";
    private readonly ttl = 86400 * 7; // 7 days default TTL

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Creates a new swarm in Redis
     */
    async createSwarm(swarmId: string, swarm: Swarm): Promise<void> {
        const key = this.getSwarmKey(swarmId);
        const data = JSON.stringify(swarm);
        
        try {
            // Store swarm data
            await redis.set(key, data, "EX", this.ttl);
            
            // Update indexes
            await this.updateIndexes(swarmId, swarm);
            
            this.logger.debug("[RedisSwarmStateStore] Created swarm", { swarmId });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to create swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Retrieves a swarm from Redis
     */
    async getSwarm(swarmId: string): Promise<Swarm | null> {
        const key = this.getSwarmKey(swarmId);
        
        try {
            const data = await redis.get(key);
            if (!data) {
                return null;
            }
            
            const swarm = JSON.parse(data) as Swarm;
            
            // Refresh TTL on access
            await redis.expire(key, this.ttl);
            
            return swarm;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Updates a swarm in Redis
     */
    async updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void> {
        const swarm = await this.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        try {
            // Apply updates
            const updatedSwarm = {
                ...swarm,
                ...updates,
                updatedAt: new Date(),
            };
            
            const key = this.getSwarmKey(swarmId);
            const data = JSON.stringify(updatedSwarm);
            
            // Store updated swarm
            await redis.set(key, data, "EX", this.ttl);
            
            // Update indexes if state changed
            if (updates.state && updates.state !== swarm.state) {
                await this.updateIndexes(swarmId, updatedSwarm);
            }
            
            this.logger.debug("[RedisSwarmStateStore] Updated swarm", {
                swarmId,
                updates: Object.keys(updates),
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to update swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Deletes a swarm from Redis
     */
    async deleteSwarm(swarmId: string): Promise<void> {
        const swarm = await this.getSwarm(swarmId);
        if (!swarm) {
            return;
        }
        
        try {
            const key = this.getSwarmKey(swarmId);
            
            // Remove from indexes
            await this.removeFromIndexes(swarmId, swarm);
            
            // Delete swarm data
            await redis.del(key);
            
            this.logger.debug("[RedisSwarmStateStore] Deleted swarm", { swarmId });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to delete swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Gets the current state of a swarm
     */
    async getSwarmState(swarmId: string): Promise<SwarmState> {
        const swarm = await this.getSwarm(swarmId);
        return swarm?.state || SwarmState.UNINITIALIZED;
    }

    /**
     * Updates the state of a swarm
     */
    async updateSwarmState(swarmId: string, state: SwarmState): Promise<void> {
        await this.updateSwarm(swarmId, { state });
    }

    /**
     * Gets the team formation for a swarm
     */
    async getTeam(swarmId: string): Promise<TeamFormation | null> {
        const swarm = await this.getSwarm(swarmId);
        return swarm?.team || null;
    }

    /**
     * Updates the team formation for a swarm
     */
    async updateTeam(swarmId: string, team: TeamFormation): Promise<void> {
        await this.updateSwarm(swarmId, { team });
    }

    /**
     * Lists all active swarms
     */
    async listActiveSwarms(): Promise<string[]> {
        const activeStates = [
            SwarmState.INITIALIZING,
            SwarmState.STRATEGIZING,
            SwarmState.RESOURCE_ALLOCATION,
            SwarmState.TEAM_FORMING,
            SwarmState.READY,
            SwarmState.ACTIVE,
            SwarmState.ADAPTING,
        ];
        
        const results: string[] = [];
        
        for (const state of activeStates) {
            const swarmIds = await this.getSwarmsByState(state);
            results.push(...swarmIds);
        }
        
        return [...new Set(results)]; // Remove duplicates
    }

    /**
     * Gets swarms by state
     */
    async getSwarmsByState(state: SwarmState): Promise<string[]> {
        const key = this.getStateIndexKey(state);
        
        try {
            const members = await redis.smembers(key);
            
            // Verify swarms still exist and have correct state
            const valid: string[] = [];
            for (const swarmId of members) {
                const swarm = await this.getSwarm(swarmId);
                if (swarm && swarm.state === state) {
                    valid.push(swarmId);
                } else {
                    // Clean up stale index entry
                    await redis.srem(key, swarmId);
                }
            }
            
            return valid;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get swarms by state", {
                state,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Gets swarms by user
     */
    async getSwarmsByUser(userId: string): Promise<string[]> {
        const key = this.getUserIndexKey(userId);
        
        try {
            const members = await redis.smembers(key);
            
            // Verify swarms still exist
            const valid: string[] = [];
            for (const swarmId of members) {
                const swarm = await this.getSwarm(swarmId);
                if (swarm && swarm.metadata.userId === userId) {
                    valid.push(swarmId);
                } else {
                    // Clean up stale index entry
                    await redis.srem(key, swarmId);
                }
            }
            
            return valid;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get swarms by user", {
                userId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Team management - Enhanced
     */
    async createTeam(swarmId: string, team: SwarmTeam): Promise<void> {
        const key = this.getTeamKey(swarmId, team.id);
        const data = JSON.stringify(team);
        
        try {
            await redis.set(key, data, "EX", this.ttl);
            
            // Add to teams index
            await redis.sadd(this.getTeamsIndexKey(swarmId), team.id);
            
            this.logger.debug("[RedisSwarmStateStore] Created team", {
                swarmId,
                teamId: team.id,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to create team", {
                swarmId,
                teamId: team.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getTeamById(swarmId: string, teamId: string): Promise<SwarmTeam | null> {
        const key = this.getTeamKey(swarmId, teamId);
        
        try {
            const data = await redis.get(key);
            if (!data) {
                return null;
            }
            
            // Refresh TTL on access
            await redis.expire(key, this.ttl);
            
            return JSON.parse(data) as SwarmTeam;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get team", {
                swarmId,
                teamId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    async updateTeamById(swarmId: string, teamId: string, updates: Partial<SwarmTeam>): Promise<void> {
        const team = await this.getTeamById(swarmId, teamId);
        if (!team) {
            throw new Error(`Team ${teamId} not found in swarm ${swarmId}`);
        }
        
        try {
            const updatedTeam = {
                ...team,
                ...updates,
            };
            
            const key = this.getTeamKey(swarmId, teamId);
            const data = JSON.stringify(updatedTeam);
            
            await redis.set(key, data, "EX", this.ttl);
            
            this.logger.debug("[RedisSwarmStateStore] Updated team", {
                swarmId,
                teamId,
                updates: Object.keys(updates),
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to update team", {
                swarmId,
                teamId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async deleteTeam(swarmId: string, teamId: string): Promise<void> {
        try {
            const key = this.getTeamKey(swarmId, teamId);
            
            // Remove from teams index
            await redis.srem(this.getTeamsIndexKey(swarmId), teamId);
            
            // Delete team data
            await redis.del(key);
            
            this.logger.debug("[RedisSwarmStateStore] Deleted team", {
                swarmId,
                teamId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to delete team", {
                swarmId,
                teamId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async listTeams(swarmId: string): Promise<SwarmTeam[]> {
        try {
            const teamIds = await redis.smembers(this.getTeamsIndexKey(swarmId));
            const teams: SwarmTeam[] = [];
            
            for (const teamId of teamIds) {
                const team = await this.getTeamById(swarmId, teamId);
                if (team) {
                    teams.push(team);
                }
            }
            
            return teams;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to list teams", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Agent management
     */
    async registerAgent(swarmId: string, agent: SwarmAgent): Promise<void> {
        const key = this.getAgentKey(swarmId, agent.id);
        const data = JSON.stringify(agent);
        
        try {
            await redis.set(key, data, "EX", this.ttl);
            
            // Add to agents index
            await redis.sadd(this.getAgentsIndexKey(swarmId), agent.id);
            
            this.logger.debug("[RedisSwarmStateStore] Registered agent", {
                swarmId,
                agentId: agent.id,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to register agent", {
                swarmId,
                agentId: agent.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getAgent(swarmId: string, agentId: string): Promise<SwarmAgent | null> {
        const key = this.getAgentKey(swarmId, agentId);
        
        try {
            const data = await redis.get(key);
            if (!data) {
                return null;
            }
            
            // Refresh TTL on access
            await redis.expire(key, this.ttl);
            
            return JSON.parse(data) as SwarmAgent;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get agent", {
                swarmId,
                agentId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    async updateAgent(swarmId: string, agentId: string, updates: Partial<SwarmAgent>): Promise<void> {
        const agent = await this.getAgent(swarmId, agentId);
        if (!agent) {
            throw new Error(`Agent ${agentId} not found in swarm ${swarmId}`);
        }
        
        try {
            const updatedAgent = {
                ...agent,
                ...updates,
            };
            
            const key = this.getAgentKey(swarmId, agentId);
            const data = JSON.stringify(updatedAgent);
            
            await redis.set(key, data, "EX", this.ttl);
            
            this.logger.debug("[RedisSwarmStateStore] Updated agent", {
                swarmId,
                agentId,
                updates: Object.keys(updates),
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to update agent", {
                swarmId,
                agentId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async unregisterAgent(swarmId: string, agentId: string): Promise<void> {
        try {
            const key = this.getAgentKey(swarmId, agentId);
            
            // Remove from agents index
            await redis.srem(this.getAgentsIndexKey(swarmId), agentId);
            
            // Delete agent data
            await redis.del(key);
            
            this.logger.debug("[RedisSwarmStateStore] Unregistered agent", {
                swarmId,
                agentId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to unregister agent", {
                swarmId,
                agentId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async listAgents(swarmId: string): Promise<SwarmAgent[]> {
        try {
            const agentIds = await redis.smembers(this.getAgentsIndexKey(swarmId));
            const agents: SwarmAgent[] = [];
            
            for (const agentId of agentIds) {
                const agent = await this.getAgent(swarmId, agentId);
                if (agent) {
                    agents.push(agent);
                }
            }
            
            return agents;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to list agents", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Blackboard operations
     */
    async addBlackboardItem(swarmId: string, item: BlackboardItem): Promise<void> {
        const key = this.getBlackboardItemKey(swarmId, item.id);
        const data = JSON.stringify(item);
        
        try {
            await redis.set(key, data, "EX", this.ttl);
            
            // Add to blackboard index
            await redis.sadd(this.getBlackboardIndexKey(swarmId), item.id);
            
            this.logger.debug("[RedisSwarmStateStore] Added blackboard item", {
                swarmId,
                itemId: item.id,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to add blackboard item", {
                swarmId,
                itemId: item.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getBlackboardItems(swarmId: string, filter?: (item: BlackboardItem) => boolean): Promise<BlackboardItem[]> {
        try {
            const itemIds = await redis.smembers(this.getBlackboardIndexKey(swarmId));
            const items: BlackboardItem[] = [];
            
            for (const itemId of itemIds) {
                const key = this.getBlackboardItemKey(swarmId, itemId);
                const data = await redis.get(key);
                if (data) {
                    const item = JSON.parse(data) as BlackboardItem;
                    if (!filter || filter(item)) {
                        items.push(item);
                    }
                }
            }
            
            return items;
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get blackboard items", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    async updateBlackboardItem(swarmId: string, itemId: string, updates: Partial<BlackboardItem>): Promise<void> {
        const key = this.getBlackboardItemKey(swarmId, itemId);
        
        try {
            const data = await redis.get(key);
            if (!data) {
                throw new Error(`Blackboard item ${itemId} not found`);
            }
            
            const item = JSON.parse(data) as BlackboardItem;
            const updatedItem = {
                ...item,
                ...updates,
                updatedAt: new Date(),
            };
            
            await redis.set(key, JSON.stringify(updatedItem), "EX", this.ttl);
            
            this.logger.debug("[RedisSwarmStateStore] Updated blackboard item", {
                swarmId,
                itemId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to update blackboard item", {
                swarmId,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async removeBlackboardItem(swarmId: string, itemId: string): Promise<void> {
        try {
            const key = this.getBlackboardItemKey(swarmId, itemId);
            
            // Remove from blackboard index
            await redis.srem(this.getBlackboardIndexKey(swarmId), itemId);
            
            // Delete item data
            await redis.del(key);
            
            this.logger.debug("[RedisSwarmStateStore] Removed blackboard item", {
                swarmId,
                itemId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to remove blackboard item", {
                swarmId,
                itemId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Resource allocation tracking
     */
    async allocateResource(swarmId: string, resource: SwarmResource, consumerId: string): Promise<void> {
        const key = this.getResourceAllocationKey(swarmId, resource.id);
        
        try {
            await redis.sadd(key, consumerId);
            await redis.expire(key, this.ttl);
            
            this.logger.debug("[RedisSwarmStateStore] Allocated resource", {
                swarmId,
                resourceId: resource.id,
                consumerId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to allocate resource", {
                swarmId,
                resourceId: resource.id,
                consumerId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async releaseResource(swarmId: string, resourceId: string, consumerId: string): Promise<void> {
        const key = this.getResourceAllocationKey(swarmId, resourceId);
        
        try {
            await redis.srem(key, consumerId);
            
            this.logger.debug("[RedisSwarmStateStore] Released resource", {
                swarmId,
                resourceId,
                consumerId,
            });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to release resource", {
                swarmId,
                resourceId,
                consumerId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    async getResourceAllocation(swarmId: string, resourceId: string): Promise<string[]> {
        const key = this.getResourceAllocationKey(swarmId, resourceId);
        
        try {
            return await redis.smembers(key);
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to get resource allocation", {
                swarmId,
                resourceId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Private helper methods
     */
    private getSwarmKey(swarmId: string): string {
        return `${this.keyPrefix}${swarmId}`;
    }

    private getStateIndexKey(state: SwarmState): string {
        return `${this.indexPrefix}state:${state}`;
    }

    private getUserIndexKey(userId: string): string {
        return `${this.indexPrefix}user:${userId}`;
    }

    private getTeamKey(swarmId: string, teamId: string): string {
        return `${this.keyPrefix}${swarmId}:team:${teamId}`;
    }

    private getTeamsIndexKey(swarmId: string): string {
        return `${this.keyPrefix}${swarmId}:teams`;
    }

    private getAgentKey(swarmId: string, agentId: string): string {
        return `${this.keyPrefix}${swarmId}:agent:${agentId}`;
    }

    private getAgentsIndexKey(swarmId: string): string {
        return `${this.keyPrefix}${swarmId}:agents`;
    }

    private getBlackboardItemKey(swarmId: string, itemId: string): string {
        return `${this.keyPrefix}${swarmId}:blackboard:${itemId}`;
    }

    private getBlackboardIndexKey(swarmId: string): string {
        return `${this.keyPrefix}${swarmId}:blackboard`;
    }

    private getResourceAllocationKey(swarmId: string, resourceId: string): string {
        return `${this.keyPrefix}${swarmId}:resource:${resourceId}:allocations`;
    }

    private async updateIndexes(swarmId: string, swarm: Swarm): Promise<void> {
        // Update state index
        if (swarm.state) {
            // Remove from all state indexes first
            for (const state of Object.values(SwarmState)) {
                await redis.srem(this.getStateIndexKey(state as SwarmState), swarmId);
            }
            // Add to current state index
            await redis.sadd(this.getStateIndexKey(swarm.state), swarmId);
        }
        
        // Update user index
        if (swarm.metadata.userId) {
            await redis.sadd(this.getUserIndexKey(swarm.metadata.userId), swarmId);
        }
    }

    private async removeFromIndexes(swarmId: string, swarm: Swarm): Promise<void> {
        // Remove from state index
        if (swarm.state) {
            await redis.srem(this.getStateIndexKey(swarm.state), swarmId);
        }
        
        // Remove from user index
        if (swarm.metadata.userId) {
            await redis.srem(this.getUserIndexKey(swarm.metadata.userId), swarmId);
        }
    }
}