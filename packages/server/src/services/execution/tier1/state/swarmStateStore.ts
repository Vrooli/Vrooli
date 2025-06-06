import { type Logger } from "winston";
import {
    type Swarm,
    type SwarmState,
    type SwarmConfig,
    type TeamFormation,
} from "@vrooli/shared";

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
    
    // Query operations
    listActiveSwarms(): Promise<string[]>;
    getSwarmsByState(state: SwarmState): Promise<string[]>;
    getSwarmsByUser(userId: string): Promise<string[]>;
}

/**
 * In-memory swarm state store (for development)
 */
export class InMemorySwarmStateStore implements ISwarmStateStore {
    private readonly swarms: Map<string, Swarm> = new Map();
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    async createSwarm(swarmId: string, swarm: Swarm): Promise<void> {
        this.swarms.set(swarmId, swarm);
        this.logger.debug("[SwarmStateStore] Created swarm", { swarmId });
    }

    async getSwarm(swarmId: string): Promise<Swarm | null> {
        return this.swarms.get(swarmId) || null;
    }

    async updateSwarm(swarmId: string, updates: Partial<Swarm>): Promise<void> {
        const swarm = await this.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }
        
        Object.assign(swarm, updates, { updatedAt: new Date() });
        this.logger.debug("[SwarmStateStore] Updated swarm", { swarmId });
    }

    async deleteSwarm(swarmId: string): Promise<void> {
        this.swarms.delete(swarmId);
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
        for (const [id, swarm] of this.swarms) {
            if (![SwarmState.STOPPED, SwarmState.FAILED, SwarmState.TERMINATED].includes(swarm.state)) {
                active.push(id);
            }
        }
        return active;
    }

    async getSwarmsByState(state: SwarmState): Promise<string[]> {
        const matches: string[] = [];
        for (const [id, swarm] of this.swarms) {
            if (swarm.state === state) {
                matches.push(id);
            }
        }
        return matches;
    }

    async getSwarmsByUser(userId: string): Promise<string[]> {
        const matches: string[] = [];
        for (const [id, swarm] of this.swarms) {
            if (swarm.metadata.userId === userId) {
                matches.push(id);
            }
        }
        return matches;
    }
}