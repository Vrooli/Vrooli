import { type EmergentAgent } from "../../../../services/execution/cross-cutting/agents/emergentAgent.js";
import { logger } from "../../../../services/logger.js";

export type EvolutionStage = "reactive" | "proactive" | "predictive" | "autonomous";

export interface EvolutionMetrics {
    stage: EvolutionStage;
    responseTime: number;
    accuracy: number;
    predictiveCapability: boolean;
    autonomyLevel: number;
    learningRate: number;
    adaptationSpeed: number;
}

export interface EvolutionCriteria {
    reactive: EvolutionMetrics;
    proactive: EvolutionMetrics;
    predictive: EvolutionMetrics;
    autonomous: EvolutionMetrics;
}

export interface EvolutionResult {
    currentStage: EvolutionStage;
    metrics: EvolutionMetrics;
    timeline: EvolutionEvent[];
    transitionCount: number;
    totalDuration: number;
}

export interface EvolutionEvent {
    timestamp: Date;
    fromStage: EvolutionStage;
    toStage: EvolutionStage;
    triggerMetric: string;
    metricValue: number;
}

export class EvolutionValidator {
    private criteria: EvolutionCriteria;
    private events: EvolutionEvent[] = [];
    private startTime: number;

    constructor(criteria?: Partial<EvolutionCriteria>) {
        this.criteria = this.buildCriteria(criteria);
        this.startTime = Date.now();
    }

    /**
     * Test that an agent evolves through stages based on measurable criteria
     */
    async testEvolution(
        agent: EmergentAgent,
        targetStage: EvolutionStage,
        options: {
            timeout?: number;
            accelerationFactor?: number;
            stimuli?: EvolutionStimulus[];
        } = {},
    ): Promise<EvolutionResult> {
        const { timeout = 120000, accelerationFactor = 10, stimuli = [] } = options;
        
        const initialStage = this.detectStage(agent);
        logger.info("Starting evolution test", {
            agentId: agent.id,
            initialStage,
            targetStage,
        });

        // Set up evolution monitoring
        this.monitorEvolution(agent);

        // Run evolution simulation
        const finalMetrics = await this.runEvolutionSimulation(
            agent,
            targetStage,
            timeout,
            accelerationFactor,
            stimuli,
        );

        const currentStage = this.detectStage(agent);

        return {
            currentStage,
            metrics: finalMetrics,
            timeline: this.events,
            transitionCount: this.events.length,
            totalDuration: Date.now() - this.startTime,
        };
    }

    /**
     * Verify evolution happened naturally without forced progression
     */
    async verifyNaturalEvolution(
        agent: EmergentAgent,
        observationPeriod: number,
    ): Promise<boolean> {
        const initialMetrics = await this.captureMetrics(agent);
        const initialStage = this.stageFromMetrics(initialMetrics);

        // Observe without intervention
        await new Promise(resolve => setTimeout(resolve, observationPeriod));

        const finalMetrics = await this.captureMetrics(agent);
        const finalStage = this.stageFromMetrics(finalMetrics);

        // Check if evolution occurred naturally
        return this.getStageIndex(finalStage) > this.getStageIndex(initialStage);
    }

    /**
     * Test specific evolution path with quantifiable checkpoints
     */
    async testEvolutionPath(
        agent: EmergentAgent,
        path: EvolutionPath,
    ): Promise<EvolutionPathResult> {
        const results: StageResult[] = [];

        for (const stage of path.stages) {
            const stageResult = await this.testStageTransition(
                agent,
                stage,
                path.stimulusGenerator,
            );
            results.push(stageResult);

            if (!stageResult.achieved) {
                break; // Stop if stage not achieved
            }
        }

        return {
            path: path.name,
            completed: results.every(r => r.achieved),
            stages: results,
            totalDuration: results.reduce((sum, r) => sum + r.duration, 0),
        };
    }

    private buildCriteria(partial?: Partial<EvolutionCriteria>): EvolutionCriteria {
        const defaults: EvolutionCriteria = {
            reactive: {
                stage: "reactive",
                responseTime: 1000,
                accuracy: 0.7,
                predictiveCapability: false,
                autonomyLevel: 0.1,
                learningRate: 0.1,
                adaptationSpeed: 0.1,
            },
            proactive: {
                stage: "proactive",
                responseTime: 500,
                accuracy: 0.85,
                predictiveCapability: false,
                autonomyLevel: 0.3,
                learningRate: 0.3,
                adaptationSpeed: 0.3,
            },
            predictive: {
                stage: "predictive",
                responseTime: 200,
                accuracy: 0.95,
                predictiveCapability: true,
                autonomyLevel: 0.6,
                learningRate: 0.5,
                adaptationSpeed: 0.5,
            },
            autonomous: {
                stage: "autonomous",
                responseTime: 100,
                accuracy: 0.99,
                predictiveCapability: true,
                autonomyLevel: 0.9,
                learningRate: 0.7,
                adaptationSpeed: 0.7,
            },
        };

        return { ...defaults, ...partial };
    }

