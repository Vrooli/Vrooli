/**
 * Performance Benchmarking Framework (Phase 4 Improvement)
 * 
 * Provides realistic performance measurement and evolution validation for execution fixtures.
 * Validates that system improvements are measurable and that evolution pathways actually
 * result in better performance over time.
 * 
 * This complements structural and behavioral validation with quantitative performance analysis.
 */

import { describe, it, expect } from "vitest";
import type { 
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject,
} from "@vrooli/shared";
import type { 
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    ValidationResult,
} from "./executionValidationUtils.js";
import type { 
    ExecutionResult,
    ExecutionOptions,
    EvolutionExecutionResult,
} from "./executionRunner.js";
import { ExecutionFixtureRunner } from "./executionRunner.js";

// ================================================================================================
// Performance Benchmarking Types
// ================================================================================================

export interface BenchmarkConfig {
    /** Number of iterations to run for statistical significance */
    iterations: number;
    /** Input data for benchmarking */
    input: any;
    /** Expected output pattern for accuracy calculation */
    expectedOutput?: any;
    /** Performance targets */
    targets?: PerformanceTargets;
    /** Benchmark environment settings */
    environment?: {
        warmupIterations?: number;
        cooldownMs?: number;
        maxConcurrency?: number;
        resourceLimits?: {
            maxMemoryMB?: number;
            maxCpuPercent?: number;
            maxTokens?: number;
        };
    };
}

export interface PerformanceTargets {
    /** Maximum acceptable latency in milliseconds */
    maxLatencyMs?: number;
    /** Minimum required accuracy (0-1) */
    minAccuracy?: number;
    /** Maximum cost per execution */
    maxCost?: number;
    /** Maximum memory usage in MB */
    maxMemoryMB?: number;
    /** Required availability (0-1) */
    minAvailability?: number;
    /** Throughput targets */
    minThroughputPerSecond?: number;
}

export interface BenchmarkResult {
    /** Statistical metrics across all iterations */
    metrics: PerformanceMetrics;
    /** Individual iteration results */
    iterations: IterationResult[];
    /** Performance targets validation */
    targetsValidation: TargetsValidationResult;
    /** Detected performance characteristics */
    characteristics: PerformanceCharacteristics;
    /** Recommendations for improvement */
    recommendations: PerformanceRecommendation[];
}

export interface PerformanceMetrics {
    latency: StatisticalMetrics;
    accuracy: StatisticalMetrics;
    cost: StatisticalMetrics;
    memoryUsage: StatisticalMetrics;
    cpuUsage: StatisticalMetrics;
    tokenUsage: StatisticalMetrics;
    throughput: number; // executions per second
    availability: number; // percentage of successful executions
    reliability: number; // consistency of performance
}

export interface StatisticalMetrics {
    mean: number;
    median: number;
    p95: number;
    p99: number;
    min: number;
    max: number;
    stdDev: number;
}

export interface IterationResult {
    iteration: number;
    executionResult: ExecutionResult;
    timestamp: number;
    success: boolean;
}

export interface TargetsValidationResult {
    allTargetsMet: boolean;
    failedTargets: string[];
    marginsByTarget: Record<string, number>; // How much over/under target
}

export interface PerformanceCharacteristics {
    scalabilityPattern: "linear" | "logarithmic" | "constant" | "degrading";
    performanceStability: "stable" | "variable" | "degrading" | "improving";
    resourceEfficiency: "efficient" | "moderate" | "inefficient";
    errorHandlingImpact: "minimal" | "moderate" | "significant";
}

export interface PerformanceRecommendation {
    category: "latency" | "accuracy" | "cost" | "memory" | "reliability";
    priority: "low" | "medium" | "high" | "critical";
    description: string;
    potentialImprovement: string;
    implementationComplexity: "low" | "medium" | "high";
}

// ================================================================================================
// Evolution Performance Validation
// ================================================================================================

export interface EvolutionBenchmarkResult {
    /** Performance results for each evolution stage */
    stageResults: BenchmarkResult[];
    /** Validation of improvements across stages */
    evolutionValidation: EvolutionValidationResult;
    /** Learning curve analysis */
    learningCurve: LearningCurveAnalysis;
    /** Compound improvement metrics */
    compoundImprovements: CompoundImprovementMetrics;
}

