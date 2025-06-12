import { type Logger } from "winston";
import { getExecutionArchitecture } from "./executionArchitecture.js";
import { type RollingHistoryAdapter as RollingHistory } from "../monitoring/adapters/RollingHistoryAdapter.js";

/**
 * Example: Emergent Monitoring Capabilities
 * 
 * This example demonstrates how monitoring capabilities naturally emerge
 * from the telemetry shim and rolling history integration across all tiers.
 * 
 * Key patterns that emerge:
 * 1. Performance bottleneck detection
 * 2. Resource usage patterns
 * 3. Strategy effectiveness
 * 4. Error clustering
 * 5. Swarm behavior analysis
 */

export class EmergentMonitoringExample {
    private readonly logger: Logger;
    private history: RollingHistory | null = null;
    
    constructor(logger: Logger) {
        this.logger = logger;
    }
    
    /**
     * Initialize monitoring with execution architecture
     */
    async initialize(): Promise<void> {
        const architecture = await getExecutionArchitecture({
            telemetryEnabled: true,
            historyEnabled: true,
            historyBufferSize: 10000,
        });
        
        this.history = architecture.getRollingHistory();
        if (!this.history) {
            throw new Error("Rolling history not available");
        }
        
        // Start pattern detection loop
        this.startPatternDetection();
    }
    
    /**
     * Start continuous pattern detection
     */
    private startPatternDetection(): void {
        setInterval(() => {
            this.detectEmergentPatterns();
        }, 60000); // Every minute
    }
    
    /**
     * Detect emergent patterns from rolling history
     */
    private async detectEmergentPatterns(): Promise<void> {
        if (!this.history) return;
        
        // 1. Detect performance bottlenecks
        const performanceBottlenecks = this.detectBottlenecks();
        if (performanceBottlenecks.length > 0) {
            this.logger.warn("[EmergentMonitoring] Performance bottlenecks detected", {
                bottlenecks: performanceBottlenecks,
            });
        }
        
        // 2. Analyze resource usage patterns
        const resourcePatterns = this.analyzeResourceUsage();
        if (resourcePatterns.hasAnomalies) {
            this.logger.warn("[EmergentMonitoring] Resource usage anomalies", {
                patterns: resourcePatterns,
            });
        }
        
        // 3. Evaluate strategy effectiveness
        const strategyMetrics = this.evaluateStrategies();
        this.logger.info("[EmergentMonitoring] Strategy effectiveness", {
            metrics: strategyMetrics,
        });
        
        // 4. Detect error clusters
        const errorClusters = this.detectErrorClusters();
        if (errorClusters.length > 0) {
            this.logger.error("[EmergentMonitoring] Error clusters detected", {
                clusters: errorClusters,
            });
        }
        
        // 5. Analyze swarm coordination patterns
        const swarmPatterns = this.analyzeSwarmBehavior();
        this.logger.info("[EmergentMonitoring] Swarm behavior analysis", {
            patterns: swarmPatterns,
        });
    }
    
    /**
     * Detect performance bottlenecks from execution patterns
     */
    private detectBottlenecks(): Array<{
        component: string;
        avgDuration: number;
        p95Duration: number;
        frequency: number;
    }> {
        const bottlenecks: Array<any> = [];
        
        // Analyze step completion events
        const stepEvents = this.history!.getEventsByType(/\.step\.completed$/);
        
        // Group by component and calculate metrics
        const componentMetrics = new Map<string, number[]>();
        
        for (const event of stepEvents) {
            const component = event.component;
            const duration = event.data.duration as number || 0;
            
            if (!componentMetrics.has(component)) {
                componentMetrics.set(component, []);
            }
            componentMetrics.get(component)!.push(duration);
        }
        
        // Calculate statistics and identify bottlenecks
        for (const [component, durations] of componentMetrics) {
            if (durations.length === 0) continue;
            
            const sorted = durations.sort((a, b) => a - b);
            const avg = durations.reduce((a, b) => a + b, 0) / durations.length;
            const p95Index = Math.floor(durations.length * 0.95);
            const p95 = sorted[p95Index] || sorted[sorted.length - 1];
            
            // Bottleneck if p95 > 30 seconds
            if (p95 > 30000) {
                bottlenecks.push({
                    component,
                    avgDuration: avg,
                    p95Duration: p95,
                    frequency: durations.length,
                });
            }
        }
        
        return bottlenecks;
    }
    