    private async captureMetrics(agent: EmergentAgent): Promise<EvolutionMetrics> {
        const performance = await agent.getPerformanceMetrics();
        
        return {
            stage: this.detectStage(agent),
            responseTime: performance.avgResponseTime || 1000,
            accuracy: performance.accuracy || 0,
            predictiveCapability: this.hasPredictiveCapability(agent),
            autonomyLevel: this.calculateAutonomyLevel(agent),
            learningRate: performance.learningRate || 0,
            adaptationSpeed: performance.adaptationSpeed || 0,
        };
    }

    private detectStage(agent: EmergentAgent): EvolutionStage {
        const metrics = agent.getCurrentMetrics();
        
        // Check stages from highest to lowest
        if (this.meetsStageMetrics(metrics, this.criteria.autonomous)) {
            return "autonomous";
        } else if (this.meetsStageMetrics(metrics, this.criteria.predictive)) {
            return "predictive";
        } else if (this.meetsStageMetrics(metrics, this.criteria.proactive)) {
            return "proactive";
        } else {
            return "reactive";
        }
    }

    private meetsStageMetrics(
        current: any,
        required: EvolutionMetrics,
    ): boolean {
        return (
            current.responseTime <= required.responseTime &&
            current.accuracy >= required.accuracy &&
            current.autonomyLevel >= required.autonomyLevel &&
            (!required.predictiveCapability || current.predictiveCapability)
        );
    }

    private stageFromMetrics(metrics: EvolutionMetrics): EvolutionStage {
        // Determine stage based on metrics
        for (const stage of ["autonomous", "predictive", "proactive", "reactive"] as EvolutionStage[]) {
            if (this.meetsStageMetrics(metrics, this.criteria[stage])) {
                return stage;
            }
        }
        return "reactive";
    }

    private getStageIndex(stage: EvolutionStage): number {
        const stages: EvolutionStage[] = ["reactive", "proactive", "predictive", "autonomous"];
        return stages.indexOf(stage);
    }

    private monitorEvolution(agent: EmergentAgent) {
        let lastStage = this.detectStage(agent);

        // Check for stage transitions periodically
        const interval = setInterval(() => {
            const currentStage = this.detectStage(agent);
            
            if (currentStage !== lastStage) {
                const metrics = agent.getCurrentMetrics();
                this.recordTransition(lastStage, currentStage, metrics);
                lastStage = currentStage;
            }
        }, 1000);

        // Clean up interval when test completes
        agent.once("test:complete", () => clearInterval(interval));
    }

    private recordTransition(
        from: EvolutionStage,
        to: EvolutionStage,
        metrics: any,
    ) {
        // Determine which metric triggered the transition
        const triggerMetric = this.identifyTriggerMetric(from, to, metrics);

        this.events.push({
            timestamp: new Date(),
            fromStage: from,
            toStage: to,
            triggerMetric: triggerMetric.name,
            metricValue: triggerMetric.value,
        });

        logger.info("Evolution stage transition", {
            from,
            to,
            trigger: triggerMetric,
        });
    }

    private identifyTriggerMetric(
        from: EvolutionStage,
        to: EvolutionStage,
        metrics: any,
    ): { name: string; value: number } {
        const fromCriteria = this.criteria[from];
        const toCriteria = this.criteria[to];

        // Find which metric changed most significantly
        const changes = {
            responseTime: Math.abs(metrics.responseTime - fromCriteria.responseTime),
            accuracy: Math.abs(metrics.accuracy - fromCriteria.accuracy),
            autonomyLevel: Math.abs(metrics.autonomyLevel - fromCriteria.autonomyLevel),
        };

        const maxChange = Object.entries(changes).reduce((max, [metric, change]) => 
            change > max.value ? { name: metric, value: change } : max,
            { name: "unknown", value: 0 },
        );

        return maxChange;
    }

