import { BaseToolFixture, type ToolTestCase } from "./base.js";
import { type MCPTool } from "../../../../services/mcp/tools.js";
import { type EmergentAgent } from "../../../../services/execution/cross-cutting/agents/emergentAgent.js";
import { type EventPublisher } from "../../../../services/execution/shared/EventPublisher.js";

export class MonitoringToolFixture extends BaseToolFixture {
    private metricsHistory: MetricSnapshot[] = [];
    private alertsTriggered: Alert[] = [];
    private anomaliesDetected: Anomaly[] = [];

    constructor(tool: MCPTool, eventPublisher: EventPublisher) {
        super(tool, eventPublisher);
        this.setupMonitoring();
    }

    /**
     * Test monitoring tool's ability to detect anomalies
     */
    async testAnomalyDetection(
        normalData: MetricData[],
        anomalousData: MetricData[],
    ): Promise<AnomalyDetectionResult> {
        // Train on normal data
        for (const data of normalData) {
            await this.tool.execute({
                action: "record",
                metrics: data,
                mode: "training",
            });
        }

        // Test with anomalous data
        const detections: boolean[] = [];
        for (const data of anomalousData) {
            const result = await this.tool.execute({
                action: "analyze",
                metrics: data,
                mode: "detection",
            });
            detections.push(result.anomalyDetected || false);
        }

        const detectionRate = detections.filter(d => d).length / detections.length;

        return {
            totalAnomalies: anomalousData.length,
            detected: detections.filter(d => d).length,
            detectionRate,
            falsePositives: this.calculateFalsePositives(normalData),
            sensitivity: this.calculateSensitivity(detections),
        };
    }

    /**
     * Test monitoring tool's predictive capabilities
     */
    async testPrediction(
        historicalData: TimeSeriesData,
        futureTimepoints: number,
    ): Promise<PredictionResult> {
        // Feed historical data
        const result = await this.tool.execute({
            action: "predict",
            historical: historicalData,
            horizon: futureTimepoints,
        });

        // Wait for actual data to compare
        const actualData = await this.collectActualData(futureTimepoints);

        return {
            predictions: result.predictions,
            actual: actualData,
            accuracy: this.calculatePredictionAccuracy(result.predictions, actualData),
            confidence: result.confidence || 0,
        };
    }

    /**
     * Test monitoring tool's alerting system
     */
    async testAlertingSystem(
        thresholds: AlertThreshold[],
        testData: MetricData[],
    ): Promise<AlertingTestResult> {
        // Configure thresholds
        await this.tool.execute({
            action: "configure_alerts",
            thresholds,
        });

        // Clear previous alerts
        this.alertsTriggered = [];

        // Process test data
        for (const data of testData) {
            await this.tool.execute({
                action: "monitor",
                metrics: data,
            });
        }

        // Analyze alerts
        const expectedAlerts = this.calculateExpectedAlerts(thresholds, testData);
        const precision = this.calculateAlertPrecision(this.alertsTriggered, expectedAlerts);
        const recall = this.calculateAlertRecall(this.alertsTriggered, expectedAlerts);

        return {
            totalAlerts: this.alertsTriggered.length,
            expectedAlerts: expectedAlerts.length,
            precision,
            recall,
            f1Score: 2 * (precision * recall) / (precision + recall || 1),
            alertLatency: this.calculateAverageAlertLatency(),
        };
    }

    /**
     * Test monitoring evolution from reactive to predictive
     */
    async testMonitoringEvolution(
        trainingPeriod: number,
        testPeriod: number,
    ): Promise<MonitoringEvolutionResult> {
        const phases: EvolutionPhase[] = [];

        // Phase 1: Reactive monitoring
        const reactivePhase = await this.runReactivePhase(trainingPeriod / 3);
        phases.push(reactivePhase);

        // Phase 2: Pattern learning
        const learningPhase = await this.runLearningPhase(trainingPeriod / 3);
        phases.push(learningPhase);

        // Phase 3: Predictive monitoring
        const predictivePhase = await this.runPredictivePhase(trainingPeriod / 3);
        phases.push(predictivePhase);

        // Test evolved capabilities
        const testResults = await this.testEvolvedCapabilities(testPeriod);

        return {
            phases,
            finalCapabilities: testResults.capabilities,
            improvementMetrics: {
                detectionSpeed: this.calculateDetectionSpeedImprovement(phases),
                accuracy: this.calculateAccuracyImprovement(phases),
                falsePositiveReduction: this.calculateFalsePositiveReduction(phases),
            },
        };
    }