    /**
     * Analyze resource usage patterns
     */
    private analyzeResourceUsage(): {
        hasAnomalies: boolean;
        avgCreditsPerStep: number;
        creditSpikes: number[];
        resourceExhaustion: boolean;
    } {
        const resourceEvents = this.history!.getEventsByType(/resource/);
        
        let totalCredits = 0;
        let stepCount = 0;
        const creditSpikes: number[] = [];
        let exhaustionCount = 0;
        
        for (const event of resourceEvents) {
            if (event.type.includes('allocated')) {
                const credits = event.data.credits as number || 0;
                totalCredits += credits;
                stepCount++;
                
                // Spike if > 1000 credits
                if (credits > 1000) {
                    creditSpikes.push(credits);
                }
            } else if (event.type.includes('exceeded') || event.type.includes('quota_abort')) {
                exhaustionCount++;
            }
        }
        
        const avgCreditsPerStep = stepCount > 0 ? totalCredits / stepCount : 0;
        
        return {
            hasAnomalies: creditSpikes.length > 0 || exhaustionCount > 0,
            avgCreditsPerStep,
            creditSpikes,
            resourceExhaustion: exhaustionCount > 0,
        };
    }
    
    /**
     * Evaluate strategy effectiveness
     */
    private evaluateStrategies(): Record<string, {
        successRate: number;
        avgDuration: number;
        usage: number;
    }> {
        const strategyEvents = this.history!.getEventsByType('strategy_selected');
        const completionEvents = this.history!.getEventsByType(/\.completed$/);
        
        const strategyMetrics = new Map<string, {
            total: number;
            successful: number;
            durations: number[];
        }>();
        
        // Track strategy selections
        for (const event of strategyEvents) {
            const strategy = event.data.selected as string;
            if (!strategyMetrics.has(strategy)) {
                strategyMetrics.set(strategy, {
                    total: 0,
                    successful: 0,
                    durations: [],
                });
            }
            strategyMetrics.get(strategy)!.total++;
        }
        
        // Track completions
        for (const event of completionEvents) {
            const strategy = event.data.strategy as string;
            if (strategy && strategyMetrics.has(strategy)) {
                const metrics = strategyMetrics.get(strategy)!;
                metrics.successful++;
                if (event.data.duration) {
                    metrics.durations.push(event.data.duration as number);
                }
            }
        }
        
        // Calculate final metrics
        const result: Record<string, any> = {};
        for (const [strategy, metrics] of strategyMetrics) {
            const avgDuration = metrics.durations.length > 0
                ? metrics.durations.reduce((a, b) => a + b, 0) / metrics.durations.length
                : 0;
                
            result[strategy] = {
                successRate: metrics.total > 0 ? metrics.successful / metrics.total : 0,
                avgDuration,
                usage: metrics.total,
            };
        }
        
        return result;
    }
    
