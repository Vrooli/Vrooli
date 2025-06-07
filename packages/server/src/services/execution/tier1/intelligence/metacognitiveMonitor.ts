import { type Logger } from "winston";
import {
    type SwarmProgress,
    type SwarmResources,
    type SwarmDecision,
    type MetacognitiveReflection,
    type SwarmPerformance,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/eventBus.js";

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
 * Metacognitive metrics
 */
export interface MetacognitiveMetrics {
    decisionQuality: number;
    adaptationRate: number;
    learningEfficiency: number;
    predictionAccuracy: number;
    selfAwarenessLevel: number;
}

/**
 * Cognitive bias detection
 */
export interface CognitiveBias {
    type: "confirmation" | "anchoring" | "availability" | "overconfidence" | "groupthink";
    severity: "low" | "medium" | "high";
    description: string;
    mitigation: string;
}

/**
 * Adaptation recommendation
 */
export interface AdaptationRecommendation {
    type: "strategy" | "team" | "resource" | "goal" | "process";
    action: string;
    rationale: string;
    urgency: "low" | "medium" | "high";
    expectedImprovement: number;
}

/**
 * MetacognitiveMonitor - Self-awareness and reflection for swarm intelligence
 * 
 * This component implements metacognitive capabilities that allow the swarm
 * to reason about its own reasoning. It provides:
 * 
 * - Performance self-assessment
 * - Cognitive bias detection
 * - Learning efficiency tracking
 * - Strategy effectiveness evaluation
 * - Adaptive behavior recommendations
 * 
 * The monitor enables the swarm to improve its decision-making processes
 * over time through continuous self-reflection and adaptation.
 */
export class MetacognitiveMonitor {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly performanceHistory: Map<string, SwarmPerformance[]> = new Map();
    private readonly decisionOutcomes: Map<string, Map<string, "success" | "failure" | "pending">> = new Map();
    private readonly biasDetectionThreshold: number = 0.7;
    private readonly reflectionInterval: number = 300000; // 5 minutes
    private reflectionTimer?: NodeJS.Timer;
    
    // Configuration constants  
    private readonly GOAL_WEIGHT = 0.4;
    private readonly RESOURCE_WEIGHT = 0.2;
    private readonly TEAM_WEIGHT = 0.2;
    private readonly ADAPTATION_WEIGHT = 0.1;
    private readonly LEARNING_WEIGHT = 0.1;
    private readonly VARIETY_THRESHOLD = 0.3;
    private readonly FAILURE_RATE_THRESHOLD = 0.3;

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.startReflectionLoop();
        this.subscribeToEvents();
    }

    /**
     * Analyzes swarm performance and generates insights
     */
    async analyzePerformance(
        input: PerformanceAnalysisInput,
    ): Promise<MetacognitiveReflection> {
        this.logger.debug("[MetacognitiveMonitor] Analyzing performance", {
            swarmId: input.swarmId,
            decisionCount: input.decisions.length,
        });

        // Calculate performance metrics
        const performance = await this.calculatePerformanceMetrics(input);
        
        // Detect cognitive biases
        const biases = await this.detectCognitiveBiases(input);
        
        // Generate learnings
        const learnings = await this.extractLearnings(input, performance, biases);
        
        // Generate adaptation recommendations
        const adaptations = await this.generateAdaptations(input, performance, biases);
        
        // Calculate confidence
        const confidence = this.calculateConfidence(performance, biases);

        const reflection: MetacognitiveReflection = {
            swarmId: input.swarmId,
            timestamp: new Date(),
            performance,
            learnings,
            adaptations: adaptations.map(a => a.action),
            confidence,
        };

        // Store performance history
        this.updatePerformanceHistory(input.swarmId, performance);

        // Emit reflection event
        await this.eventBus.publish("swarm.events", {
            type: "METACOGNITIVE_REFLECTION",
            swarmId: input.swarmId,
            timestamp: new Date(),
            metadata: { reflection },
        });

        return reflection;
    }

    /**
     * Gets performance metrics for a swarm
     */
    async getPerformanceMetrics(swarmId: string): Promise<MetacognitiveMetrics> {
        const history = this.performanceHistory.get(swarmId) || [];
        
        if (history.length === 0) {
            return {
                decisionQuality: 0.5,
                adaptationRate: 0.5,
                learningEfficiency: 0.5,
                predictionAccuracy: 0.5,
                selfAwarenessLevel: 0.5,
            };
        }

        // Calculate averages from recent history
        const recentHistory = history.slice(-10); // Last 10 entries
        const avgPerformance = this.averagePerformance(recentHistory);
        
        // Calculate metacognitive metrics
        const outcomes = this.decisionOutcomes.get(swarmId) || new Map();
        const successRate = this.calculateSuccessRate(outcomes);
        
        return {
            decisionQuality: successRate,
            adaptationRate: avgPerformance.adaptationRate,
            learningEfficiency: avgPerformance.learningRate,
            predictionAccuracy: this.calculatePredictionAccuracy(swarmId),
            selfAwarenessLevel: this.calculateSelfAwareness(swarmId),
        };
    }

    /**
     * Records decision outcome
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

        // Trigger immediate reflection if significant failure
        if (outcome === "failure") {
            const failures = Array.from(this.decisionOutcomes.get(swarmId)!.values())
                .filter(o => o === "failure").length;
            
            if (failures > 3) {
                this.logger.warn("[MetacognitiveMonitor] Multiple failures detected", {
                    swarmId,
                    failureCount: failures,
                });
            }
        }
    }

    /**
     * Private helper methods
     */
    private async calculatePerformanceMetrics(
        input: PerformanceAnalysisInput,
    ): Promise<SwarmPerformance> {
        const { progress, resources, decisions } = input;
        
        // Goal progress
        const goalProgress = progress.tasksCompleted / Math.max(progress.tasksTotal, 1);
        
        // Resource efficiency
        const resourceEfficiency = resources.totalBudget > 0 ?
            1 - (resources.usedBudget / resources.totalBudget) : 0;
        
        // Team cohesion (simplified)
        const teamCohesion = 0.8; // Would be calculated from team interactions
        
        // Adaptation rate
        const adaptationRate = this.calculateAdaptationRate(decisions);
        
        // Learning rate
        const learningRate = this.calculateLearningRate(input.swarmId);
        
        // Overall effectiveness
        const overallEffectiveness = (
            goalProgress * this.GOAL_WEIGHT +
            resourceEfficiency * this.RESOURCE_WEIGHT +
            teamCohesion * this.TEAM_WEIGHT +
            adaptationRate * this.ADAPTATION_WEIGHT +
            learningRate * this.LEARNING_WEIGHT
        );
        
        return {
            goalProgress,
            resourceEfficiency,
            teamCohesion,
            adaptationRate,
            learningRate,
            overallEffectiveness,
        };
    }

    private async detectCognitiveBiases(
        input: PerformanceAnalysisInput,
    ): Promise<CognitiveBias[]> {
        const biases: CognitiveBias[] = [];
        const { decisions } = input;
        
        // Confirmation bias - tendency to stick with initial strategies
        const uniqueDecisionTypes = new Set(decisions.map(d => d.decision.split("(")[0]));
        if (uniqueDecisionTypes.size < decisions.length * this.VARIETY_THRESHOLD) {
            biases.push({
                type: "confirmation",
                severity: "medium",
                description: "Limited variety in decision types suggests confirmation bias",
                mitigation: "Actively explore alternative strategies",
            });
        }
        
        // Overconfidence bias - high confidence despite failures
        const outcomes = this.decisionOutcomes.get(input.swarmId) || new Map();
        const failureRate = 1 - this.calculateSuccessRate(outcomes);
        const MIN_DECISIONS_FOR_BIAS_CHECK = 10;
        if (failureRate > this.FAILURE_RATE_THRESHOLD && decisions.length > MIN_DECISIONS_FOR_BIAS_CHECK) {
            biases.push({
                type: "overconfidence",
                severity: "high",
                description: "High failure rate suggests overconfidence in decisions",
                mitigation: "Increase decision validation and testing",
            });
        }
        
        // Anchoring bias - over-reliance on early information
        if (decisions.length > 5) {
            const earlyDecisions = decisions.slice(0, 3).map(d => d.decision);
            const recentDecisions = decisions.slice(-3).map(d => d.decision);
            const similarity = this.calculateSimilarity(earlyDecisions, recentDecisions);
            
            if (similarity > this.biasDetectionThreshold) {
                biases.push({
                    type: "anchoring",
                    severity: "low",
                    description: "Decisions show anchoring to early choices",
                    mitigation: "Re-evaluate assumptions and constraints",
                });
            }
        }
        
        return biases;
    }

    private async extractLearnings(
        input: PerformanceAnalysisInput,
        performance: SwarmPerformance,
        biases: CognitiveBias[],
    ): Promise<string[]> {
        const learnings: string[] = [];
        
        // Performance-based learnings
        if (performance.goalProgress > 0.8) {
            learnings.push("Current strategies are effective for goal achievement");
        } else if (performance.goalProgress < 0.3) {
            learnings.push("Goal progress is slower than expected - strategy revision needed");
        }
        
        if (performance.resourceEfficiency < 0.5) {
            learnings.push("Resource consumption is high - optimization opportunities exist");
        }
        
        // Bias-based learnings
        for (const bias of biases) {
            if (bias.severity === "high") {
                learnings.push(`High ${bias.type} bias detected - ${bias.mitigation}`);
            }
        }
        
        // Decision pattern learnings
        const decisionPatterns = this.analyzeDecisionPatterns(input.decisions);
        learnings.push(...decisionPatterns);
        
        return learnings;
    }

    private async generateAdaptations(
        input: PerformanceAnalysisInput,
        performance: SwarmPerformance,
        biases: CognitiveBias[],
    ): Promise<AdaptationRecommendation[]> {
        const adaptations: AdaptationRecommendation[] = [];
        
        // Performance-based adaptations
        if (performance.goalProgress < 0.5 && input.progress.tasksCompleted > 5) {
            adaptations.push({
                type: "strategy",
                action: "adapt_strategy(accelerate_execution)",
                rationale: "Goal progress is below expected rate",
                urgency: "high",
                expectedImprovement: 0.3,
            });
        }
        
        if (performance.resourceEfficiency < 0.3) {
            adaptations.push({
                type: "resource",
                action: "optimize_resource_allocation",
                rationale: "Resource efficiency is critically low",
                urgency: "high",
                expectedImprovement: 0.4,
            });
        }
        
        // Bias-based adaptations
        for (const bias of biases) {
            if (bias.severity === "high" || bias.severity === "medium") {
                adaptations.push({
                    type: "process",
                    action: `mitigate_bias(${bias.type})`,
                    rationale: bias.description,
                    urgency: bias.severity === "high" ? "high" : "medium",
                    expectedImprovement: 0.2,
                });
            }
        }
        
        // Learning-based adaptations
        if (performance.learningRate < 0.3) {
            adaptations.push({
                type: "process",
                action: "increase_exploration",
                rationale: "Learning rate is low - need more experimentation",
                urgency: "medium",
                expectedImprovement: 0.25,
            });
        }
        
        return adaptations;
    }

    private calculateConfidence(
        performance: SwarmPerformance,
        biases: CognitiveBias[],
    ): number {
        // Base confidence on performance
        let confidence = performance.overallEffectiveness;
        
        // Reduce confidence based on biases
        for (const bias of biases) {
            const penalty = {
                low: 0.05,
                medium: 0.1,
                high: 0.2,
            }[bias.severity];
            confidence = Math.max(0, confidence - penalty);
        }
        
        // Ensure reasonable bounds
        return Math.max(0.1, Math.min(0.9, confidence));
    }

    private calculateAdaptationRate(decisions: SwarmDecision[]): number {
        if (decisions.length < 2) return 0.5;
        
        // Count strategy changes
        let changes = 0;
        for (let i = 1; i < decisions.length; i++) {
            const prevType = decisions[i - 1].decision.split("(")[0];
            const currType = decisions[i].decision.split("(")[0];
            if (prevType !== currType) {
                changes++;
            }
        }
        
        return Math.min(1, changes / (decisions.length - 1));
    }

    private calculateLearningRate(swarmId: string): number {
        const outcomes = this.decisionOutcomes.get(swarmId) || new Map();
        const outcomeArray = Array.from(outcomes.entries())
            .sort((a, b) => a[0].localeCompare(b[0]));
        
        if (outcomeArray.length < 5) return 0.5;
        
        // Calculate improvement over time
        const windowSize = 5;
        const windows = Math.floor(outcomeArray.length / windowSize);
        
        if (windows < 2) return 0.5;
        
        let improvementSum = 0;
        for (let i = 1; i < windows; i++) {
            const prevWindow = outcomeArray.slice((i - 1) * windowSize, i * windowSize);
            const currWindow = outcomeArray.slice(i * windowSize, (i + 1) * windowSize);
            
            const prevSuccess = prevWindow.filter(([_, o]) => o === "success").length / windowSize;
            const currSuccess = currWindow.filter(([_, o]) => o === "success").length / windowSize;
            
            improvementSum += Math.max(0, currSuccess - prevSuccess);
        }
        
        return Math.min(1, improvementSum / (windows - 1));
    }

    private calculateSuccessRate(outcomes: Map<string, "success" | "failure" | "pending">): number {
        const completed = Array.from(outcomes.values())
            .filter(o => o !== "pending");
        
        if (completed.length === 0) return 0.5;
        
        const successes = completed.filter(o => o === "success").length;
        return successes / completed.length;
    }

    private calculatePredictionAccuracy(swarmId: string): number {
        // Simplified - would compare predicted vs actual outcomes
        const history = this.performanceHistory.get(swarmId) || [];
        if (history.length < 2) return 0.5;
        
        // Check if performance improved as predicted
        let accuratePredictions = 0;
        for (let i = 1; i < history.length; i++) {
            if (history[i].overallEffectiveness > history[i - 1].overallEffectiveness) {
                accuratePredictions++;
            }
        }
        
        return accuratePredictions / (history.length - 1);
    }

    private calculateSelfAwareness(swarmId: string): number {
        // Based on accuracy of self-assessments
        const history = this.performanceHistory.get(swarmId) || [];
        
        if (history.length === 0) return 0.5;
        
        // Simplified self-awareness calculation
        const latestPerformance = history[history.length - 1];
        return latestPerformance.overallEffectiveness;
    }

    private calculateSimilarity(set1: string[], set2: string[]): number {
        const types1 = set1.map(s => s.split("(")[0]);
        const types2 = set2.map(s => s.split("(")[0]);
        
        let matches = 0;
        for (const type of types1) {
            if (types2.includes(type)) {
                matches++;
            }
        }
        
        return matches / Math.max(types1.length, types2.length);
    }

    private analyzeDecisionPatterns(decisions: SwarmDecision[]): string[] {
        const patterns: string[] = [];
        
        // Analyze decision frequency
        const frequency = new Map<string, number>();
        for (const decision of decisions) {
            const type = decision.decision.split("(")[0];
            frequency.set(type, (frequency.get(type) || 0) + 1);
        }
        
        // Find dominant patterns
        const total = decisions.length;
        for (const [type, count] of frequency) {
            const RELIANCE_THRESHOLD = 0.4;
            const PERCENTAGE_MULTIPLIER = 100;
            if (count / total > RELIANCE_THRESHOLD) {
                const percentage = Math.round(count / total * PERCENTAGE_MULTIPLIER);
                patterns.push("Heavy reliance on " + type + " decisions (" + percentage + "%)");
            }
        }
        
        return patterns;
    }

    private updatePerformanceHistory(
        swarmId: string,
        performance: SwarmPerformance,
    ): void {
        if (!this.performanceHistory.has(swarmId)) {
            this.performanceHistory.set(swarmId, []);
        }
        
        const history = this.performanceHistory.get(swarmId)!;
        history.push(performance);
        
        // Keep only recent history
        if (history.length > 100) {
            history.splice(0, history.length - 100);
        }
    }

    private averagePerformance(performances: SwarmPerformance[]): SwarmPerformance {
        if (performances.length === 0) {
            return {
                goalProgress: 0,
                resourceEfficiency: 0,
                teamCohesion: 0,
                adaptationRate: 0,
                learningRate: 0,
                overallEffectiveness: 0,
            };
        }
        
        const sum = performances.reduce((acc, p) => ({
            goalProgress: acc.goalProgress + p.goalProgress,
            resourceEfficiency: acc.resourceEfficiency + p.resourceEfficiency,
            teamCohesion: acc.teamCohesion + p.teamCohesion,
            adaptationRate: acc.adaptationRate + p.adaptationRate,
            learningRate: acc.learningRate + p.learningRate,
            overallEffectiveness: acc.overallEffectiveness + p.overallEffectiveness,
        }));
        
        const count = performances.length;
        return {
            goalProgress: sum.goalProgress / count,
            resourceEfficiency: sum.resourceEfficiency / count,
            teamCohesion: sum.teamCohesion / count,
            adaptationRate: sum.adaptationRate / count,
            learningRate: sum.learningRate / count,
            overallEffectiveness: sum.overallEffectiveness / count,
        };
    }

    private startReflectionLoop(): void {
        this.reflectionTimer = setInterval(async () => {
            try {
                await this.performPeriodicReflection();
            } catch (error) {
                this.logger.error("[MetacognitiveMonitor] Reflection error", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }, this.reflectionInterval);
    }

    private async performPeriodicReflection(): Promise<void> {
        // Perform reflection for all active swarms
        for (const swarmId of this.performanceHistory.keys()) {
            const metrics = await this.getPerformanceMetrics(swarmId);
            
            // Alert on concerning metrics
            if (metrics.decisionQuality < 0.3) {
                this.logger.warn("[MetacognitiveMonitor] Low decision quality detected", {
                    swarmId,
                    quality: metrics.decisionQuality,
                });
            }
            
            if (metrics.selfAwarenessLevel < 0.3) {
                this.logger.warn("[MetacognitiveMonitor] Low self-awareness detected", {
                    swarmId,
                    awareness: metrics.selfAwarenessLevel,
                });
            }
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
        if (this.reflectionTimer) {
            clearInterval(this.reflectionTimer);
            this.reflectionTimer = undefined;
        }
        
        this.logger.info("[MetacognitiveMonitor] Stopped");
    }
}