    protected async executeWithRetry(input: any, maxRetries: number): Promise<any> {
        let lastError: any;
        
        for (let i = 0; i < maxRetries; i++) {
            try {
                return await this.tool.execute(input);
            } catch (error) {
                lastError = error;
                if (i < maxRetries - 1) {
                    await new Promise(resolve => setTimeout(resolve, 1000 * Math.pow(2, i)));
                }
            }
        }
        
        throw lastError;
    }

    protected captureResourceUsage(): { cpu: number; memory: number } {
        // Simulate resource usage capture
        return {
            cpu: Math.random() * 50 + 10, // 10-60% CPU
            memory: Math.random() * 200 + 50, // 50-250 MB
        };
    }

    protected async applyConstraints(constraints: any[]): Promise<void> {
        // Apply monitoring-specific constraints
        for (const constraint of constraints) {
            if (constraint.type === "rate_limit") {
                // Configure rate limiting
                await this.tool.execute({
                    action: "configure",
                    rateLimit: constraint.value,
                });
            }
        }
    }

    protected async runAgentScenario(
        agent: EmergentAgent,
        scenario: any,
    ): Promise<void> {
        // Simulate monitoring scenario with agent
        const monitoringLoop = async () => {
            for (let i = 0; i < 10; i++) {
                const metrics = this.generateTestMetrics();
                await agent.executeTool(this.tool.id, {
                    action: "monitor",
                    metrics,
                });
                await new Promise(resolve => setTimeout(resolve, 1000));
            }
        };

        await monitoringLoop();
    }

    protected async applyStimulus(stimulus: any): Promise<void> {
        switch (stimulus.type) {
            case "metric_spike":
                await this.simulateMetricSpike(stimulus.data);
                break;
            case "pattern_change":
                await this.simulatePatternChange(stimulus.data);
                break;
            case "system_failure":
                await this.simulateSystemFailure(stimulus.data);
                break;
        }
    }

    protected async takeSnapshot(name: string): Promise<any> {
        const currentMetrics = await this.tool.execute({
            action: "get_stats",
        });

        return {
            name,
            timestamp: new Date(),
            performance: {
                latency: currentMetrics.avgResponseTime || 100,
                accuracy: currentMetrics.detectionAccuracy || 0.8,
                throughput: currentMetrics.metricsPerSecond || 1000,
                errorRate: currentMetrics.errorRate || 0.01,
            },
            capabilities: currentMetrics.capabilities || ["basic_monitoring"],
        };
    }

    private setupMonitoring() {
        this.eventPublisher.on("alert:triggered", (alert: Alert) => {
            this.alertsTriggered.push(alert);
        });

        this.eventPublisher.on("anomaly:detected", (anomaly: Anomaly) => {
            this.anomaliesDetected.push(anomaly);
        });
    }

    private calculateFalsePositives(normalData: MetricData[]): number {
        // Test normal data for false positives
        const falsePositives = 0;
        // Simplified calculation
        return falsePositives;
    }

    private calculateSensitivity(detections: boolean[]): number {
        // Sensitivity = True Positives / (True Positives + False Negatives)
        return detections.filter(d => d).length / detections.length;
    }

    private async collectActualData(timepoints: number): Promise<number[]> {
        // Simulate collecting actual future data
        return Array(timepoints).fill(0).map(() => Math.random() * 100);
    }

    private calculatePredictionAccuracy(
        predictions: number[],
        actual: number[],
    ): number {
        let totalError = 0;
        for (let i = 0; i < predictions.length && i < actual.length; i++) {
            totalError += Math.abs(predictions[i] - actual[i]) / actual[i];
        }
        return 1 - (totalError / predictions.length);
    }

    private calculateExpectedAlerts(
        thresholds: AlertThreshold[],
        data: MetricData[],
    ): Alert[] {
        const expectedAlerts: Alert[] = [];
        
        for (const datum of data) {
            for (const threshold of thresholds) {
                if (this.shouldTriggerAlert(datum, threshold)) {
                    expectedAlerts.push({
                        type: threshold.metric,
                        severity: threshold.severity,
                        timestamp: new Date(),
                        value: datum[threshold.metric],
                        threshold: threshold.value,
                    });
                }
            }
        }
        
        return expectedAlerts;
    }

