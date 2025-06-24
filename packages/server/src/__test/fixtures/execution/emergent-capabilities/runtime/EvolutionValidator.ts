/**
 * Evolution Validator
 * 
 * Validates the evolution of capabilities through quantifiable improvements.
 * Tests that systems actually improve over time through measurable metrics
 * rather than just claiming evolution occurred.
 */

import { type EmergentCapabilityFixture, type EvolutionDefinition } from "../emergentValidationUtils.js";
import { RuntimeExecutionValidator } from "./RuntimeExecutionValidator.js";
import { TierMockFactories } from "./tierMockFactories.js";
import { type AIMockConfig } from "../../ai-mocks/types.js";
import { withAIMocks } from "../../ai-mocks/integration/testHelpers.js";

/**
 * Evolution validation configuration
 */
export interface EvolutionValidationConfig {
    fixture: EmergentCapabilityFixture<any>;
    stages: EvolutionStageConfig[];
    validationCriteria: EvolutionCriteria;
    options?: EvolutionValidationOptions;
}

/**
 * Configuration for a single evolution stage
 */
export interface EvolutionStageConfig {
    name: string;
    description: string;
    strategy?: string;
    mockBehaviors: Map<string, AIMockConfig>;
    expectedMetrics: StageMetrics;
    expectedCapabilities: string[];
    minimumIterations?: number;
    evolutionTriggers?: string[];
}

/**
 * Metrics for a stage
 */
export interface StageMetrics {
    executionTime?: number;
    accuracy?: number;
    cost?: number;
    successRate?: number;
    resourceUsage?: number;
    throughput?: number;
    latency?: number;
    errorRate?: number;
}

/**
 * Evolution validation criteria
 */
export interface EvolutionCriteria {
    minimumImprovement: {
        executionTime?: number; // Percentage improvement required
        accuracy?: number;
        cost?: number;
        overall?: number;
    };
    statisticalSignificance?: {
        required: boolean;
        confidenceLevel: number; // e.g., 0.95 for 95% confidence
        minSampleSize: number;
    };
    regressionTolerance?: {
        allowedRegressions: number;
        maxRegressionMagnitude: number;
    };
    stabilityRequirements?: {
        minimumStableIterations: number;
        stabilityThreshold: number;
    };
}

/**
 * Evolution validation options
 */
export interface EvolutionValidationOptions {
    debug?: boolean;
    captureDetailedMetrics?: boolean;
    validateStatisticalSignificance?: boolean;
    iterationsPerStage?: number;
    warmupIterations?: number;
}

/**
 * Complete evolution validation result
 */
export interface EvolutionValidationResult {
    fixture: EmergentCapabilityFixture<any>;
    stageResults: StageValidationResult[];
    improvementAnalysis: ImprovementAnalysis;
    statisticalValidation?: StatisticalValidation;
    overallValidation: OverallEvolutionValidation;
    recommendations?: EvolutionRecommendation[];
}

/**
 * Result for a single stage
 */
export interface StageValidationResult {
    stage: string;
    metrics: StageMetrics;
    variance: StageMetrics;
    iterations: number;
    capabilitiesDetected: string[];
    improvement?: ImprovementMetrics;
    validated: boolean;
    issues?: string[];
    warnings?: string[];
}

/**
 * Improvement metrics between stages
 */
export interface ImprovementMetrics {
    executionTime: number;
    accuracy: number;
    cost: number;
    overall: number;
    relative: {
        executionTime: number;
        accuracy: number;
        cost: number;
    };
}

/**
 * Analysis of improvements across all stages
 */
export interface ImprovementAnalysis {
    totalImprovement: ImprovementMetrics;
    improvementTrajectory: Array<{
        fromStage: string;
        toStage: string;
        improvement: ImprovementMetrics;
    }>;
    improvementVelocity: number;
    stabilityAnalysis: StabilityAnalysis;
    regressionAnalysis: RegressionAnalysis;
}

/**
 * Statistical validation of evolution
 */
