import { type Logger } from "winston";
import {
    type Swarm,
    type ExecutionState,
    ExecutionStates,
    type TeamFormation,
    type BlackboardItem,
    type SwarmAgent,
    type SwarmTeam,
    type SwarmResource,
} from "@vrooli/shared";
import { CacheService } from "../../../../redisConn.js";
import { RedisStore } from "../../shared/BaseStore.js";
import { RedisIndexManager } from "../../shared/RedisIndexManager.js";
import { type ISwarmStateStore } from "./swarmStateStore.js";

/**
 * Redis-based swarm state store for production use
 * 
 * This implementation extends RedisStore for basic CRUD operations
 * and adds swarm-specific functionality on top.
 */
export class RedisSwarmStateStore extends RedisStore<Swarm> implements ISwarmStateStore {
    private readonly indexPrefix = "swarm_index:";
    private readonly indexManager: RedisIndexManager;

    private constructor(logger: Logger, redis: any) {
        super(logger, redis, "swarm", 86400 * 7); // 7 days TTL
        this.indexManager = new RedisIndexManager(redis, logger, 86400 * 7);
    }

    static async create(logger: Logger): Promise<RedisSwarmStateStore> {
        const redis = await CacheService.get().raw();
        return new RedisSwarmStateStore(logger, redis);
    }

    /**
     * Creates a new swarm - wraps base set method with index updates
     */
    async createSwarm(swarmId: string, swarm: Swarm): Promise<void> {
        await this.set(swarmId, swarm);
        await this.updateIndexes(swarmId, swarm);
    }

    /**
     * Gets a swarm - delegates to base get method
     */
    async getSwarm(swarmId: string): Promise<Swarm | null> {
        return this.get(swarmId);
    }

