import { logger } from "../../../events/logger.js";
import { SwarmSocketEmitter } from "../../swarmSocketEmitter.js";
import { SwarmStateMapper } from "../../swarmStateMapper.js";
import { type ISwarmStateStore } from "../tier1/state/swarmStateStore.js";
import { type Swarm, type SwarmConfiguration, ExecutionState } from "@vrooli/shared";

/**
 * Helper class for emitting socket events from execution tiers.
 * Handles chatId resolution and provides a clean interface for tiers.
 */
export class ExecutionSocketEventEmitter {
    private static instance: ExecutionSocketEventEmitter;
    private swarmEmitter = SwarmSocketEmitter.get();
    private chatIdCache = new Map<string, string>(); // swarmId → chatId
    private stateStore: ISwarmStateStore | null = null;
    
    // Monitoring metrics
    private metrics = {
        totalEmissions: 0,
        emissionsByType: {
            stateUpdate: 0,
            configUpdate: 0,
            resourceUpdate: 0,
            teamUpdate: 0,
        },
        cacheHits: 0,
        cacheMisses: 0,
        errors: 0,
        lastEmissionTime: 0,
    };

    static get(): ExecutionSocketEventEmitter {
        if (!ExecutionSocketEventEmitter.instance) {
            ExecutionSocketEventEmitter.instance = new ExecutionSocketEventEmitter();
        }
        return ExecutionSocketEventEmitter.instance;
    }

    /**
     * Sets the state store for chatId lookups
     */
    setStateStore(stateStore: ISwarmStateStore): void {
        this.stateStore = stateStore;
    }

    /**
     * Resolves chatId for a given swarmId, using cache when possible
     */
    async getChatIdForSwarm(swarmId: string): Promise<string | null> {
        try {
            // Check cache first
            if (this.chatIdCache.has(swarmId)) {
                this.metrics.cacheHits++;
                return this.chatIdCache.get(swarmId)!;
            }
            
            this.metrics.cacheMisses++;

            // Query state store
            if (!this.stateStore) {
                logger.error("State store not set for socket event emitter", { swarmId });
                return null;
            }

            const swarm = await this.stateStore.getSwarm(swarmId);
            if (!swarm) {
                logger.warn("Swarm not found for socket event", { swarmId });
                return null;
            }

            // ChatId is stored as conversationId in metadata
            const chatId = swarm.metadata?.conversationId as string | undefined;
            if (!chatId) {
                logger.warn("No conversationId found in swarm metadata", { swarmId });
                return null;
            }

            // Cache for future lookups
            this.chatIdCache.set(swarmId, chatId);
            return chatId;
        } catch (error) {
            this.metrics.errors++;
            logger.error("Failed to resolve chatId for swarm", { error, swarmId });
            return null;
        }
    }

    /**
     * Emits a swarm state update event
     */
    async emitSwarmStateUpdate(
        swarmId: string,
        state: ExecutionState,
        message?: string,
        chatIdOverride?: string,
    ): Promise<void> {
        try {
            const chatId = chatIdOverride || await this.getChatIdForSwarm(swarmId);
            if (!chatId) return;

            this.swarmEmitter.emitSwarmStateUpdate(chatId, swarmId, state, message);
            
            this.metrics.totalEmissions++;
            this.metrics.emissionsByType.stateUpdate++;
            this.metrics.lastEmissionTime = Date.now();
        } catch (error) {
            this.metrics.errors++;
            logger.error("Failed to emit swarm state update", { error, swarmId, state });
        }
    }

    /**
     * Emits a swarm resource update event
     */
    async emitSwarmResourceUpdate(
        swarmId: string,
        resources: { allocated: number; consumed: number; remaining: number },
        chatIdOverride?: string,
    ): Promise<void> {
        try {
            const chatId = chatIdOverride || await this.getChatIdForSwarm(swarmId);
            if (!chatId) return;

            this.swarmEmitter.emitSwarmResourceUpdate(chatId, swarmId, { resources } as any);
            
            this.metrics.totalEmissions++;
            this.metrics.emissionsByType.resourceUpdate++;
            this.metrics.lastEmissionTime = Date.now();
        } catch (error) {
            this.metrics.errors++;
            logger.error("Failed to emit swarm resource update", { error, swarmId });
        }
    }