export interface EvolutionValidationResult {
    /** Whether evolution shows measurable improvement */
    improvementDetected: boolean;
    /** Specific improvements by metric */
    improvements: {
        latency: ImprovementMetric;
        accuracy: ImprovementMetric;
        cost: ImprovementMetric;
        efficiency: ImprovementMetric;
    };
    /** Evolution pathway effectiveness */
    pathwayEffectiveness: number; // 0-1 score
    /** Recommended next evolution steps */
    nextEvolutionSteps: string[];
}

export interface ImprovementMetric {
    improved: boolean;
    improvementPercent: number;
    statisticalSignificance: number; // p-value
    trendDirection: "improving" | "stable" | "degrading";
}

export interface LearningCurveAnalysis {
    /** Rate of improvement across stages */
    improvementRate: number;
    /** Whether improvements are accelerating */
    accelerating: boolean;
    /** Predicted performance at next stage */
    predictedNextStagePerformance: PerformanceMetrics;
    /** Diminishing returns analysis */
    diminishingReturns: boolean;
}

export interface CompoundImprovementMetrics {
    /** Overall system improvement factor */
    overallImprovementFactor: number;
    /** Most significant improvement areas */
    keyImprovementAreas: string[];
    /** ROI of evolution efforts */
    evolutionROI: number;
    /** Emergent performance benefits */
    emergentBenefits: string[];
}

// ================================================================================================
// Performance Benchmarker
// ================================================================================================

export class PerformanceBenchmarker {
    private executionRunner: ExecutionFixtureRunner;

    constructor() {
        this.executionRunner = new ExecutionFixtureRunner();
    }

    /**
     * Benchmark a single execution fixture
     */
    async benchmarkFixture<T extends BaseConfigObject>(
        fixture: ExecutionFixture<T>,
        config: BenchmarkConfig,
    ): Promise<BenchmarkResult> {
        // Warmup iterations
        if (config.environment?.warmupIterations) {
            await this.runWarmupIterations(fixture, config);
        }

        const iterations: IterationResult[] = [];
        const startTime = Date.now();

        // Run benchmark iterations
        for (let i = 0; i < config.iterations; i++) {
            const iterationStart = Date.now();
            
            try {
                const executionResult = await this.executionRunner.executeScenario(
                    fixture,
                    config.input,
                    {
                        timeout: config.targets?.maxLatencyMs || 30000,
                        validateEmergence: true,
                        resourceLimits: config.environment?.resourceLimits,
                    },
                );

                iterations.push({
                    iteration: i,
                    executionResult,
                    timestamp: iterationStart,
                    success: executionResult.success,
                });

                // Cooldown between iterations
                if (config.environment?.cooldownMs) {
                    await this.sleep(config.environment.cooldownMs);
                }
            } catch (error) {
                iterations.push({
                    iteration: i,
                    executionResult: {
                        success: false,
                        error: error instanceof Error ? error.message : String(error),
                        detectedCapabilities: [],
                        performanceMetrics: {
                            latency: Date.now() - iterationStart,
                            resourceUsage: { memory: 0, cpu: 0, tokens: 0 },
                        },
                    },
                    timestamp: iterationStart,
                    success: false,
                });
            }
        }

        const endTime = Date.now();
        const totalTime = endTime - startTime;

        // Calculate comprehensive metrics
        const metrics = this.calculatePerformanceMetrics(iterations, totalTime);
        const targetsValidation = this.validateTargets(metrics, config.targets);
        const characteristics = this.analyzePerformanceCharacteristics(iterations);
        const recommendations = this.generateRecommendations(metrics, characteristics, config.targets);

        return {
            metrics,
            iterations,
            targetsValidation,
            characteristics,
            recommendations,
        };
    }

    /**
     * Benchmark evolution pathway and validate improvements
     */
    async benchmarkEvolutionSequence(
        evolutionStages: RoutineFixture[],
        config: BenchmarkConfig,
    ): Promise<EvolutionBenchmarkResult> {
        const stageResults: BenchmarkResult[] = [];

        // Benchmark each evolution stage
        for (const [index, stage] of evolutionStages.entries()) {
            console.log(`Benchmarking evolution stage ${index + 1}: ${stage.evolutionStage?.current || "unknown"}`);
            
            const stageResult = await this.benchmarkFixture(stage, config);
            stageResults.push(stageResult);
        }

        // Analyze evolution improvements
        const evolutionValidation = this.validateEvolutionImprovements(stageResults);
        const learningCurve = this.analyzeLearningCurve(stageResults);
        const compoundImprovements = this.calculateCompoundImprovements(stageResults);

        return {
            stageResults,
            evolutionValidation,
            learningCurve,
            compoundImprovements,
        };
    }

