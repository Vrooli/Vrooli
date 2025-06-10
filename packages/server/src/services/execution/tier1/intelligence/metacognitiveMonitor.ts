import { type Logger } from "winston";
import {
    type SwarmProgress,
    type SwarmResources,
    type SwarmDecision,
    type MetacognitiveReflection,
    type SwarmPerformance,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/events/eventBus.js";

/**
 * Performance analysis input
 */
export interface PerformanceAnalysisInput {
    swarmId: string;
    decisions: SwarmDecision[];
    progress: SwarmProgress;
    resources: SwarmResources;
}

/**
 * MetacognitiveMonitor - Event emitter for metacognitive analysis
 * 
 * This component simply collects swarm execution data and emits events
 * for metacognitive agents to analyze. All intelligence emerges from
 * agent analysis, not from hard-coded logic.
 * 
 * Emits events for:
 * - Performance snapshots
 * - Decision outcomes
 * - Resource usage patterns
 * - Progress updates
 */
export class MetacognitiveMonitor {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly performanceSnapshots: Map<string, SwarmPerformance[]> = new Map();
    private readonly decisionOutcomes: Map<string, Map<string, "success" | "failure" | "pending">> = new Map();

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.subscribeToEvents();
    }

    /**
     * Collects performance data and emits events for agent analysis
     */
    async collectPerformanceData(
        input: PerformanceAnalysisInput,
    ): Promise<void> {
        this.logger.debug("[MetacognitiveMonitor] Collecting performance data", {
            swarmId: input.swarmId,
            decisionCount: input.decisions.length,
        });

        // Emit raw performance data for agents to analyze
        await this.eventBus.publish("swarm.events", {
            type: "PERFORMANCE_SNAPSHOT",
            swarmId: input.swarmId,
            timestamp: new Date(),
            metadata: {
                decisions: input.decisions,
                progress: input.progress,
                resources: input.resources,
                decisionOutcomes: this.getDecisionOutcomes(input.swarmId),
            },
        });

        // Store snapshot for historical reference
        this.storePerformanceSnapshot(input.swarmId, {
            goalProgress: input.progress.tasksCompleted / Math.max(input.progress.tasksTotal, 1),
            resourceEfficiency: input.resources.totalBudget > 0 
                ? 1 - (input.resources.usedBudget / input.resources.totalBudget) 
                : 0,
            teamCohesion: 0, // Will be calculated by agents
            adaptationRate: 0, // Will be calculated by agents
            learningRate: 0, // Will be calculated by agents
            overallEffectiveness: 0, // Will be calculated by agents
        });
    }

    /**
     * Gets stored performance snapshots for agent analysis
     */
    async getPerformanceSnapshots(swarmId: string): Promise<SwarmPerformance[]> {
        return this.performanceSnapshots.get(swarmId) || [];
    }

    /**
     * Records decision outcome and emits event
     */
    async recordDecisionOutcome(
        swarmId: string,
        decisionId: string,
        outcome: "success" | "failure",
    ): Promise<void> {
        if (!this.decisionOutcomes.has(swarmId)) {
            this.decisionOutcomes.set(swarmId, new Map());
        }
        
        this.decisionOutcomes.get(swarmId)!.set(decisionId, outcome);
        
        this.logger.debug("[MetacognitiveMonitor] Recorded decision outcome", {
            swarmId,
            decisionId,
            outcome,
        });

        // Emit outcome event for agents to analyze patterns
        await this.eventBus.publish("swarm.events", {
            type: "DECISION_OUTCOME",
            swarmId,
            timestamp: new Date(),
            metadata: {
                decisionId,
                outcome,
                allOutcomes: this.getDecisionOutcomes(swarmId),
            },
        });
    }

    /**
     * Private helper methods
     */
    private getDecisionOutcomes(swarmId: string): Record<string, "success" | "failure" | "pending"> {
        const outcomes = this.decisionOutcomes.get(swarmId) || new Map();
        return Object.fromEntries(outcomes);
    }

    private storePerformanceSnapshot(swarmId: string, performance: SwarmPerformance): void {
        if (!this.performanceSnapshots.has(swarmId)) {
            this.performanceSnapshots.set(swarmId, []);
        }
        
        const snapshots = this.performanceSnapshots.get(swarmId)!;
        snapshots.push(performance);
        
        // Keep only recent snapshots
        if (snapshots.length > 100) {
            snapshots.splice(0, snapshots.length - 100);
        }
    }

    private subscribeToEvents(): void {
        // Subscribe to decision outcomes
        this.eventBus.subscribe("swarm.decision.outcome", async (event) => {
            await this.recordDecisionOutcome(
                event.swarmId,
                event.decisionId,
                event.outcome,
            );
        });
    }

    /**
     * Stops the metacognitive monitor
     */
    async stop(): Promise<void> {
        this.logger.info("[MetacognitiveMonitor] Stopped");
    }
}