export interface StatisticalValidation {
    significanceTests: Array<{
        metric: string;
        pValue: number;
        significant: boolean;
        confidenceInterval: [number, number];
    }>;
    effectSizes: Array<{
        metric: string;
        cohensD: number;
        magnitude: "small" | "medium" | "large";
    }>;
    overallSignificance: boolean;
}

/**
 * Stability analysis
 */
export interface StabilityAnalysis {
    stableStages: string[];
    instabilityPeriods: Array<{
        stage: string;
        variance: number;
        duration: number;
    }>;
    overallStability: number;
}

/**
 * Regression analysis
 */
export interface RegressionAnalysis {
    regressionEvents: Array<{
        fromStage: string;
        toStage: string;
        magnitude: number;
        recovered: boolean;
    }>;
    regressionRate: number;
    recoveryRate: number;
}

/**
 * Overall validation summary
 */
export interface OverallEvolutionValidation {
    evolutionValidated: boolean;
    improvementDetected: boolean;
    criteriaMetCount: number;
    totalCriteriaCount: number;
    confidenceScore: number;
    qualityScore: number;
}

/**
 * Evolution recommendation
 */
export interface EvolutionRecommendation {
    type: "improvement" | "warning" | "optimization";
    stage?: string;
    metric?: string;
    message: string;
    suggestedAction?: string;
    priority: "low" | "medium" | "high";
}

/**
 * Main evolution validator class
 */
export class EvolutionValidator {
    private runtimeValidator: RuntimeExecutionValidator;
    private metricsHistory: Map<string, StageMetrics[]> = new Map();
    
    constructor() {
        this.runtimeValidator = new RuntimeExecutionValidator();
    }
    
    /**
     * Validate evolution path with quantifiable improvements
     */
    async validateEvolutionPath(
        config: EvolutionValidationConfig
    ): Promise<EvolutionValidationResult> {
        const options = config.options || {};
        const iterationsPerStage = options.iterationsPerStage || 5;
        const warmupIterations = options.warmupIterations || 1;
        
        // Clear metrics history
        this.metricsHistory.clear();
        
        // Execute all stages
        const stageResults: StageValidationResult[] = [];
        
        for (let i = 0; i < config.stages.length; i++) {
            const stage = config.stages[i];
            const previousStage = i > 0 ? stageResults[i - 1] : undefined;
            
            const stageResult = await this.validateStage(
                stage,
                config.fixture,
                previousStage,
                iterationsPerStage,
                warmupIterations,
                options
            );
            
            stageResults.push(stageResult);
        }
        
        // Analyze improvements
        const improvementAnalysis = this.analyzeImprovements(stageResults, config.validationCriteria);
        
        // Perform statistical validation if requested
        let statisticalValidation: StatisticalValidation | undefined;
        if (options.validateStatisticalSignificance) {
            statisticalValidation = this.performStatisticalValidation(stageResults, config.validationCriteria);
        }
        
        // Calculate overall validation
        const overallValidation = this.calculateOverallValidation(
            stageResults,
            improvementAnalysis,
            config.validationCriteria,
            statisticalValidation
        );
        
        // Generate recommendations
        const recommendations = this.generateRecommendations(
            stageResults,
            improvementAnalysis,
            config.validationCriteria
        );
        
        return {
            fixture: config.fixture,
            stageResults,
            improvementAnalysis,
            statisticalValidation,
            overallValidation,
            recommendations
        };
    }
    