    /**
     * Run competitive benchmarks against multiple fixtures
     */
    async runCompetitiveBenchmark<T extends BaseConfigObject>(
        fixtures: Array<{ name: string; fixture: ExecutionFixture<T> }>,
        config: BenchmarkConfig,
    ): Promise<CompetitiveBenchmarkResult> {
        const results: Array<{ name: string; result: BenchmarkResult }> = [];

        for (const { name, fixture } of fixtures) {
            const result = await this.benchmarkFixture(fixture, config);
            results.push({ name, result });
        }

        // Analyze comparative performance
        const ranking = this.rankFixtures(results);
        const comparativeAnalysis = this.analyzeComparativePerformance(results);

        return {
            results,
            ranking,
            comparativeAnalysis,
        };
    }

    // ================================================================================================
    // Private Implementation Methods
    // ================================================================================================

    private async runWarmupIterations<T extends BaseConfigObject>(
        fixture: ExecutionFixture<T>,
        config: BenchmarkConfig,
    ): Promise<void> {
        const warmupCount = config.environment?.warmupIterations || 3;
        
        for (let i = 0; i < warmupCount; i++) {
            try {
                await this.executionRunner.executeScenario(fixture, config.input, {
                    timeout: 10000,
                    validateEmergence: false,
                });
            } catch {
                // Ignore warmup errors
            }
        }
    }

    private calculatePerformanceMetrics(
        iterations: IterationResult[],
        totalTimeMs: number,
    ): PerformanceMetrics {
        const successfulIterations = iterations.filter(i => i.success);
        const latencies = successfulIterations.map(i => i.executionResult.performanceMetrics.latency);
        const accuracies = successfulIterations.map(i => i.executionResult.performanceMetrics.accuracy || 0);
        const costs = successfulIterations.map(i => i.executionResult.performanceMetrics.cost || 0);
        const memoryUsages = successfulIterations.map(i => i.executionResult.performanceMetrics.resourceUsage.memory);
        const cpuUsages = successfulIterations.map(i => i.executionResult.performanceMetrics.resourceUsage.cpu);
        const tokenUsages = successfulIterations.map(i => i.executionResult.performanceMetrics.resourceUsage.tokens);

        return {
            latency: this.calculateStatistics(latencies),
            accuracy: this.calculateStatistics(accuracies),
            cost: this.calculateStatistics(costs),
            memoryUsage: this.calculateStatistics(memoryUsages),
            cpuUsage: this.calculateStatistics(cpuUsages),
            tokenUsage: this.calculateStatistics(tokenUsages),
            throughput: (successfulIterations.length / totalTimeMs) * 1000, // per second
            availability: successfulIterations.length / iterations.length,
            reliability: this.calculateReliability(latencies),
        };
    }

    private calculateStatistics(values: number[]): StatisticalMetrics {
        if (values.length === 0) {
            return {
                mean: 0, median: 0, p95: 0, p99: 0,
                min: 0, max: 0, stdDev: 0,
            };
        }

        const sorted = values.slice().sort((a, b) => a - b);
        const mean = values.reduce((sum, v) => sum + v, 0) / values.length;
        const variance = values.reduce((sum, v) => sum + Math.pow(v - mean, 2), 0) / values.length;

        return {
            mean,
            median: this.percentile(sorted, 0.5),
            p95: this.percentile(sorted, 0.95),
            p99: this.percentile(sorted, 0.99),
            min: sorted[0],
            max: sorted[sorted.length - 1],
            stdDev: Math.sqrt(variance),
        };
    }

    private percentile(sortedValues: number[], p: number): number {
        const index = p * (sortedValues.length - 1);
        const lower = Math.floor(index);
        const upper = Math.ceil(index);
        const weight = index % 1;

        if (lower === upper) {
            return sortedValues[lower];
        }

        return sortedValues[lower] * (1 - weight) + sortedValues[upper] * weight;
    }

