import { type Logger } from "winston";
import { type ISwarmStateStore } from "./swarmStateStore.js";
import { InMemorySwarmStateStore } from "./swarmStateStore.js";
import { RedisSwarmStateStore } from "./redisSwarmStateStore.js";

/**
 * Factory for creating swarm state store implementations
 */
export class SwarmStateStoreFactory {
    private static instance: ISwarmStateStore | null = null;

    /**
     * Gets or creates the swarm state store instance
     */
    static getInstance(logger: Logger, forceNew = false): ISwarmStateStore {
        if (!this.instance || forceNew) {
            this.instance = this.createStore(logger);
        }
        return this.instance;
    }

    /**
     * Creates a new store instance based on environment
     */
    private static createStore(logger: Logger): ISwarmStateStore {
        const useRedis = process.env.SWARM_STATE_STORE === "redis" || 
                        process.env.NODE_ENV === "production";

        if (useRedis) {
            logger.info("[SwarmStateStoreFactory] Using Redis state store");
            return new RedisSwarmStateStore(logger);
        } else {
            logger.info("[SwarmStateStoreFactory] Using in-memory state store");
            return new InMemorySwarmStateStore(logger);
        }
    }

    /**
     * Clears the current instance (useful for testing)
     */
    static clearInstance(): void {
        this.instance = null;
    }
}