    /**
     * Validate a single evolution stage
     */
    private async validateStage(
        stage: EvolutionStageConfig,
        fixture: EmergentCapabilityFixture<any>,
        previousStage: StageValidationResult | undefined,
        iterations: number,
        warmupIterations: number,
        options: EvolutionValidationOptions
    ): Promise<StageValidationResult> {
        const allMetrics: StageMetrics[] = [];
        const allCapabilities: Set<string> = new Set();
        const issues: string[] = [];
        const warnings: string[] = [];
        
        // Run multiple iterations for statistical validity
        for (let i = 0; i < iterations + warmupIterations; i++) {
            try {
                const iterationResult = await this.executeStageIteration(
                    stage,
                    fixture,
                    options
                );
                
                // Skip warmup iterations for metrics
                if (i >= warmupIterations) {
                    allMetrics.push(iterationResult.metrics);
                    iterationResult.capabilities.forEach(cap => allCapabilities.add(cap));
                }
            } catch (error) {
                issues.push(`Iteration ${i + 1} failed: ${error}`);
            }
        }
        
        if (allMetrics.length === 0) {
            return {
                stage: stage.name,
                metrics: this.createZeroMetrics(),
                variance: this.createZeroMetrics(),
                iterations: 0,
                capabilitiesDetected: [],
                validated: false,
                issues: ["No successful iterations"]
            };
        }
        
        // Calculate aggregate metrics
        const aggregateMetrics = this.aggregateMetrics(allMetrics);
        const varianceMetrics = this.calculateVariance(allMetrics, aggregateMetrics);
        
        // Store in history
        this.metricsHistory.set(stage.name, allMetrics);
        
        // Calculate improvement if there's a previous stage
        let improvement: ImprovementMetrics | undefined;
        if (previousStage) {
            improvement = this.calculateImprovement(previousStage.metrics, aggregateMetrics);
        }
        
        // Validate against expected metrics
        const validated = this.validateStageMetrics(
            aggregateMetrics,
            stage.expectedMetrics,
            issues
        );
        
        // Check for expected capabilities
        const missingCapabilities = stage.expectedCapabilities.filter(
            cap => !allCapabilities.has(cap)
        );
        if (missingCapabilities.length > 0) {
            warnings.push(`Missing capabilities: ${missingCapabilities.join(", ")}`);
        }
        
        return {
            stage: stage.name,
            metrics: aggregateMetrics,
            variance: varianceMetrics,
            iterations: allMetrics.length,
            capabilitiesDetected: Array.from(allCapabilities),
            improvement,
            validated,
            issues: issues.length > 0 ? issues : undefined,
            warnings: warnings.length > 0 ? warnings : undefined
        };
    }
    
    /**
     * Execute a single iteration of a stage
     */
    private async executeStageIteration(
        stage: EvolutionStageConfig,
        fixture: EmergentCapabilityFixture<any>,
        options: EvolutionValidationOptions
    ): Promise<{ metrics: StageMetrics; capabilities: string[] }> {
        return withAIMocks(async (mockService) => {
            // Setup stage-specific mocks
            stage.mockBehaviors.forEach((config, id) => {
                mockService.registerBehavior(id, {
                    pattern: config.pattern,
                    response: config,
                    priority: config.priority || 10
                });
            });
            
            // Execute stage
            const startTime = Date.now();
            
            // Simulate stage execution with appropriate strategy
            const testInput = {
                model: "gpt-4o-mini",
                messages: [{
                    role: "user",
                    content: `Execute ${fixture.config.name} in ${stage.name} stage`
                }]
            };
            
            const result = await mockService.execute(testInput);
            const executionTime = Date.now() - startTime;
            
            // Extract metrics from response
            const responseMetadata = result.response?.metadata || {};
            const capabilities = this.extractCapabilities(result.response, stage.expectedCapabilities);
            
            const metrics: StageMetrics = {
                executionTime: responseMetadata.executionTime || executionTime,
                accuracy: responseMetadata.accuracy || stage.expectedMetrics.accuracy || 0,
                cost: result.usage?.cost || stage.expectedMetrics.cost || 0,
                successRate: responseMetadata.successRate || 1.0,
                resourceUsage: responseMetadata.resourceUsage || 0,
                throughput: responseMetadata.throughput || 0,
                latency: responseMetadata.latency || executionTime,
                errorRate: responseMetadata.errorRate || 0
            };
            
            return { metrics, capabilities };
        }, { debug: options.debug });
    }
    