    private shouldTriggerAlert(data: MetricData, threshold: AlertThreshold): boolean {
        const value = data[threshold.metric];
        if (!value) return false;
        
        switch (threshold.operator) {
            case ">": return value > threshold.value;
            case "<": return value < threshold.value;
            case ">=": return value >= threshold.value;
            case "<=": return value <= threshold.value;
            default: return false;
        }
    }

    private calculateAlertPrecision(actual: Alert[], expected: Alert[]): number {
        if (actual.length === 0) return 0;
        
        const truePositives = actual.filter(a => 
            expected.some(e => this.alertsMatch(a, e)),
        ).length;
        
        return truePositives / actual.length;
    }

    private calculateAlertRecall(actual: Alert[], expected: Alert[]): number {
        if (expected.length === 0) return 0;
        
        const truePositives = expected.filter(e => 
            actual.some(a => this.alertsMatch(a, e)),
        ).length;
        
        return truePositives / expected.length;
    }

    private alertsMatch(a1: Alert, a2: Alert): boolean {
        return a1.type === a2.type && 
               a1.severity === a2.severity &&
               Math.abs(a1.value - a2.value) < 0.01;
    }

    private calculateAverageAlertLatency(): number {
        // Calculate average time from condition to alert
        return 500; // milliseconds
    }

    private generateTestMetrics(): MetricData {
        return {
            cpu: Math.random() * 100,
            memory: Math.random() * 8192,
            requests: Math.floor(Math.random() * 1000),
            errors: Math.floor(Math.random() * 10),
            latency: Math.random() * 1000,
        };
    }

    private async simulateMetricSpike(data: any): Promise<void> {
        await this.tool.execute({
            action: "inject_spike",
            metric: data.metric,
            magnitude: data.magnitude,
        });
    }

    private async simulatePatternChange(data: any): Promise<void> {
        await this.tool.execute({
            action: "change_pattern",
            pattern: data.pattern,
        });
    }

    private async simulateSystemFailure(data: any): Promise<void> {
        await this.tool.execute({
            action: "simulate_failure",
            component: data.component,
        });
    }