    private calculateReliability(latencies: number[]): number {
        if (latencies.length < 2) return 1.0;
        
        const stats = this.calculateStatistics(latencies);
        const coefficientOfVariation = stats.stdDev / stats.mean;
        
        // Convert coefficient of variation to reliability score (0-1)
        return Math.max(0, 1 - coefficientOfVariation);
    }

    private validateTargets(
        metrics: PerformanceMetrics,
        targets?: PerformanceTargets,
    ): TargetsValidationResult {
        if (!targets) {
            return {
                allTargetsMet: true,
                failedTargets: [],
                marginsByTarget: {},
            };
        }

        const failedTargets: string[] = [];
        const marginsByTarget: Record<string, number> = {};

        // Validate each target
        if (targets.maxLatencyMs !== undefined) {
            const margin = metrics.latency.p95 - targets.maxLatencyMs;
            marginsByTarget.maxLatencyMs = margin;
            if (margin > 0) failedTargets.push("maxLatencyMs");
        }

        if (targets.minAccuracy !== undefined) {
            const margin = targets.minAccuracy - metrics.accuracy.mean;
            marginsByTarget.minAccuracy = margin;
            if (margin > 0) failedTargets.push("minAccuracy");
        }

        if (targets.maxCost !== undefined) {
            const margin = metrics.cost.mean - targets.maxCost;
            marginsByTarget.maxCost = margin;
            if (margin > 0) failedTargets.push("maxCost");
        }

        if (targets.maxMemoryMB !== undefined) {
            const margin = metrics.memoryUsage.p95 - targets.maxMemoryMB;
            marginsByTarget.maxMemoryMB = margin;
            if (margin > 0) failedTargets.push("maxMemoryMB");
        }

        if (targets.minAvailability !== undefined) {
            const margin = targets.minAvailability - metrics.availability;
            marginsByTarget.minAvailability = margin;
            if (margin > 0) failedTargets.push("minAvailability");
        }

        return {
            allTargetsMet: failedTargets.length === 0,
            failedTargets,
            marginsByTarget,
        };
    }

    private analyzePerformanceCharacteristics(iterations: IterationResult[]): PerformanceCharacteristics {
        // Analyze patterns in the iteration data
        const latencies = iterations.map(i => i.executionResult.performanceMetrics.latency);
        
        // Determine scalability pattern (simplified analysis)
        const scalabilityPattern = this.determineScalabilityPattern(latencies);
        const performanceStability = this.determineStabilityPattern(latencies);
        const resourceEfficiency = this.determineResourceEfficiency(iterations);
        const errorHandlingImpact = this.determineErrorHandlingImpact(iterations);

        return {
            scalabilityPattern,
            performanceStability,
            resourceEfficiency,
            errorHandlingImpact,
        };
    }

    private determineScalabilityPattern(latencies: number[]): PerformanceCharacteristics["scalabilityPattern"] {
        if (latencies.length < 3) return "constant";
        
        // Simple trend analysis
        const firstHalf = latencies.slice(0, Math.floor(latencies.length / 2));
        const secondHalf = latencies.slice(Math.floor(latencies.length / 2));
        
        const firstAvg = firstHalf.reduce((sum, l) => sum + l, 0) / firstHalf.length;
        const secondAvg = secondHalf.reduce((sum, l) => sum + l, 0) / secondHalf.length;
        
        const improvement = (firstAvg - secondAvg) / firstAvg;
        
        if (improvement > 0.1) return "improving";
        if (improvement < -0.1) return "degrading";
        return "constant";
    }

    private determineStabilityPattern(latencies: number[]): PerformanceCharacteristics["performanceStability"] {
        const stats = this.calculateStatistics(latencies);
        const coefficientOfVariation = stats.stdDev / stats.mean;
        
        if (coefficientOfVariation < 0.1) return "stable";
        if (coefficientOfVariation < 0.3) return "variable";
        return "degrading";
    }

    private determineResourceEfficiency(iterations: IterationResult[]): PerformanceCharacteristics["resourceEfficiency"] {
        const avgMemory = iterations.reduce((sum, i) => 
            sum + i.executionResult.performanceMetrics.resourceUsage.memory, 0) / iterations.length;
        
        if (avgMemory < 500) return "efficient";
        if (avgMemory < 1000) return "moderate";
        return "inefficient";
    }

    private determineErrorHandlingImpact(iterations: IterationResult[]): PerformanceCharacteristics["errorHandlingImpact"] {
        const errorRate = 1 - (iterations.filter(i => i.success).length / iterations.length);
        
        if (errorRate < 0.05) return "minimal";
        if (errorRate < 0.15) return "moderate";
        return "significant";
    }