    /**
     * Updates a swarm - uses base methods with index updates
     */
    async updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void> {
        const swarm = await this.get(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        const oldState = swarm.state;
        const updatedSwarm = {
            ...swarm,
            ...updates,
            updatedAt: new Date(),
        };
        
        await this.set(swarmId, updatedSwarm);
        
        // Update indexes if state changed
        if (updates.state && updates.state !== oldState) {
            await this.updateIndexes(swarmId, updatedSwarm);
        }
    }

    /**
     * Deletes a swarm - uses base delete with index cleanup
     */
    async deleteSwarm(swarmId: string): Promise<void> {
        const swarm = await this.get(swarmId);
        if (!swarm) {
            return;
        }
        
        // Remove from indexes first
        await this.removeFromIndexes(swarmId, swarm);
        
        // Clean up subsidiary data
        await this.cleanupSwarmData(swarmId);
        
        // Delete main swarm
        await this.delete(swarmId);
    }

    /**
     * Gets the current state of a swarm
     */
    async getSwarmState(swarmId: string): Promise<ExecutionState> {
        const swarm = await this.getSwarm(swarmId);
        return swarm?.state || ExecutionStates.UNINITIALIZED;
    }

    /**
     * Updates the state of a swarm
     */
    async updateSwarmState(swarmId: string, state: ExecutionState): Promise<void> {
        const oldState = await this.getSwarmState(swarmId);
        await this.updateSwarm(swarmId, { state });
        
        // Use IndexManager for state transition
        await this.indexManager.updateStateIndex(
            swarmId,
            oldState === ExecutionStates.UNINITIALIZED ? null : oldState,
            state,
            (s) => this.getStateIndexKey(s),
            Object.values(ExecutionStates),
        );
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
            ExecutionStates.STARTING,
            ExecutionStates.RUNNING,
            ExecutionStates.IDLE,
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
    async getSwarmsByState(state: ExecutionState): Promise<string[]> {
        const key = this.getStateIndexKey(state);
        
        try {
            const members = await this.indexManager.getSetMembers(key);
            
            // Verify swarms still exist and have correct state
            const valid: string[] = [];
            for (const swarmId of members) {
                const swarm = await this.getSwarm(swarmId);
                if (swarm && swarm.state === state) {
                    valid.push(swarmId);
                } else {
                    // Clean up stale index entry
                    await this.indexManager.removeFromSet(key, swarmId);
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
            const members = await this.indexManager.getSetMembers(key);
            
            // Verify swarms still exist
            const valid: string[] = [];
            for (const swarmId of members) {
                const swarm = await this.getSwarm(swarmId);
                if (swarm && swarm.metadata.userId === userId) {
                    valid.push(swarmId);
                } else {
                    // Clean up stale index entry
                    await this.indexManager.removeFromSet(key, swarmId);
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
            
            // Add to teams index using IndexManager
            await this.indexManager.addToSet(this.getTeamsIndexKey(swarmId), team.id);
            
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
            
            // Remove from teams index using IndexManager
            await this.indexManager.removeFromSet(this.getTeamsIndexKey(swarmId), teamId);
            
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
            const teamIds = await this.indexManager.getSetMembers(this.getTeamsIndexKey(swarmId));
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
            
            // Add to agents index using IndexManager
            await this.indexManager.addToSet(this.getAgentsIndexKey(swarmId), agent.id);
            
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
            
            // Remove from agents index using IndexManager
            await this.indexManager.removeFromSet(this.getAgentsIndexKey(swarmId), agentId);
            
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
            const agentIds = await this.indexManager.getSetMembers(this.getAgentsIndexKey(swarmId));
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
            
            // Add to blackboard index using IndexManager
            await this.indexManager.addToSet(this.getBlackboardIndexKey(swarmId), item.id);
            
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
            const itemIds = await this.indexManager.getSetMembers(this.getBlackboardIndexKey(swarmId));
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
            
            // Remove from blackboard index using IndexManager
            await this.indexManager.removeFromSet(this.getBlackboardIndexKey(swarmId), itemId);
            
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
     * Clean up all subsidiary data for a swarm
     */
    private async cleanupSwarmData(swarmId: string): Promise<void> {
        try {
            // Get all teams and clean them up
            const teamIds = await redis.smembers(this.getTeamsIndexKey(swarmId));
            for (const teamId of teamIds) {
                await redis.del(this.getTeamKey(swarmId, teamId));
            }
            await redis.del(this.getTeamsIndexKey(swarmId));
            
            // Get all agents and clean them up
            const agentIds = await redis.smembers(this.getAgentsIndexKey(swarmId));
            for (const agentId of agentIds) {
                await redis.del(this.getAgentKey(swarmId, agentId));
            }
            await redis.del(this.getAgentsIndexKey(swarmId));
            
            // Clean up blackboard items
            const itemIds = await redis.smembers(this.getBlackboardIndexKey(swarmId));
            for (const itemId of itemIds) {
                await redis.del(this.getBlackboardItemKey(swarmId, itemId));
            }
            await redis.del(this.getBlackboardIndexKey(swarmId));
            
            this.logger.debug("[RedisSwarmStateStore] Cleaned up swarm data", { swarmId });
        } catch (error) {
            this.logger.error("[RedisSwarmStateStore] Failed to cleanup swarm data", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private getStateIndexKey(state: ExecutionState): string {
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
        // Update state index using IndexManager
        if (swarm.state) {
            await this.indexManager.updateStateIndex(
                swarmId,
                null, // No old state for new swarms
                swarm.state,
                (s) => this.getStateIndexKey(s),
                Object.values(ExecutionStates),
            );
        }
        
        // Update user index
        if (swarm.metadata.userId) {
            await this.indexManager.addToSet(this.getUserIndexKey(swarm.metadata.userId), swarmId);
        }
    }

    private async removeFromIndexes(swarmId: string, swarm: Swarm): Promise<void> {
        // Remove from state index using IndexManager
        if (swarm.state) {
            await this.indexManager.removeFromSet(this.getStateIndexKey(swarm.state), swarmId);
        }
        
        // Remove from user index
        if (swarm.metadata.userId) {
            await this.indexManager.removeFromSet(this.getUserIndexKey(swarm.metadata.userId), swarmId);
        }
    }
}