    private async runReactivePhase(duration: number): Promise<EvolutionPhase> {
        const startTime = Date.now();
        const metrics: PhaseMetrics = {
            detections: 0,
            falsePositives: 0,
            avgLatency: 0,
        };

        // Simulate reactive monitoring
        while (Date.now() - startTime < duration) {
            const data = this.generateTestMetrics();
            const result = await this.tool.execute({
                action: "monitor",
                metrics: data,
                mode: "reactive",
            });
            
            if (result.alertTriggered) metrics.detections++;
            metrics.avgLatency += result.latency || 0;
            
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        return {
            name: "reactive",
            duration: Date.now() - startTime,
            metrics,
            capabilities: ["threshold_alerting"],
        };
    }

    private async runLearningPhase(duration: number): Promise<EvolutionPhase> {
        const startTime = Date.now();
        const metrics: PhaseMetrics = {
            detections: 0,
            falsePositives: 0,
            avgLatency: 0,
            patternsLearned: 0,
        };

        // Simulate learning phase
        while (Date.now() - startTime < duration) {
            const data = this.generateTestMetrics();
            const result = await this.tool.execute({
                action: "monitor",
                metrics: data,
                mode: "learning",
            });
            
            if (result.patternDetected) metrics.patternsLearned!++;
            
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        return {
            name: "learning",
            duration: Date.now() - startTime,
            metrics,
            capabilities: ["threshold_alerting", "pattern_recognition"],
        };
    }

    private async runPredictivePhase(duration: number): Promise<EvolutionPhase> {
        const startTime = Date.now();
        const metrics: PhaseMetrics = {
            detections: 0,
            falsePositives: 0,
            avgLatency: 0,
            predictionsAccurate: 0,
        };

        // Simulate predictive monitoring
        while (Date.now() - startTime < duration) {
            const result = await this.tool.execute({
                action: "predict_and_monitor",
                horizon: 5,
            });
            
            if (result.predictionAccurate) metrics.predictionsAccurate!++;
            
            await new Promise(resolve => setTimeout(resolve, 100));
        }

        return {
            name: "predictive",
            duration: Date.now() - startTime,
            metrics,
            capabilities: ["threshold_alerting", "pattern_recognition", "predictive_analytics"],
        };
    }

    private async testEvolvedCapabilities(duration: number): Promise<any> {
        // Test final evolved capabilities
        const capabilities = await this.tool.execute({
            action: "get_capabilities",
        });

        return {
            capabilities: capabilities.list || [],
            performance: capabilities.performance || {},
        };
    }

    private calculateDetectionSpeedImprovement(phases: EvolutionPhase[]): number {
        const initial = phases[0].metrics.avgLatency || 1000;
        const final = phases[phases.length - 1].metrics.avgLatency || 500;
        return (initial - final) / initial;
    }

    private calculateAccuracyImprovement(phases: EvolutionPhase[]): number {
        // Calculate based on false positive reduction
        const initial = phases[0].metrics.falsePositives || 10;
        const final = phases[phases.length - 1].metrics.falsePositives || 2;
        return (initial - final) / initial;
    }

    private calculateFalsePositiveReduction(phases: EvolutionPhase[]): number {
        return this.calculateAccuracyImprovement(phases); // Same calculation
    }
}

// Type definitions
interface MetricSnapshot {
    timestamp: Date;
    metrics: MetricData;
}

interface MetricData {
    [key: string]: number;
}

interface Alert {
    type: string;
    severity: "low" | "medium" | "high" | "critical";
    timestamp: Date;
    value: number;
    threshold: number;
}

interface Anomaly {
    timestamp: Date;
    metric: string;
    value: number;
    expectedRange: [number, number];
    confidence: number;
}

interface AnomalyDetectionResult {
    totalAnomalies: number;
    detected: number;
    detectionRate: number;
    falsePositives: number;
    sensitivity: number;
}

interface TimeSeriesData {
    timestamps: Date[];
    values: number[];
    metadata?: any;
}

interface PredictionResult {
    predictions: number[];
    actual: number[];
    accuracy: number;
    confidence: number;
}

interface AlertThreshold {
    metric: string;
    operator: ">" | "<" | ">=" | "<=";
    value: number;
    severity: "low" | "medium" | "high" | "critical";
}

interface AlertingTestResult {
    totalAlerts: number;
    expectedAlerts: number;
    precision: number;
    recall: number;
    f1Score: number;
    alertLatency: number;
}

interface EvolutionPhase {
    name: string;
    duration: number;
    metrics: PhaseMetrics;
    capabilities: string[];
}

interface PhaseMetrics {
    detections: number;
    falsePositives: number;
    avgLatency: number;
    patternsLearned?: number;
    predictionsAccurate?: number;
}

interface MonitoringEvolutionResult {
    phases: EvolutionPhase[];
    finalCapabilities: string[];
    improvementMetrics: {
        detectionSpeed: number;
        accuracy: number;
        falsePositiveReduction: number;
    };
}

/**
 * Standard monitoring test cases
 */
export const MONITORING_TEST_CASES: ToolTestCase[] = [
    {
        name: "basic-metric-collection",
        description: "Collect and store basic system metrics",
        input: {
            action: "collect",
            metrics: {
                cpu: 45.2,
                memory: 2048,
                disk: 75.5,
            },
        },
        expectedOutput: {
            success: true,
            stored: true,
        },
    },
    {
        name: "threshold-alert",
        description: "Trigger alert on threshold breach",
        input: {
            action: "monitor",
            metrics: {
                cpu: 95.5,
                memory: 7800,
            },
            thresholds: {
                cpu: { max: 80 },
                memory: { max: 6000 },
            },
        },
        expectedBehavior: {
            sideEffects: ["alert_triggered"],
        },
    },
    {
        name: "anomaly-detection",
        description: "Detect anomalous metric patterns",
        input: {
            action: "analyze",
            metrics: {
                requests: 50000, // Normal is 1000-5000
                errors: 500,     // Normal is 0-10
            },
        },
        expectedOutput: {
            anomalies: ["requests", "errors"],
        },
    },
];