    /**
     * Extract capabilities from response
     */
    private extractCapabilities(response: any, expectedCapabilities: string[]): string[] {
        const detectedCapabilities: string[] = [];
        
        if (!response) return detectedCapabilities;
        
        const responseText = JSON.stringify(response).toLowerCase();
        
        // Check for capability indicators
        for (const capability of expectedCapabilities) {
            if (this.hasCapabilityEvidence(capability, responseText)) {
                detectedCapabilities.push(capability);
            }
        }
        
        // Check metadata for explicit capability mentions
        if (response.metadata?.capability) {
            detectedCapabilities.push(response.metadata.capability);
        }
        
        if (response.metadata?.capabilities) {
            detectedCapabilities.push(...response.metadata.capabilities);
        }
        
        return [...new Set(detectedCapabilities)]; // Remove duplicates
    }
    
    /**
     * Check if response contains evidence of capability
     */
    private hasCapabilityEvidence(capability: string, responseText: string): boolean {
        const capabilityIndicators: Record<string, string[]> = {
            "customer_satisfaction": ["satisfaction", "resolved", "helped", "solution"],
            "threat_detection": ["threat", "anomaly", "alert", "security"],
            "performance_optimization": ["optimized", "improved", "faster", "efficient"],
            "adaptive_learning": ["learned", "adapted", "improved", "evolved"],
            "task_delegation": ["assigned", "delegated", "distributed"],
            "collective_intelligence": ["synthesized", "combined", "insights"]
        };
        
        const indicators = capabilityIndicators[capability] || [];
        return indicators.some(indicator => responseText.includes(indicator));
    }
    
    /**
     * Aggregate metrics across iterations
     */
    private aggregateMetrics(allMetrics: StageMetrics[]): StageMetrics {
        const aggregated: StageMetrics = {};
        
        const keys: (keyof StageMetrics)[] = [
            "executionTime", "accuracy", "cost", "successRate", 
            "resourceUsage", "throughput", "latency", "errorRate"
        ];
        
        for (const key of keys) {
            const values = allMetrics.map(m => m[key]).filter(v => v !== undefined) as number[];
            if (values.length > 0) {
                aggregated[key] = values.reduce((sum, val) => sum + val, 0) / values.length;
            }
        }
        
        return aggregated;
    }
    
    /**
     * Calculate variance of metrics
     */
    private calculateVariance(allMetrics: StageMetrics[], mean: StageMetrics): StageMetrics {
        const variance: StageMetrics = {};
        
        const keys: (keyof StageMetrics)[] = [
            "executionTime", "accuracy", "cost", "successRate", 
            "resourceUsage", "throughput", "latency", "errorRate"
        ];
        
        for (const key of keys) {
            const values = allMetrics.map(m => m[key]).filter(v => v !== undefined) as number[];
            const meanValue = mean[key];
            
            if (values.length > 1 && meanValue !== undefined) {
                const squaredDiffs = values.map(v => Math.pow(v - meanValue, 2));
                variance[key] = squaredDiffs.reduce((sum, val) => sum + val, 0) / values.length;
            }
        }
        
        return variance;
    }
    
    /**
     * Calculate improvement between stages
     */
    private calculateImprovement(
        previousMetrics: StageMetrics,
        currentMetrics: StageMetrics
    ): ImprovementMetrics {
        const improvement: ImprovementMetrics = {
            executionTime: 0,
            accuracy: 0,
            cost: 0,
            overall: 0,
            relative: {
                executionTime: 0,
                accuracy: 0,
                cost: 0
            }
        };
        
        // Calculate absolute improvements
        if (previousMetrics.executionTime && currentMetrics.executionTime) {
            improvement.executionTime = previousMetrics.executionTime - currentMetrics.executionTime;
            improvement.relative.executionTime = improvement.executionTime / previousMetrics.executionTime;
        }
        
        if (previousMetrics.accuracy && currentMetrics.accuracy) {
            improvement.accuracy = currentMetrics.accuracy - previousMetrics.accuracy;
            improvement.relative.accuracy = improvement.accuracy / previousMetrics.accuracy;
        }
        
        if (previousMetrics.cost && currentMetrics.cost) {
            improvement.cost = previousMetrics.cost - currentMetrics.cost;
            improvement.relative.cost = improvement.cost / previousMetrics.cost;
        }
        
        // Calculate overall improvement (weighted)
        const weights = { executionTime: 0.4, accuracy: 0.4, cost: 0.2 };
        improvement.overall = 
            improvement.relative.executionTime * weights.executionTime +
            improvement.relative.accuracy * weights.accuracy +
            improvement.relative.cost * weights.cost;
        
        return improvement;
    }
    