    private async runEvolutionSimulation(
        agent: EmergentAgent,
        targetStage: EvolutionStage,
        timeout: number,
        accelerationFactor: number,
        stimuli: EvolutionStimulus[],
    ): Promise<EvolutionMetrics> {
        const startTime = Date.now();
        let stimulusIndex = 0;

        while (Date.now() - startTime < timeout) {
            const currentStage = this.detectStage(agent);
            
            if (this.getStageIndex(currentStage) >= this.getStageIndex(targetStage)) {
                // Target stage reached
                return await this.captureMetrics(agent);
            }

            // Apply stimuli to encourage evolution
            if (stimulusIndex < stimuli.length) {
                await this.applyStimulus(agent, stimuli[stimulusIndex]);
                stimulusIndex++;
            }

            // Simulate accelerated time
            await agent.simulateTimePassage(1000 * accelerationFactor);

            // Small delay
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        // Timeout reached
        return await this.captureMetrics(agent);
    }

    private async applyStimulus(
        agent: EmergentAgent,
        stimulus: EvolutionStimulus,
    ) {
        logger.debug("Applying evolution stimulus", {
            type: stimulus.type,
            intensity: stimulus.intensity,
        });

        switch (stimulus.type) {
            case "challenge":
                await agent.handleChallenge(stimulus.data);
                break;
            case "pattern":
                await agent.observePattern(stimulus.data);
                break;
            case "feedback":
                await agent.receiveFeedback(stimulus.data);
                break;
            case "resource":
                await agent.adjustResources(stimulus.data);
                break;
        }
    }

    private hasPredictiveCapability(agent: EmergentAgent): boolean {
        const capabilities = agent.getCapabilities();
        return capabilities.some(cap => 
            cap.includes("predict") || 
            cap.includes("forecast") || 
            cap.includes("anticipate"),
        );
    }

    private calculateAutonomyLevel(agent: EmergentAgent): number {
        const config = agent.getConfiguration();
        const metrics = agent.getCurrentMetrics();

        // Calculate based on decision-making independence
        const factors = {
            selfDirected: config.autonomy?.selfDirected ? 0.3 : 0,
            learningEnabled: config.learning?.enabled ? 0.2 : 0,
            adaptiveGoals: config.goals?.adaptive ? 0.2 : 0,
            decisionMaking: metrics.independentDecisions / (metrics.totalDecisions || 1) * 0.3,
        };

        return Object.values(factors).reduce((sum, val) => sum + val, 0);
    }

    private async testStageTransition(
        agent: EmergentAgent,
        stage: StageDefinition,
        stimulusGenerator: (stage: string) => EvolutionStimulus[],
    ): Promise<StageResult> {
        const startTime = Date.now();
        const stimuli = stimulusGenerator(stage.name);

        // Apply stimuli and test for stage achievement
        for (const stimulus of stimuli) {
            await this.applyStimulus(agent, stimulus);
            
            const metrics = await this.captureMetrics(agent);
            if (this.meetsStageMetrics(metrics, stage.requiredMetrics)) {
                return {
                    stage: stage.name,
                    achieved: true,
                    duration: Date.now() - startTime,
                    finalMetrics: metrics,
                };
            }
        }

        return {
            stage: stage.name,
            achieved: false,
            duration: Date.now() - startTime,
            finalMetrics: await this.captureMetrics(agent),
        };
    }
}

export interface EvolutionStimulus {
    type: "challenge" | "pattern" | "feedback" | "resource";
    intensity: number;
    data: any;
}

export interface EvolutionPath {
    name: string;
    stages: StageDefinition[];
    stimulusGenerator: (stage: string) => EvolutionStimulus[];
}

export interface StageDefinition {
    name: string;
    requiredMetrics: EvolutionMetrics;
    timeout: number;
}

export interface StageResult {
    stage: string;
    achieved: boolean;
    duration: number;
    finalMetrics: EvolutionMetrics;
}

export interface EvolutionPathResult {
    path: string;
    completed: boolean;
    stages: StageResult[];
    totalDuration: number;
}

/**
 * Create standard evolution stimuli for testing
 */
export function createEvolutionStimuli(
    targetStage: EvolutionStage,
): EvolutionStimulus[] {
    const stimuli: EvolutionStimulus[] = [];

    switch (targetStage) {
        case "proactive":
            stimuli.push(
                {
                    type: "pattern",
                    intensity: 0.5,
                    data: { pattern: "recurring_issue", frequency: 5 },
                },
                {
                    type: "feedback",
                    intensity: 0.7,
                    data: { metric: "response_time", target: 500 },
                },
            );
            break;

        case "predictive":
            stimuli.push(
                {
                    type: "pattern",
                    intensity: 0.8,
                    data: { pattern: "complex_sequence", predictability: 0.8 },
                },
                {
                    type: "challenge",
                    intensity: 0.9,
                    data: { type: "anticipate_failure", leadTime: 5000 },
                },
            );
            break;

        case "autonomous":
            stimuli.push(
                {
                    type: "resource",
                    intensity: 1.0,
                    data: { constraint: "minimal_guidance", autonomy: 0.9 },
                },
                {
                    type: "challenge",
                    intensity: 1.0,
                    data: { type: "self_directed_optimization", goals: ["efficiency", "accuracy"] },
                },
            );
            break;
    }

    return stimuli;
}