    /**
     * Emits a swarm config update event
     */
    async emitSwarmConfigUpdate(
        swarmId: string,
        configUpdate: any,
        chatIdOverride?: string,
    ): Promise<void> {
        try {
            const chatId = chatIdOverride || await this.getChatIdForSwarm(swarmId);
            if (!chatId) return;

            this.swarmEmitter.emitSwarmConfigUpdate(chatId, configUpdate);
            
            this.metrics.totalEmissions++;
            this.metrics.emissionsByType.configUpdate++;
            this.metrics.lastEmissionTime = Date.now();
        } catch (error) {
            this.metrics.errors++;
            logger.error("Failed to emit swarm config update", { error, swarmId });
        }
    }

    /**
     * Emits a swarm team update event
     */
    async emitSwarmTeamUpdate(
        swarmId: string,
        update: {
            teamId?: string;
            swarmLeader?: string;
            subtaskLeaders?: Record<string, string>;
        },
        chatIdOverride?: string,
    ): Promise<void> {
        try {
            const chatId = chatIdOverride || await this.getChatIdForSwarm(swarmId);
            if (!chatId) return;

            this.swarmEmitter.emitSwarmTeamUpdate(
                chatId,
                swarmId,
                update.teamId,
                update.swarmLeader,
                update.subtaskLeaders,
            );
            
            this.metrics.totalEmissions++;
            this.metrics.emissionsByType.teamUpdate++;
            this.metrics.lastEmissionTime = Date.now();
        } catch (error) {
            this.metrics.errors++;
            logger.error("Failed to emit swarm team update", { error, swarmId });
        }
    }

    /**
     * Convenience method to emit all relevant updates after a swarm state change
     */
    async emitFullSwarmUpdate(
        swarmConfig: SwarmConfiguration,
        swarm: Swarm,
        chatIdOverride?: string,
    ): Promise<void> {
        const chatId = chatIdOverride || swarmConfig.chatConfig.__typename === "ChatConfigObject" 
            ? swarmConfig.metadata?.conversationId as string
            : await this.getChatIdForSwarm(swarm.id);
            
        if (!chatId) return;

        this.swarmEmitter.emitFullSwarmUpdate(chatId, swarmConfig, swarm);
    }

    /**
     * Clears the chatId cache (useful for testing or memory management)
     */
    clearCache(): void {
        this.chatIdCache.clear();
    }

    /**
     * Gets comprehensive statistics for monitoring
     */
    getCacheStats(): { 
        size: number; 
        hitRate: number;
        totalEmissions: number;
        emissionsByType: typeof this.metrics.emissionsByType;
        errors: number;
        lastEmissionTime: number;
        cacheEfficiency: number;
    } {
        const totalLookups = this.metrics.cacheHits + this.metrics.cacheMisses;
        const hitRate = totalLookups > 0 ? (this.metrics.cacheHits / totalLookups) : 0;
        const cacheEfficiency = totalLookups > 0 ? (this.metrics.cacheHits / totalLookups) * 100 : 0;
        
        return {
            size: this.chatIdCache.size,
            hitRate,
            totalEmissions: this.metrics.totalEmissions,
            emissionsByType: { ...this.metrics.emissionsByType },
            errors: this.metrics.errors,
            lastEmissionTime: this.metrics.lastEmissionTime,
            cacheEfficiency,
        };
    }

    /**
     * Gets performance metrics for monitoring
     */
    getMetrics() {
        return {
            ...this.getCacheStats(),
            uptime: Date.now() - (this.metrics.lastEmissionTime || Date.now()),
            averageEmissionsPerMinute: this.calculateEmissionRate(),
        };
    }

    /**
     * Calculates emission rate per minute
     */
    private calculateEmissionRate(): number {
        if (this.metrics.totalEmissions === 0 || this.metrics.lastEmissionTime === 0) {
            return 0;
        }
        
        const uptimeMinutes = (Date.now() - this.metrics.lastEmissionTime) / (1000 * 60);
        return uptimeMinutes > 0 ? this.metrics.totalEmissions / uptimeMinutes : 0;
    }
}