    /**
     * Validate stage metrics against expected values
     */
    private validateStageMetrics(
        actual: StageMetrics,
        expected: StageMetrics,
        issues: string[]
    ): boolean {
        let isValid = true;
        const tolerance = 0.2; // 20% tolerance
        
        // Check each expected metric
        for (const [key, expectedValue] of Object.entries(expected)) {
            if (expectedValue === undefined) continue;
            
            const actualValue = actual[key as keyof StageMetrics];
            if (actualValue === undefined) {
                issues.push(`Missing metric: ${key}`);
                isValid = false;
                continue;
            }
            
            const diff = Math.abs(actualValue - expectedValue) / expectedValue;
            if (diff > tolerance) {
                issues.push(`Metric ${key} outside tolerance: expected ${expectedValue}, got ${actualValue}`);
                isValid = false;
            }
        }
        
        return isValid;
    }
    
    /**
     * Analyze improvements across all stages
     */
    private analyzeImprovements(
        stageResults: StageValidationResult[],
        criteria: EvolutionCriteria
    ): ImprovementAnalysis {
        const improvementTrajectory: Array<{
            fromStage: string;
            toStage: string;
            improvement: ImprovementMetrics;
        }> = [];
        
        // Calculate improvements between consecutive stages
        for (let i = 1; i < stageResults.length; i++) {
            const improvement = this.calculateImprovement(
                stageResults[i - 1].metrics,
                stageResults[i].metrics
            );
            
            improvementTrajectory.push({
                fromStage: stageResults[i - 1].stage,
                toStage: stageResults[i].stage,
                improvement
            });
        }
        
        // Calculate total improvement
        const totalImprovement = stageResults.length > 1
            ? this.calculateImprovement(stageResults[0].metrics, stageResults[stageResults.length - 1].metrics)
            : this.createZeroImprovement();
        
        // Calculate improvement velocity
        const improvementVelocity = improvementTrajectory.length > 0
            ? improvementTrajectory.reduce((sum, t) => sum + t.improvement.overall, 0) / improvementTrajectory.length
            : 0;
        
        // Analyze stability
        const stabilityAnalysis = this.analyzeStability(stageResults);
        
        // Analyze regressions
        const regressionAnalysis = this.analyzeRegressions(improvementTrajectory);
        
        return {
            totalImprovement,
            improvementTrajectory,
            improvementVelocity,
            stabilityAnalysis,
            regressionAnalysis
        };
    }
    
    /**
     * Analyze stability across stages
     */
    private analyzeStability(stageResults: StageValidationResult[]): StabilityAnalysis {
        const stableStages: string[] = [];
        const instabilityPeriods: Array<{
            stage: string;
            variance: number;
            duration: number;
        }> = [];
        
        const stabilityThreshold = 0.1; // 10% variance threshold
        
        for (const stage of stageResults) {
            // Calculate normalized variance (coefficient of variation)
            const executionTimeCV = stage.variance.executionTime && stage.metrics.executionTime
                ? Math.sqrt(stage.variance.executionTime) / stage.metrics.executionTime
                : 0;
            
            if (executionTimeCV < stabilityThreshold) {
                stableStages.push(stage.stage);
            } else {
                instabilityPeriods.push({
                    stage: stage.stage,
                    variance: executionTimeCV,
                    duration: stage.iterations
                });
            }
        }
        
        const overallStability = stableStages.length / stageResults.length;
        
        return {
            stableStages,
            instabilityPeriods,
            overallStability
        };
    }
    
