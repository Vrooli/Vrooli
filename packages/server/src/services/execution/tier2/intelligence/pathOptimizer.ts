/**
 * Path Optimizer - Simple path data collection for optimization agents
 * 
 * This component collects path execution data and emits events.
 * Path optimization emerges from optimization agents analyzing patterns.
 */

import { type Logger } from "winston";
import {
    type Location,
    type Navigator,
    type StepInfo,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/eventBus.js";

/**
 * Path execution data
 */
export interface PathExecutionData {
    runId: string;
    routineId: string;
    steps: StepInfo[];
    actualPath: string[];
    duration: number;
    timestamp: Date;
}

/**
 * PathOptimizer - Event emitter for path analysis
 * 
 * Collects path execution data and emits events for optimization agents.
 * Does NOT implement optimization algorithms - those emerge from agent analysis.
 */
export class PathOptimizer {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly pathExecutions: Map<string, PathExecutionData[]> = new Map();

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
    }

    /**
     * Records path execution for analysis
     */
    async recordPathExecution(
        runId: string,
        routineId: string,
        steps: StepInfo[],
        actualPath: string[],
        duration: number,
    ): Promise<void> {
        const data: PathExecutionData = {
            runId,
            routineId,
            steps,
            actualPath,
            duration,
            timestamp: new Date(),
        };

        // Store for historical reference
        if (!this.pathExecutions.has(routineId)) {
            this.pathExecutions.set(routineId, []);
        }
        this.pathExecutions.get(routineId)!.push(data);

        // Emit path execution event for optimization agents
        await this.eventBus.publish("path.events", {
            type: "PATH_EXECUTED",
            timestamp: new Date(),
            metadata: data,
        });

        this.logger.debug("[PathOptimizer] Recorded path execution", {
            runId,
            routineId,
            duration,
            stepCount: steps.length,
        });
    }

    /**
     * Gets historical path executions for a routine
     */
    getPathHistory(routineId: string): PathExecutionData[] {
        return this.pathExecutions.get(routineId) || [];
    }

    /**
     * Emits path analysis request for agents
     */
    async requestPathAnalysis(
        routineId: string,
        currentSteps: StepInfo[],
    ): Promise<void> {
        await this.eventBus.publish("path.events", {
            type: "PATH_ANALYSIS_REQUEST",
            timestamp: new Date(),
            metadata: {
                routineId,
                currentSteps,
                history: this.getPathHistory(routineId),
            },
        });
    }
}