    /**
     * Detect error clusters
     */
    private detectErrorClusters(): Array<{
        errorType: string;
        count: number;
        timeWindow: number;
        components: string[];
    }> {
        const errorEvents = this.history!.getEventsByType(/error|failed/);
        const clusters: Array<any> = [];
        
        // Group errors by type and time window (5 minutes)
        const timeWindow = 300000; // 5 minutes
        const errorGroups = new Map<string, Array<{
            timestamp: number;
            component: string;
        }>>();
        
        for (const event of errorEvents) {
            const errorType = event.data.errorType as string || 'unknown';
            const timestamp = event.timestamp.getTime();
            
            if (!errorGroups.has(errorType)) {
                errorGroups.set(errorType, []);
            }
            
            errorGroups.get(errorType)!.push({
                timestamp,
                component: event.component,
            });
        }
        
        // Detect clusters
        for (const [errorType, errors] of errorGroups) {
            // Sort by timestamp
            errors.sort((a, b) => a.timestamp - b.timestamp);
            
            // Find clusters
            let clusterStart = errors[0].timestamp;
            let clusterErrors = [errors[0]];
            
            for (let i = 1; i < errors.length; i++) {
                if (errors[i].timestamp - clusterStart <= timeWindow) {
                    clusterErrors.push(errors[i]);
                } else {
                    // End of cluster
                    if (clusterErrors.length >= 3) {
                        clusters.push({
                            errorType,
                            count: clusterErrors.length,
                            timeWindow: clusterErrors[clusterErrors.length - 1].timestamp - clusterStart,
                            components: [...new Set(clusterErrors.map(e => e.component))],
                        });
                    }
                    
                    // Start new cluster
                    clusterStart = errors[i].timestamp;
                    clusterErrors = [errors[i]];
                }
            }
            
            // Check final cluster
            if (clusterErrors.length >= 3) {
                clusters.push({
                    errorType,
                    count: clusterErrors.length,
                    timeWindow: clusterErrors[clusterErrors.length - 1].timestamp - clusterStart,
                    components: [...new Set(clusterErrors.map(e => e.component))],
                });
            }
        }
        
        return clusters;
    }
    
    /**
     * Analyze swarm behavior patterns
     */
    private analyzeSwarmBehavior(): {
        activeSwarms: number;
        avgDecisionsPerCycle: number;
        consensusRate: number;
        adaptationFrequency: number;
    } {
        const swarmEvents = this.history!.getEventsByTier('tier1');
        
        const activeSwarms = new Set<string>();
        let totalDecisions = 0;
        let totalCycles = 0;
        let consensusAchieved = 0;
        let adaptations = 0;
        
        for (const event of swarmEvents) {
            if (event.data.swarmId) {
                activeSwarms.add(event.data.swarmId as string);
            }
            
            if (event.type.includes('decisions.made')) {
                totalDecisions += event.data.totalDecisions as number || 0;
                totalCycles++;
                
                const approved = event.data.approvedDecisions as number || 0;
                const total = event.data.totalDecisions as number || 1;
                if (approved / total >= 0.7) {
                    consensusAchieved++;
                }
            } else if (event.type.includes('reflection.completed')) {
                adaptations += event.data.adaptationCount as number || 0;
            }
        }
        
        return {
            activeSwarms: activeSwarms.size,
            avgDecisionsPerCycle: totalCycles > 0 ? totalDecisions / totalCycles : 0,
            consensusRate: totalCycles > 0 ? consensusAchieved / totalCycles : 0,
            adaptationFrequency: swarmEvents.length > 0 ? adaptations / swarmEvents.length : 0,
        };
    }
    
    /**
     * Generate monitoring report
     */
    async generateReport(): Promise<{
        timestamp: Date;
        patterns: any;
        recommendations: string[];
    }> {
        const patterns = this.history!.detectPatterns(3600000); // Last hour
        
        const recommendations: string[] = [];
        
        // Generate recommendations based on patterns
        if (patterns.eventsPerMinute > 1000) {
            recommendations.push("High event rate detected. Consider scaling infrastructure.");
        }
        
        if (patterns.hasHighActivity) {
            recommendations.push("System under high load. Monitor resource usage closely.");
        }
        
        // Check tier distribution
        const tierDist = patterns.tierDistribution;
        if (tierDist.tier3 > tierDist.tier1 * 10) {
            recommendations.push("Tier 3 heavily loaded. Consider optimizing routine complexity.");
        }
        
        return {
            timestamp: new Date(),
            patterns,
            recommendations,
        };
    }
}

/**
 * Example usage
 */
export async function runMonitoringExample(): Promise<void> {
    const logger = console as any;
    const monitor = new EmergentMonitoringExample(logger);
    
    await monitor.initialize();
    
    // Generate initial report
    const report = await monitor.generateReport();
    console.log("Monitoring Report:", JSON.stringify(report, null, 2));
}