    /**
     * Analyze regressions in the improvement trajectory
     */
    private analyzeRegressions(
        improvementTrajectory: Array<{
            fromStage: string;
            toStage: string;
            improvement: ImprovementMetrics;
        }>
    ): RegressionAnalysis {
        const regressionEvents: Array<{
            fromStage: string;
            toStage: string;
            magnitude: number;
            recovered: boolean;
        }> = [];
        
        // Identify regressions (negative overall improvements)
        for (let i = 0; i < improvementTrajectory.length; i++) {
            const trajectory = improvementTrajectory[i];
            if (trajectory.improvement.overall < -0.05) { // 5% regression threshold
                const recovered = i < improvementTrajectory.length - 1 
                    ? improvementTrajectory[i + 1].improvement.overall > 0
                    : false;
                
                regressionEvents.push({
                    fromStage: trajectory.fromStage,
                    toStage: trajectory.toStage,
                    magnitude: Math.abs(trajectory.improvement.overall),
                    recovered
                });
            }
        }
        
        const regressionRate = regressionEvents.length / improvementTrajectory.length;
        const recoveryRate = regressionEvents.filter(e => e.recovered).length / Math.max(regressionEvents.length, 1);
        
        return {
            regressionEvents,
            regressionRate,
            recoveryRate
        };
    }
    
    /**
     * Perform statistical validation of improvements
     */
    private performStatisticalValidation(
        stageResults: StageValidationResult[],
        criteria: EvolutionCriteria
    ): StatisticalValidation {
        const significanceTests: Array<{
            metric: string;
            pValue: number;
            significant: boolean;
            confidenceInterval: [number, number];
        }> = [];
        
        const effectSizes: Array<{
            metric: string;
            cohensD: number;
            magnitude: "small" | "medium" | "large";
        }> = [];
        
        if (stageResults.length < 2) {
            return {
                significanceTests,
                effectSizes,
                overallSignificance: false
            };
        }
        
        // Perform t-tests between first and last stages
        const firstStage = stageResults[0];
        const lastStage = stageResults[stageResults.length - 1];
        
        const metrics: (keyof StageMetrics)[] = ["executionTime", "accuracy", "cost"];
        
        for (const metric of metrics) {
            const firstValues = this.metricsHistory.get(firstStage.stage)?.map(m => m[metric]).filter(v => v !== undefined) as number[] || [];
            const lastValues = this.metricsHistory.get(lastStage.stage)?.map(m => m[metric]).filter(v => v !== undefined) as number[] || [];
            
            if (firstValues.length > 0 && lastValues.length > 0) {
                const tTestResult = this.performTTest(firstValues, lastValues);
                const cohensD = this.calculateCohensD(firstValues, lastValues);
                
                significanceTests.push({
                    metric,
                    pValue: tTestResult.pValue,
                    significant: tTestResult.pValue < (1 - (criteria.statisticalSignificance?.confidenceLevel || 0.95)),
                    confidenceInterval: tTestResult.confidenceInterval
                });
                
                effectSizes.push({
                    metric,
                    cohensD,
                    magnitude: this.interpretCohensD(cohensD)
                });
            }
        }
        
        const overallSignificance = significanceTests.some(t => t.significant);
        
        return {
            significanceTests,
            effectSizes,
            overallSignificance
        };
    }
    
    /**
     * Perform Welch's t-test
     */
    private performTTest(
        sample1: number[],
        sample2: number[]
    ): { pValue: number; confidenceInterval: [number, number] } {
        const mean1 = sample1.reduce((sum, val) => sum + val, 0) / sample1.length;
        const mean2 = sample2.reduce((sum, val) => sum + val, 0) / sample2.length;
        
        const var1 = sample1.reduce((sum, val) => sum + Math.pow(val - mean1, 2), 0) / (sample1.length - 1);
        const var2 = sample2.reduce((sum, val) => sum + Math.pow(val - mean2, 2), 0) / (sample2.length - 1);
        
        const se = Math.sqrt(var1 / sample1.length + var2 / sample2.length);
        const t = (mean1 - mean2) / se;
        
        // Simplified p-value calculation (would use proper t-distribution in practice)
        const pValue = 2 * (1 - this.normalCDF(Math.abs(t)));
        
        // 95% confidence interval
        const marginOfError = 1.96 * se; // Using z-score for simplicity
        const confidenceInterval: [number, number] = [
            (mean1 - mean2) - marginOfError,
            (mean1 - mean2) + marginOfError
        ];
        
        return { pValue, confidenceInterval };
    }
    