    private generateRecommendations(
        metrics: PerformanceMetrics,
        characteristics: PerformanceCharacteristics,
        targets?: PerformanceTargets,
    ): PerformanceRecommendation[] {
        const recommendations: PerformanceRecommendation[] = [];

        // Latency recommendations
        if (metrics.latency.p95 > (targets?.maxLatencyMs || 1000)) {
            recommendations.push({
                category: "latency",
                priority: "high",
                description: "High latency detected in 95th percentile",
                potentialImprovement: "Optimize critical path and reduce computational complexity",
                implementationComplexity: "medium",
            });
        }

        // Accuracy recommendations
        if (metrics.accuracy.mean < (targets?.minAccuracy || 0.9)) {
            recommendations.push({
                category: "accuracy",
                priority: "critical",
                description: "Accuracy below target threshold",
                potentialImprovement: "Improve model training or add validation layers",
                implementationComplexity: "high",
            });
        }

        // Memory efficiency recommendations
        if (characteristics.resourceEfficiency === "inefficient") {
            recommendations.push({
                category: "memory",
                priority: "medium",
                description: "High memory usage detected",
                potentialImprovement: "Implement memory optimization and garbage collection",
                implementationComplexity: "medium",
            });
        }

        return recommendations;
    }

    private validateEvolutionImprovements(stageResults: BenchmarkResult[]): EvolutionValidationResult {
        if (stageResults.length < 2) {
            return {
                improvementDetected: false,
                improvements: {
                    latency: { improved: false, improvementPercent: 0, statisticalSignificance: 1, trendDirection: "stable" },
                    accuracy: { improved: false, improvementPercent: 0, statisticalSignificance: 1, trendDirection: "stable" },
                    cost: { improved: false, improvementPercent: 0, statisticalSignificance: 1, trendDirection: "stable" },
                    efficiency: { improved: false, improvementPercent: 0, statisticalSignificance: 1, trendDirection: "stable" },
                },
                pathwayEffectiveness: 0,
                nextEvolutionSteps: [],
            };
        }

        // Compare first and last stages
        const first = stageResults[0];
        const last = stageResults[stageResults.length - 1];

        const latencyImprovement = this.calculateImprovement(
            first.metrics.latency.mean,
            last.metrics.latency.mean,
            "lower_is_better",
        );

        const accuracyImprovement = this.calculateImprovement(
            first.metrics.accuracy.mean,
            last.metrics.accuracy.mean,
            "higher_is_better",
        );

        const costImprovement = this.calculateImprovement(
            first.metrics.cost.mean,
            last.metrics.cost.mean,
            "lower_is_better",
        );

        const efficiencyImprovement = this.calculateImprovement(
            first.metrics.throughput,
            last.metrics.throughput,
            "higher_is_better",
        );

        const overallImprovement = [
            latencyImprovement.improved,
            accuracyImprovement.improved,
            costImprovement.improved,
            efficiencyImprovement.improved,
        ].filter(Boolean).length > 2;

        return {
            improvementDetected: overallImprovement,
            improvements: {
                latency: latencyImprovement,
                accuracy: accuracyImprovement,
                cost: costImprovement,
                efficiency: efficiencyImprovement,
            },
            pathwayEffectiveness: this.calculatePathwayEffectiveness(stageResults),
            nextEvolutionSteps: this.suggestNextEvolutionSteps(stageResults),
        };
    }

    private calculateImprovement(
        baseline: number,
        current: number,
        direction: "higher_is_better" | "lower_is_better",
    ): ImprovementMetric {
        const improvementPercent = direction === "higher_is_better"
            ? ((current - baseline) / baseline) * 100
            : ((baseline - current) / baseline) * 100;

        const improved = improvementPercent > 0;
        
        return {
            improved,
            improvementPercent,
            statisticalSignificance: this.calculateStatisticalSignificance(baseline, current),
            trendDirection: improved ? "improving" : improvementPercent < -5 ? "degrading" : "stable",
        };
    }

    private calculateStatisticalSignificance(baseline: number, current: number): number {
        // Simplified statistical significance calculation
        // In a real implementation, this would use proper statistical tests
        const difference = Math.abs(current - baseline);
        const relativeDifference = difference / baseline;
        
        if (relativeDifference > 0.1) return 0.01; // Highly significant
        if (relativeDifference > 0.05) return 0.05; // Significant
        return 0.5; // Not significant
    }

