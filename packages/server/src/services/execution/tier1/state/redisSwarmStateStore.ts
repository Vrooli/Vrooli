import { type Logger } from "winston";
import {
    type Swarm,
    type SwarmState,
    type TeamFormation,
    SwarmStatus,
} from "@vrooli/shared";
import { redis } from "../../../../services/redisConn.js";
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