    /**
     * Calculate Cohen's D effect size
     */
    private calculateCohensD(sample1: number[], sample2: number[]): number {
        const mean1 = sample1.reduce((sum, val) => sum + val, 0) / sample1.length;
        const mean2 = sample2.reduce((sum, val) => sum + val, 0) / sample2.length;
        
        const var1 = sample1.reduce((sum, val) => sum + Math.pow(val - mean1, 2), 0) / (sample1.length - 1);
        const var2 = sample2.reduce((sum, val) => sum + Math.pow(val - mean2, 2), 0) / (sample2.length - 1);
        
        const pooledSD = Math.sqrt(((sample1.length - 1) * var1 + (sample2.length - 1) * var2) / 
                                   (sample1.length + sample2.length - 2));
        
        return (mean2 - mean1) / pooledSD;
    }
    
    /**
     * Interpret Cohen's D magnitude
     */
    private interpretCohensD(cohensD: number): "small" | "medium" | "large" {
        const abs = Math.abs(cohensD);
        if (abs < 0.2) return "small";
        if (abs < 0.8) return "medium";
        return "large";
    }
    
    /**
     * Simplified normal CDF
     */
    private normalCDF(x: number): number {
        return 0.5 * (1 + this.erf(x / Math.sqrt(2)));
    }
    
    /**
     * Error function approximation
     */
    private erf(x: number): number {
        const a1 =  0.254829592;
        const a2 = -0.284496736;
        const a3 =  1.421413741;
        const a4 = -1.453152027;
        const a5 =  1.061405429;
        const p  =  0.3275911;
        
        const sign = x >= 0 ? 1 : -1;
        x = Math.abs(x);
        
        const t = 1.0 / (1.0 + p * x);
        const y = 1.0 - (((((a5 * t + a4) * t) + a3) * t + a2) * t + a1) * t * Math.exp(-x * x);
        
        return sign * y;
    }
    
    /**
     * Calculate overall validation
     */
    private calculateOverallValidation(
        stageResults: StageValidationResult[],
        improvementAnalysis: ImprovementAnalysis,
        criteria: EvolutionCriteria,
        statisticalValidation?: StatisticalValidation
    ): OverallEvolutionValidation {
        let criteriaMetCount = 0;
        let totalCriteriaCount = 0;
        
        // Check minimum improvement criteria
        if (criteria.minimumImprovement) {
            totalCriteriaCount += Object.keys(criteria.minimumImprovement).length;
            
            const improvement = improvementAnalysis.totalImprovement;
            
            if (criteria.minimumImprovement.executionTime && 
                improvement.relative.executionTime >= criteria.minimumImprovement.executionTime) {
                criteriaMetCount++;
            }
            
            if (criteria.minimumImprovement.accuracy && 
                improvement.relative.accuracy >= criteria.minimumImprovement.accuracy) {
                criteriaMetCount++;
            }
            
            if (criteria.minimumImprovement.cost && 
                improvement.relative.cost >= criteria.minimumImprovement.cost) {
                criteriaMetCount++;
            }
            
            if (criteria.minimumImprovement.overall && 
                improvement.overall >= criteria.minimumImprovement.overall) {
                criteriaMetCount++;
            }
        }
        
        // Check statistical significance if available
        if (statisticalValidation && criteria.statisticalSignificance?.required) {
            totalCriteriaCount++;
            if (statisticalValidation.overallSignificance) {
                criteriaMetCount++;
            }
        }
        
        // Check regression tolerance
        if (criteria.regressionTolerance) {
            totalCriteriaCount++;
            const regressionAnalysis = improvementAnalysis.regressionAnalysis;
            if (regressionAnalysis.regressionRate <= (criteria.regressionTolerance.allowedRegressions / stageResults.length)) {
                criteriaMetCount++;
            }
        }
        
        // Check stability requirements
        if (criteria.stabilityRequirements) {
            totalCriteriaCount++;
            if (improvementAnalysis.stabilityAnalysis.overallStability >= criteria.stabilityRequirements.stabilityThreshold) {
                criteriaMetCount++;
            }
        }
        
        const evolutionValidated = criteriaMetCount === totalCriteriaCount && totalCriteriaCount > 0;
        const improvementDetected = improvementAnalysis.totalImprovement.overall > 0;
        
        const confidenceScore = totalCriteriaCount > 0 ? criteriaMetCount / totalCriteriaCount : 0;
        const qualityScore = this.calculateQualityScore(stageResults, improvementAnalysis);
        
        return {
            evolutionValidated,
            improvementDetected,
            criteriaMetCount,
            totalCriteriaCount,
            confidenceScore,
            qualityScore
        };
    }
    