    private calculatePathwayEffectiveness(stageResults: BenchmarkResult[]): number {
        // Calculate overall effectiveness of the evolution pathway
        let effectivenessScore = 0;
        let comparisons = 0;

        for (let i = 1; i < stageResults.length; i++) {
            const prev = stageResults[i - 1];
            const curr = stageResults[i];

            // Check improvements in key metrics
            if (curr.metrics.latency.mean < prev.metrics.latency.mean) effectivenessScore++;
            if (curr.metrics.accuracy.mean > prev.metrics.accuracy.mean) effectivenessScore++;
            if (curr.metrics.cost.mean < prev.metrics.cost.mean) effectivenessScore++;
            if (curr.metrics.throughput > prev.metrics.throughput) effectivenessScore++;

            comparisons += 4;
        }

        return comparisons > 0 ? effectivenessScore / comparisons : 0;
    }

    private suggestNextEvolutionSteps(stageResults: BenchmarkResult[]): string[] {
        const suggestions: string[] = [];
        const latest = stageResults[stageResults.length - 1];

        // Analyze weakest areas and suggest improvements
        if (latest.metrics.latency.p95 > 1000) {
            suggestions.push("Optimize latency through caching and parallel processing");
        }

        if (latest.metrics.accuracy.mean < 0.9) {
            suggestions.push("Improve accuracy through better model training or ensemble methods");
        }

        if (latest.metrics.cost.mean > 0.1) {
            suggestions.push("Reduce costs through model optimization and resource efficiency");
        }

        if (latest.metrics.availability < 0.95) {
            suggestions.push("Improve reliability through better error handling and redundancy");
        }

        return suggestions;
    }

    private analyzeLearningCurve(stageResults: BenchmarkResult[]): LearningCurveAnalysis {
        // Analyze the learning curve across evolution stages
        const improvementRates = [];
        
        for (let i = 1; i < stageResults.length; i++) {
            const prev = stageResults[i - 1];
            const curr = stageResults[i];
            
            const latencyImprovement = (prev.metrics.latency.mean - curr.metrics.latency.mean) / prev.metrics.latency.mean;
            improvementRates.push(latencyImprovement);
        }

        const avgImprovementRate = improvementRates.reduce((sum, rate) => sum + rate, 0) / improvementRates.length;
        const accelerating = improvementRates.length > 1 && 
            improvementRates[improvementRates.length - 1] > improvementRates[0];

        // Predict next stage performance (simplified)
        const latest = stageResults[stageResults.length - 1];
        const predictedNextStagePerformance: PerformanceMetrics = {
            ...latest.metrics,
            latency: {
                ...latest.metrics.latency,
                mean: latest.metrics.latency.mean * (1 - avgImprovementRate),
            },
        };

        return {
            improvementRate: avgImprovementRate,
            accelerating,
            predictedNextStagePerformance,
            diminishingReturns: avgImprovementRate < 0.05,
        };
    }

    private calculateCompoundImprovements(stageResults: BenchmarkResult[]): CompoundImprovementMetrics {
        if (stageResults.length < 2) {
            return {
                overallImprovementFactor: 1,
                keyImprovementAreas: [],
                evolutionROI: 0,
                emergentBenefits: [],
            };
        }

        const first = stageResults[0];
        const last = stageResults[stageResults.length - 1];

        const latencyFactor = first.metrics.latency.mean / last.metrics.latency.mean;
        const accuracyFactor = last.metrics.accuracy.mean / first.metrics.accuracy.mean;
        const throughputFactor = last.metrics.throughput / first.metrics.throughput;

        const overallImprovementFactor = (latencyFactor + accuracyFactor + throughputFactor) / 3;

        return {
            overallImprovementFactor,
            keyImprovementAreas: this.identifyKeyImprovementAreas(first, last),
            evolutionROI: this.calculateEvolutionROI(stageResults),
            emergentBenefits: this.identifyEmergentBenefits(stageResults),
        };
    }