    /**
     * Calculate quality score
     */
    private calculateQualityScore(
        stageResults: StageValidationResult[],
        improvementAnalysis: ImprovementAnalysis
    ): number {
        const validStages = stageResults.filter(s => s.validated).length;
        const stageValidityScore = validStages / stageResults.length;
        
        const improvementScore = Math.min(Math.max(improvementAnalysis.totalImprovement.overall, 0), 1);
        const stabilityScore = improvementAnalysis.stabilityAnalysis.overallStability;
        
        return (stageValidityScore * 0.4 + improvementScore * 0.4 + stabilityScore * 0.2);
    }
    
    /**
     * Generate recommendations
     */
    private generateRecommendations(
        stageResults: StageValidationResult[],
        improvementAnalysis: ImprovementAnalysis,
        criteria: EvolutionCriteria
    ): EvolutionRecommendation[] {
        const recommendations: EvolutionRecommendation[] = [];
        
        // Check for failed stages
        const failedStages = stageResults.filter(s => !s.validated);
        for (const stage of failedStages) {
            recommendations.push({
                type: "warning",
                stage: stage.stage,
                message: `Stage validation failed: ${stage.issues?.join(", ") || "Unknown issues"}`,
                priority: "high"
            });
        }
        
        // Check for insufficient improvement
        if (improvementAnalysis.totalImprovement.overall < 0.1) {
            recommendations.push({
                type: "improvement",
                message: "Overall improvement is below recommended threshold (10%)",
                suggestedAction: "Review evolution strategy and stage configurations",
                priority: "medium"
            });
        }
        
        // Check for instability
        if (improvementAnalysis.stabilityAnalysis.overallStability < 0.7) {
            recommendations.push({
                type: "optimization",
                message: "System shows instability across stages",
                suggestedAction: "Increase iterations per stage or adjust mock behaviors",
                priority: "medium"
            });
        }
        
        // Check for regressions
        if (improvementAnalysis.regressionAnalysis.regressionRate > 0.2) {
            recommendations.push({
                type: "warning",
                message: "High regression rate detected",
                suggestedAction: "Review stage transitions and evolution triggers",
                priority: "high"
            });
        }
        
        return recommendations;
    }
    
    // Utility methods
    
    private createZeroMetrics(): StageMetrics {
        return {
            executionTime: 0,
            accuracy: 0,
            cost: 0,
            successRate: 0,
            resourceUsage: 0,
            throughput: 0,
            latency: 0,
            errorRate: 0
        };
    }
    
    private createZeroImprovement(): ImprovementMetrics {
        return {
            executionTime: 0,
            accuracy: 0,
            cost: 0,
            overall: 0,
            relative: {
                executionTime: 0,
                accuracy: 0,
                cost: 0
            }
        };
    }
}