    private identifyKeyImprovementAreas(first: BenchmarkResult, last: BenchmarkResult): string[] {
        const improvements: string[] = [];

        if (last.metrics.latency.mean < first.metrics.latency.mean * 0.8) {
            improvements.push("latency");
        }
        if (last.metrics.accuracy.mean > first.metrics.accuracy.mean * 1.1) {
            improvements.push("accuracy");
        }
        if (last.metrics.throughput > first.metrics.throughput * 1.2) {
            improvements.push("throughput");
        }
        if (last.metrics.cost.mean < first.metrics.cost.mean * 0.8) {
            improvements.push("cost");
        }

        return improvements;
    }

    private calculateEvolutionROI(stageResults: BenchmarkResult[]): number {
        // Simplified ROI calculation based on performance improvements vs. complexity
        const first = stageResults[0];
        const last = stageResults[stageResults.length - 1];

        const performanceGain = (last.metrics.throughput / first.metrics.throughput) - 1;
        const complexityIncrease = stageResults.length - 1; // Simplified complexity measure

        return performanceGain / Math.max(complexityIncrease, 1);
    }

    private identifyEmergentBenefits(stageResults: BenchmarkResult[]): string[] {
        // Identify unexpected benefits that emerged during evolution
        const benefits: string[] = [];

        // Check for emergent properties
        const latest = stageResults[stageResults.length - 1];
        
        if (latest.metrics.reliability > 0.95) {
            benefits.push("high_reliability");
        }
        if (latest.metrics.availability > 0.99) {
            benefits.push("exceptional_availability");
        }
        if (latest.characteristics.resourceEfficiency === "efficient") {
            benefits.push("resource_efficiency");
        }

        return benefits;
    }

    private rankFixtures(
        results: Array<{ name: string; result: BenchmarkResult }>,
    ): FixtureRanking[] {
        return results
            .map(({ name, result }) => ({
                name,
                overallScore: this.calculateOverallScore(result),
                result,
            }))
            .sort((a, b) => b.overallScore - a.overallScore);
    }

    private calculateOverallScore(result: BenchmarkResult): number {
        // Composite score considering multiple factors
        const weights = {
            latency: 0.3,
            accuracy: 0.3,
            availability: 0.2,
            cost: 0.1,
            reliability: 0.1,
        };

        // Normalize metrics to 0-1 scale (simplified)
        const latencyScore = Math.max(0, 1 - (result.metrics.latency.mean / 5000));
        const accuracyScore = result.metrics.accuracy.mean;
        const availabilityScore = result.metrics.availability;
        const costScore = Math.max(0, 1 - result.metrics.cost.mean);
        const reliabilityScore = result.metrics.reliability;

        return (
            weights.latency * latencyScore +
            weights.accuracy * accuracyScore +
            weights.availability * availabilityScore +
            weights.cost * costScore +
            weights.reliability * reliabilityScore
        );
    }

    private analyzeComparativePerformance(
        results: Array<{ name: string; result: BenchmarkResult }>,
    ): ComparativeAnalysis {
        // Analyze performance differences between fixtures
        const performanceGaps = new Map<string, number>();
        const strengthsByFixture = new Map<string, string[]>();

        for (const { name, result } of results) {
            const strengths: string[] = [];
            
            // Identify strengths
            if (result.metrics.latency.mean < 1000) strengths.push("low_latency");
            if (result.metrics.accuracy.mean > 0.9) strengths.push("high_accuracy");
            if (result.metrics.availability > 0.95) strengths.push("high_availability");
            if (result.metrics.cost.mean < 0.05) strengths.push("cost_effective");

            strengthsByFixture.set(name, strengths);
        }

        return {
            performanceGaps,
            strengthsByFixture,
            recommendedUseCases: this.generateUseCaseRecommendations(results),
        };
    }

    private generateUseCaseRecommendations(
        results: Array<{ name: string; result: BenchmarkResult }>,
    ): Record<string, string[]> {
        const recommendations: Record<string, string[]> = {};

        for (const { name, result } of results) {
            const useCases: string[] = [];

            if (result.metrics.latency.mean < 500) {
                useCases.push("real_time_applications");
            }
            if (result.metrics.accuracy.mean > 0.95) {
                useCases.push("high_precision_tasks");
            }
            if (result.metrics.cost.mean < 0.02) {
                useCases.push("high_volume_processing");
            }
            if (result.metrics.availability > 0.99) {
                useCases.push("mission_critical_systems");
            }

            recommendations[name] = useCases;
        }

        return recommendations;
    }

    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// ================================================================================================
// Supporting Types for Competitive Benchmarking
// ================================================================================================

export interface CompetitiveBenchmarkResult {
    results: Array<{ name: string; result: BenchmarkResult }>;
    ranking: FixtureRanking[];
    comparativeAnalysis: ComparativeAnalysis;
}

export interface FixtureRanking {
    name: string;
    overallScore: number;
    result: BenchmarkResult;
}

export interface ComparativeAnalysis {
    performanceGaps: Map<string, number>;
    strengthsByFixture: Map<string, string[]>;
    recommendedUseCases: Record<string, string[]>;
}

// ================================================================================================
// Test Utilities for Performance Benchmarking
// ================================================================================================

/**
 * Run performance benchmark tests for a fixture
 */
export function runPerformanceBenchmarkTests<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    benchmarkConfig: BenchmarkConfig,
    fixtureName: string,
): void {
    describe(`${fixtureName} performance benchmarks`, () => {
        const benchmarker = new PerformanceBenchmarker();

        it("should meet performance targets", async () => {
            const result = await benchmarker.benchmarkFixture(fixture, benchmarkConfig);
            
            if (benchmarkConfig.targets) {
                expect(result.targetsValidation.allTargetsMet).toBe(true);
                
                if (!result.targetsValidation.allTargetsMet) {
                    console.error("Failed targets:", result.targetsValidation.failedTargets);
                    console.error("Margins:", result.targetsValidation.marginsByTarget);
                }
            }
        }, 60000); // 1 minute timeout

        it("should demonstrate acceptable performance characteristics", async () => {
            const result = await benchmarker.benchmarkFixture(fixture, benchmarkConfig);
            
            // Basic performance expectations
            expect(result.metrics.availability).toBeGreaterThan(0.8);
            expect(result.metrics.reliability).toBeGreaterThan(0.7);
            
            // Should not have critical performance issues
            expect(result.characteristics.performanceStability).not.toBe("degrading");
            expect(result.characteristics.errorHandlingImpact).not.toBe("significant");
        }, 60000);

        it("should provide actionable performance recommendations", async () => {
            const result = await benchmarker.benchmarkFixture(fixture, benchmarkConfig);
            
            // Should generate recommendations if targets not met
            if (!result.targetsValidation.allTargetsMet) {
                expect(result.recommendations.length).toBeGreaterThan(0);
                
                // Recommendations should be categorized and prioritized
                result.recommendations.forEach(rec => {
                    expect(rec.category).toBeDefined();
                    expect(rec.priority).toBeDefined();
                    expect(rec.description).toBeDefined();
                });
            }
        }, 30000);
    });
}

/**
 * Run evolution benchmark tests for routine evolution stages
 */
export function runEvolutionBenchmarkTests(
    evolutionStages: RoutineFixture[],
    benchmarkConfig: BenchmarkConfig,
    fixtureName: string,
): void {
    describe(`${fixtureName} evolution benchmarks`, () => {
        const benchmarker = new PerformanceBenchmarker();

        it("should show measurable improvement across evolution stages", async () => {
            const result = await benchmarker.benchmarkEvolutionSequence(evolutionStages, benchmarkConfig);
            
            expect(result.evolutionValidation.improvementDetected).toBe(true);
            
            // At least one key metric should improve
            const improvements = result.evolutionValidation.improvements;
            const improvedMetrics = [
                improvements.latency.improved,
                improvements.accuracy.improved,
                improvements.cost.improved,
                improvements.efficiency.improved,
            ].filter(Boolean).length;
            
            expect(improvedMetrics).toBeGreaterThan(0);
        }, 120000); // 2 minute timeout

        it("should demonstrate effective evolution pathway", async () => {
            const result = await benchmarker.benchmarkEvolutionSequence(evolutionStages, benchmarkConfig);
            
            expect(result.evolutionValidation.pathwayEffectiveness).toBeGreaterThan(0.5);
            expect(result.compoundImprovements.overallImprovementFactor).toBeGreaterThan(1.0);
        }, 120000);

        it("should provide next evolution steps", async () => {
            const result = await benchmarker.benchmarkEvolutionSequence(evolutionStages, benchmarkConfig);
            
            expect(result.evolutionValidation.nextEvolutionSteps.length).toBeGreaterThan(0);
            expect(result.learningCurve.predictedNextStagePerformance).toBeDefined();
        }, 120000);
    });
}
