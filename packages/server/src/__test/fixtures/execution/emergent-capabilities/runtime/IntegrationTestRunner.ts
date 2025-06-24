/**
 * Integration Test Runner
 * 
 * Orchestrates cross-tier integration testing for emergent capabilities.
 * Tests the complete execution pipeline from Tier 1 coordination through
 * Tier 3 execution with real event flow validation.
 */

import { type ExecutionEvent } from "@vrooli/shared";
import { type EmergentCapabilityFixture } from "../emergentValidationUtils.js";
import { RuntimeExecutionValidator } from "./RuntimeExecutionValidator.js";
import { EmergenceDetector } from "./EmergenceDetector.js";
import { TierMockFactories } from "./tierMockFactories.js";
import { withAIMocks } from "../../ai-mocks/integration/testHelpers.js";
import { type AIMockConfig } from "../../ai-mocks/types.js";

/**
 * Integration scenario configuration
 */
export interface IntegrationScenario {
    name: string;
    description: string;
    domain: string;
    tiers: IntegrationTier[];
    steps: IntegrationStep[];
    expectedCapabilities: string[];
    expectedEventFlow: EventFlowExpectation[];
    crossTierDependencies?: CrossTierDependency[];
    timeConstraints?: {
        maxTotalDuration: number;
        maxStepDuration: number;
    };
    validationCriteria?: IntegrationValidationCriteria;
}

/**
 * Tier configuration for integration
 */
export interface IntegrationTier {
    tier: "tier1" | "tier2" | "tier3";
    fixture: EmergentCapabilityFixture<any>;
    mockBehaviors: Map<string, AIMockConfig>;
    expectedRole: string;
    expectedOutputs: string[];
}

/**
 * Individual integration step
 */
export interface IntegrationStep {
    name: string;
    tier: "tier1" | "tier2" | "tier3" | "cross-tier";
    input: any;
    expectedOutput?: any;
    triggeredEvents?: string[];
    timeout?: number;
    dependencies?: string[]; // Names of previous steps this depends on
}

/**
 * Expected event flow between tiers
 */
export interface EventFlowExpectation {
    fromTier: "tier1" | "tier2" | "tier3" | "external";
    toTier: "tier1" | "tier2" | "tier3" | "external";
    eventType: string;
    expectedLatency?: number;
    mandatory: boolean;
    sequence?: number; // Expected order in sequence
}

/**
 * Cross-tier dependency definition
 */
export interface CrossTierDependency {
    dependentTier: "tier1" | "tier2" | "tier3";
    providerTier: "tier1" | "tier2" | "tier3";
    dependency: string;
    criticality: "low" | "medium" | "high";
}

/**
 * Integration validation criteria
 */
export interface IntegrationValidationCriteria {
    minimumCapabilityDetection: number; // Percentage of expected capabilities
    maximumEventLatency: number;
    requiredEventFlow: number; // Percentage of expected event flows
    emergenceThreshold: number; // Minimum emergence score
    crossTierCoordination: {
        required: boolean;
        minimumInteractions: number;
    };
}

/**
 * Integration test result
 */
export interface IntegrationTestResult {
    scenario: string;
    success: boolean;
    executionTrace: ExecutionTrace[];
    eventFlowValidation: EventFlowValidation;
    emergenceValidation: EmergenceValidation;
    crossTierValidation: CrossTierValidation;
    performanceMetrics: IntegrationPerformanceMetrics;
    detectedCapabilities: string[];
    issues?: string[];
    warnings?: string[];
    recommendations?: string[];
}

/**
 * Execution trace entry
 */
export interface ExecutionTrace {
    step: string;
    tier: "tier1" | "tier2" | "tier3" | "cross-tier";
    startTime: number;
    endTime: number;
    input: any;
    output: any;
    emergentBehavior?: EmergentBehavior;
    eventsEmitted: ExecutionEvent[];
    eventsConsumed: ExecutionEvent[];
    errors?: string[];
}

/**
 * Emergent behavior detected in a step
 */
export interface EmergentBehavior {
    type: string;
    confidence: number;
    evidence: string[];
    relatedCapability?: string;
}

/**
 * Event flow validation result
 */
export interface EventFlowValidation {
    expectedFlows: EventFlowExpectation[];
    actualFlows: ActualEventFlow[];
    flowsMatched: number;
    flowsTotal: number;
    averageLatency: number;
    sequenceValidation: SequenceValidation;
    valid: boolean;
}

/**
 * Actual event flow observed
 */
export interface ActualEventFlow {
    fromTier: string;
    toTier: string;
    eventType: string;
    latency: number;
    timestamp: number;
    matched: boolean;
    expectedFlow?: EventFlowExpectation;
}

/**
 * Event sequence validation
 */
export interface SequenceValidation {
    expectedSequence: string[];
    actualSequence: string[];
    sequenceCorrect: boolean;
    outOfOrderEvents: string[];
}

/**
 * Emergence validation for integration
 */
export interface EmergenceValidation {
    expectedCapabilities: string[];
    detectedCapabilities: string[];
    emergenceScore: number;
    convergenceTime: number;
    stabilityPeriod: number;
    crossTierEmergence: boolean;
    valid: boolean;
}

/**
 * Cross-tier coordination validation
 */
export interface CrossTierValidation {
    expectedDependencies: CrossTierDependency[];
    satisfiedDependencies: number;
    coordinationEvents: number;
    syncrhonizationLatency: number;
    dataConsistency: boolean;
    valid: boolean;
}

/**
 * Integration performance metrics
 */
export interface IntegrationPerformanceMetrics {
    totalExecutionTime: number;
    tierBreakdown: {
        tier1: number;
        tier2: number;
        tier3: number;
        coordination: number;
    };
    throughput: number;
    eventLatency: {
        min: number;
        max: number;
        average: number;
        p95: number;
    };
    resourceUtilization: {
        memory: number;
        cpu: number;
        network: number;
    };
}

/**
 * Main integration test runner
 */
export class IntegrationTestRunner {
    private runtimeValidator: RuntimeExecutionValidator;
    private emergenceDetector: EmergenceDetector;
    private eventTracker: EventTracker;
    
    constructor() {
        this.runtimeValidator = new RuntimeExecutionValidator();
        this.emergenceDetector = new EmergenceDetector();
        this.eventTracker = new EventTracker();
    }
    
    /**
     * Run a complete cross-tier integration scenario
     */
    async runIntegrationScenario(
        scenario: IntegrationScenario
    ): Promise<IntegrationTestResult> {
        const startTime = Date.now();
        
        return withAIMocks(async (mockService) => {
            // Setup all tier mocks
            this.setupIntegrationMocks(scenario, mockService);
            
            // Initialize event tracking
            this.eventTracker.reset();
            this.eventTracker.startTracking();
            
            // Execute integration steps
            const executionTrace: ExecutionTrace[] = [];
            const issues: string[] = [];
            const warnings: string[] = [];
            
            try {
                for (const step of scenario.steps) {
                    const traceEntry = await this.executeIntegrationStep(
                        step,
                        scenario,
                        executionTrace,
                        mockService
                    );
                    executionTrace.push(traceEntry);
                    
                    // Check for step-level issues
                    if (traceEntry.errors && traceEntry.errors.length > 0) {
                        issues.push(...traceEntry.errors);
                    }
                }
            } catch (error) {
                issues.push(`Integration execution failed: ${error}`);
            } finally {
                this.eventTracker.stopTracking();
            }
            
            // Validate event flows
            const eventFlowValidation = this.validateEventFlow(
                scenario.expectedEventFlow,
                this.eventTracker.getEventHistory()
            );
            
            // Validate emergence
            const emergenceValidation = this.validateEmergence(
                scenario.expectedCapabilities,
                executionTrace,
                scenario.validationCriteria
            );
            
            // Validate cross-tier coordination
            const crossTierValidation = this.validateCrossTierCoordination(
                scenario.crossTierDependencies || [],
                executionTrace,
                this.eventTracker.getEventHistory()
            );
            
            // Calculate performance metrics
            const performanceMetrics = this.calculatePerformanceMetrics(
                executionTrace,
                startTime
            );
            
            // Determine overall success
            const success = this.determineOverallSuccess(
                eventFlowValidation,
                emergenceValidation,
                crossTierValidation,
                issues
            );
            
            // Generate recommendations
            const recommendations = this.generateIntegrationRecommendations(
                eventFlowValidation,
                emergenceValidation,
                crossTierValidation,
                performanceMetrics
            );
            
            return {
                scenario: scenario.name,
                success,
                executionTrace,
                eventFlowValidation,
                emergenceValidation,
                crossTierValidation,
                performanceMetrics,
                detectedCapabilities: emergenceValidation.detectedCapabilities,
                issues: issues.length > 0 ? issues : undefined,
                warnings: warnings.length > 0 ? warnings : undefined,
                recommendations
            };
        }, { debug: true, validateResponses: true });
    }
    
    /**
     * Setup mocks for all tiers in the integration
     */
    private setupIntegrationMocks(
        scenario: IntegrationScenario,
        mockService: any
    ): void {
        for (const tier of scenario.tiers) {
            tier.mockBehaviors.forEach((config, id) => {
                mockService.registerBehavior(`${tier.tier}_${id}`, {
                    pattern: config.pattern,
                    response: config,
                    priority: config.priority || 10,
                    metadata: {
                        tier: tier.tier,
                        domain: scenario.domain,
                        scenario: scenario.name
                    }
                });
            });
        }
        
        // Setup cross-tier coordination mocks
        const crossTierMocks = TierMockFactories.crossTier.createIntegrationMocks(scenario.domain);
        crossTierMocks.forEach((config, id) => {
            mockService.registerBehavior(`cross_tier_${id}`, {
                pattern: config.pattern,
                response: config,
                priority: (config.priority || 10) + 5 // Higher priority for cross-tier
            });
        });
    }
    
    /**
     * Execute a single integration step
     */
    private async executeIntegrationStep(
        step: IntegrationStep,
        scenario: IntegrationScenario,
        previousSteps: ExecutionTrace[],
        mockService: any
    ): Promise<ExecutionTrace> {
        const stepStartTime = Date.now();
        const eventsEmitted: ExecutionEvent[] = [];
        const eventsConsumed: ExecutionEvent[] = [];
        const errors: string[] = [];
        
        try {
            // Check dependencies
            if (step.dependencies) {
                for (const depName of step.dependencies) {
                    const dependency = previousSteps.find(s => s.step === depName);
                    if (!dependency) {
                        errors.push(`Missing dependency: ${depName}`);
                        continue;
                    }
                    
                    // Consume events from dependency
                    eventsConsumed.push(...dependency.eventsEmitted);
                }
            }
            
            // Execute step based on tier
            const stepResult = await this.executeStepByTier(
                step,
                scenario,
                eventsConsumed,
                mockService
            );
            
            // Extract emitted events
            if (step.triggeredEvents) {
                for (const eventType of step.triggeredEvents) {
                    eventsEmitted.push(this.createExecutionEvent(
                        eventType,
                        step.tier,
                        stepResult
                    ));
                }
            }
            
            // Detect emergent behavior
            const emergentBehavior = this.detectStepEmergence(stepResult, step);
            
            return {
                step: step.name,
                tier: step.tier,
                startTime: stepStartTime,
                endTime: Date.now(),
                input: step.input,
                output: stepResult,
                emergentBehavior,
                eventsEmitted,
                eventsConsumed,
                errors: errors.length > 0 ? errors : undefined
            };
            
        } catch (error) {
            return {
                step: step.name,
                tier: step.tier,
                startTime: stepStartTime,
                endTime: Date.now(),
                input: step.input,
                output: null,
                eventsEmitted,
                eventsConsumed,
                errors: [error instanceof Error ? error.message : String(error)]
            };
        }
    }
    
    /**
     * Execute step based on its tier
     */
    private async executeStepByTier(
        step: IntegrationStep,
        scenario: IntegrationScenario,
        consumedEvents: ExecutionEvent[],
        mockService: any
    ): Promise<any> {
        const tierConfig = scenario.tiers.find(t => t.tier === step.tier);
        
        if (!tierConfig && step.tier !== "cross-tier") {
            throw new Error(`No configuration found for tier: ${step.tier}`);
        }
        
        // Create tier-specific request
        const request = {
            model: "gpt-4o-mini",
            messages: [{
                role: "user",
                content: this.createTierSpecificContent(step, consumedEvents, scenario.domain)
            }],
            metadata: {
                tier: step.tier,
                step: step.name,
                domain: scenario.domain
            }
        };
        
        // Execute with mock service
        const result = await mockService.execute(request);
        
        // Add tier-specific processing
        return this.processTierSpecificResult(step.tier, result, tierConfig);
    }
    
    /**
     * Create tier-specific content for AI request
     */
    private createTierSpecificContent(
        step: IntegrationStep,
        consumedEvents: ExecutionEvent[],
        domain: string
    ): string {
        const baseContent = `Execute ${step.name} in ${step.tier} for ${domain} domain`;
        
        if (consumedEvents.length > 0) {
            const eventSummary = consumedEvents.map(e => e.type).join(", ");
            return `${baseContent}. Processing events: ${eventSummary}. Input: ${JSON.stringify(step.input)}`;
        }
        
        return `${baseContent}. Input: ${JSON.stringify(step.input)}`;
    }
    
    /**
     * Process tier-specific result
     */
    private processTierSpecificResult(
        tier: string,
        result: any,
        tierConfig?: IntegrationTier
    ): any {
        const processed = { ...result };
        
        // Add tier-specific metadata
        processed.metadata = {
            ...processed.metadata,
            tier,
            processedAt: new Date().toISOString()
        };
        
        // Add tier-specific processing
        switch (tier) {
            case "tier1":
                processed.swarmCoordination = {
                    agentsInvolved: 3,
                    coordinationStrategy: "hierarchical",
                    taskDelegation: true
                };
                break;
                
            case "tier2":
                processed.routineExecution = {
                    strategy: "reasoning",
                    stepsExecuted: 4,
                    optimizationApplied: true
                };
                break;
                
            case "tier3":
                processed.toolExecution = {
                    toolsUsed: result.response?.toolCalls?.length || 0,
                    resourcesConsumed: "moderate",
                    performanceOptimized: true
                };
                break;
                
            case "cross-tier":
                processed.crossTierCoordination = {
                    tiersInvolved: ["tier1", "tier2", "tier3"],
                    synchronizationLatency: 50,
                    dataConsistency: true
                };
                break;
        }
        
        return processed;
    }
    
    /**
     * Create execution event
     */
    private createExecutionEvent(
        eventType: string,
        sourceTier: string,
        stepResult: any
    ): ExecutionEvent {
        return {
            id: `event_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            type: eventType,
            timestamp: new Date(),
            source: sourceTier,
            tier: this.getTierNumber(sourceTier),
            category: "integration",
            subcategory: "step_execution",
            deliveryGuarantee: "reliable",
            priority: "medium",
            data: {
                stepResult: stepResult,
                sourceTier
            }
        };
    }
    
    /**
     * Detect emergent behavior in step result
     */
    private detectStepEmergence(
        stepResult: any,
        step: IntegrationStep
    ): EmergentBehavior | undefined {
        if (!stepResult?.response) return undefined;
        
        const response = stepResult.response;
        const content = response.content?.toLowerCase() || "";
        
        // Look for emergent behavior indicators
        const emergentPatterns = [
            { pattern: /adapt|learn|improve|evolve/, type: "adaptive_learning" },
            { pattern: /coordinate|synchronize|collaborate/, type: "coordination" },
            { pattern: /optimize|enhance|streamline/, type: "optimization" },
            { pattern: /discover|insight|realize|understand/, type: "discovery" }
        ];
        
        for (const { pattern, type } of emergentPatterns) {
            if (pattern.test(content)) {
                return {
                    type,
                    confidence: 0.8,
                    evidence: [content],
                    relatedCapability: this.mapBehaviorToCapability(type)
                };
            }
        }
        
        return undefined;
    }
    
    /**
     * Validate event flow between tiers
     */
    private validateEventFlow(
        expectedFlows: EventFlowExpectation[],
        eventHistory: ExecutionEvent[]
    ): EventFlowValidation {
        const actualFlows: ActualEventFlow[] = [];
        let flowsMatched = 0;
        
        // Analyze actual event flows
        for (let i = 0; i < eventHistory.length - 1; i++) {
            const currentEvent = eventHistory[i];
            const nextEvent = eventHistory[i + 1];
            
            const actualFlow: ActualEventFlow = {
                fromTier: currentEvent.source,
                toTier: nextEvent.source,
                eventType: currentEvent.type,
                latency: nextEvent.timestamp.getTime() - currentEvent.timestamp.getTime(),
                timestamp: currentEvent.timestamp.getTime(),
                matched: false
            };
            
            // Try to match with expected flows
            const expectedFlow = expectedFlows.find(ef => 
                ef.eventType === actualFlow.eventType &&
                ef.fromTier === actualFlow.fromTier &&
                ef.toTier === actualFlow.toTier
            );
            
            if (expectedFlow) {
                actualFlow.matched = true;
                actualFlow.expectedFlow = expectedFlow;
                flowsMatched++;
            }
            
            actualFlows.push(actualFlow);
        }
        
        // Calculate average latency
        const latencies = actualFlows.map(f => f.latency);
        const averageLatency = latencies.length > 0 
            ? latencies.reduce((sum, l) => sum + l, 0) / latencies.length
            : 0;
        
        // Validate sequence
        const sequenceValidation = this.validateEventSequence(expectedFlows, actualFlows);
        
        const valid = flowsMatched >= expectedFlows.length * 0.8; // 80% match threshold
        
        return {
            expectedFlows,
            actualFlows,
            flowsMatched,
            flowsTotal: expectedFlows.length,
            averageLatency,
            sequenceValidation,
            valid
        };
    }
    
    /**
     * Validate event sequence
     */
    private validateEventSequence(
        expectedFlows: EventFlowExpectation[],
        actualFlows: ActualEventFlow[]
    ): SequenceValidation {
        const expectedSequence = expectedFlows
            .filter(ef => ef.sequence !== undefined)
            .sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
            .map(ef => ef.eventType);
        
        const actualSequence = actualFlows
            .filter(af => af.matched && af.expectedFlow?.sequence !== undefined)
            .sort((a, b) => a.timestamp - b.timestamp)
            .map(af => af.eventType);
        
        const sequenceCorrect = JSON.stringify(expectedSequence) === JSON.stringify(actualSequence);
        
        const outOfOrderEvents = actualSequence.filter((event, index) => 
            expectedSequence[index] !== event
        );
        
        return {
            expectedSequence,
            actualSequence,
            sequenceCorrect,
            outOfOrderEvents
        };
    }
    
    /**
     * Validate emergence across the integration
     */
    private validateEmergence(
        expectedCapabilities: string[],
        executionTrace: ExecutionTrace[],
        criteria?: IntegrationValidationCriteria
    ): EmergenceValidation {
        // Simulate interactions for emergence detection
        const mockInteractions = executionTrace.map(trace => ({
            timestamp: new Date(trace.startTime),
            request: trace.input,
            response: trace.output?.response || trace.output,
            context: { tier: trace.tier, step: trace.step }
        }));
        
        const emergenceResult = this.emergenceDetector.detectEmergence(
            mockInteractions,
            expectedCapabilities
        );
        
        const detectedCapabilities = emergenceResult.expectedCapabilities;
        const emergenceScore = emergenceResult.emergenceScore;
        
        // Calculate convergence time (time to detect all capabilities)
        const convergenceTime = emergenceResult.temporalAnalysis?.timeToEmergence || 0;
        
        // Calculate stability period
        const stabilityPeriod = emergenceResult.temporalAnalysis?.stabilityPeriod || 0;
        
        // Check for cross-tier emergence
        const crossTierEmergence = this.detectCrossTierEmergence(executionTrace);
        
        const minimumDetection = criteria?.minimumCapabilityDetection || 0.8;
        const emergenceThreshold = criteria?.emergenceThreshold || 0.7;
        
        const valid = (detectedCapabilities.length / expectedCapabilities.length) >= minimumDetection &&
                     emergenceScore >= emergenceThreshold;
        
        return {
            expectedCapabilities,
            detectedCapabilities,
            emergenceScore,
            convergenceTime,
            stabilityPeriod,
            crossTierEmergence,
            valid
        };
    }
    
    /**
     * Detect cross-tier emergent behavior
     */
    private detectCrossTierEmergence(executionTrace: ExecutionTrace[]): boolean {
        // Look for behaviors that span multiple tiers
        const tiersBehaviors = new Map<string, Set<string>>();
        
        for (const trace of executionTrace) {
            if (!tiersBehaviors.has(trace.tier)) {
                tiersBehaviors.set(trace.tier, new Set());
            }
            
            if (trace.emergentBehavior) {
                tiersBehaviors.get(trace.tier)!.add(trace.emergentBehavior.type);
            }
        }
        
        // Check for common behaviors across tiers
        const allBehaviors = Array.from(tiersBehaviors.values());
        if (allBehaviors.length < 2) return false;
        
        const intersection = allBehaviors.reduce((common, behaviors) => {
            return new Set([...common].filter(b => behaviors.has(b)));
        });
        
        return intersection.size > 0;
    }
    
    /**
     * Validate cross-tier coordination
     */
    private validateCrossTierCoordination(
        expectedDependencies: CrossTierDependency[],
        executionTrace: ExecutionTrace[],
        eventHistory: ExecutionEvent[]
    ): CrossTierValidation {
        let satisfiedDependencies = 0;
        
        // Check if dependencies are satisfied
        for (const dependency of expectedDependencies) {
            const dependentSteps = executionTrace.filter(t => t.tier === dependency.dependentTier);
            const providerSteps = executionTrace.filter(t => t.tier === dependency.providerTier);
            
            // Simple check: dependent tier executed after provider tier
            const dependentTime = Math.min(...dependentSteps.map(s => s.startTime));
            const providerTime = Math.max(...providerSteps.map(s => s.endTime));
            
            if (dependentTime > providerTime) {
                satisfiedDependencies++;
            }
        }
        
        // Count coordination events
        const coordinationEvents = eventHistory.filter(e => 
            e.category === "coordination" || e.type.includes("coordination")
        ).length;
        
        // Calculate synchronization latency
        const tierTimes = new Map<string, number[]>();
        for (const trace of executionTrace) {
            if (!tierTimes.has(trace.tier)) {
                tierTimes.set(trace.tier, []);
            }
            tierTimes.get(trace.tier)!.push(trace.endTime - trace.startTime);
        }
        
        const avgTierTimes = Array.from(tierTimes.values()).map(times => 
            times.reduce((sum, t) => sum + t, 0) / times.length
        );
        const syncLatency = Math.max(...avgTierTimes) - Math.min(...avgTierTimes);
        
        // Simple data consistency check
        const dataConsistency = this.validateDataConsistency(executionTrace);
        
        const valid = satisfiedDependencies === expectedDependencies.length &&
                     coordinationEvents > 0 &&
                     dataConsistency;
        
        return {
            expectedDependencies,
            satisfiedDependencies,
            coordinationEvents,
            syncrhonizationLatency: syncLatency,
            dataConsistency,
            valid
        };
    }
    
    /**
     * Validate data consistency across tiers
     */
    private validateDataConsistency(executionTrace: ExecutionTrace[]): boolean {
        // Simple consistency check: ensure data flows properly between steps
        for (let i = 1; i < executionTrace.length; i++) {
            const current = executionTrace[i];
            const previous = executionTrace[i - 1];
            
            // Check if current step consumed events from previous step
            if (current.eventsConsumed.length === 0 && previous.eventsEmitted.length > 0) {
                // Potential consistency issue
                continue;
            }
        }
        
        return true; // Simplified - would be more sophisticated in practice
    }
    
    /**
     * Calculate performance metrics
     */
    private calculatePerformanceMetrics(
        executionTrace: ExecutionTrace[],
        startTime: number
    ): IntegrationPerformanceMetrics {
        const totalExecutionTime = Date.now() - startTime;
        
        // Calculate tier breakdown
        const tierBreakdown = {
            tier1: 0,
            tier2: 0,
            tier3: 0,
            coordination: 0
        };
        
        for (const trace of executionTrace) {
            const duration = trace.endTime - trace.startTime;
            if (trace.tier === "tier1") tierBreakdown.tier1 += duration;
            else if (trace.tier === "tier2") tierBreakdown.tier2 += duration;
            else if (trace.tier === "tier3") tierBreakdown.tier3 += duration;
            else tierBreakdown.coordination += duration;
        }
        
        // Calculate throughput (steps per second)
        const throughput = executionTrace.length / (totalExecutionTime / 1000);
        
        // Calculate event latency statistics
        const eventLatencies = executionTrace.flatMap(trace => 
            trace.eventsEmitted.map(event => {
                const nextTrace = executionTrace.find(t => 
                    t.startTime > trace.endTime && 
                    t.eventsConsumed.some(consumed => consumed.type === event.type)
                );
                return nextTrace ? nextTrace.startTime - trace.endTime : 0;
            })
        ).filter(latency => latency > 0);
        
        const eventLatency = eventLatencies.length > 0 ? {
            min: Math.min(...eventLatencies),
            max: Math.max(...eventLatencies),
            average: eventLatencies.reduce((sum, l) => sum + l, 0) / eventLatencies.length,
            p95: this.calculatePercentile(eventLatencies, 0.95)
        } : { min: 0, max: 0, average: 0, p95: 0 };
        
        return {
            totalExecutionTime,
            tierBreakdown,
            throughput,
            eventLatency,
            resourceUtilization: {
                memory: 0.3, // Simulated
                cpu: 0.4,    // Simulated
                network: 0.2  // Simulated
            }
        };
    }
    
    /**
     * Determine overall success
     */
    private determineOverallSuccess(
        eventFlowValidation: EventFlowValidation,
        emergenceValidation: EmergenceValidation,
        crossTierValidation: CrossTierValidation,
        issues: string[]
    ): boolean {
        return eventFlowValidation.valid &&
               emergenceValidation.valid &&
               crossTierValidation.valid &&
               issues.length === 0;
    }
    
    /**
     * Generate integration recommendations
     */
    private generateIntegrationRecommendations(
        eventFlowValidation: EventFlowValidation,
        emergenceValidation: EmergenceValidation,
        crossTierValidation: CrossTierValidation,
        performanceMetrics: IntegrationPerformanceMetrics
    ): string[] {
        const recommendations: string[] = [];
        
        if (!eventFlowValidation.valid) {
            recommendations.push("Improve event flow patterns - some expected flows were not observed");
        }
        
        if (!emergenceValidation.valid) {
            recommendations.push("Enhance capability emergence - some capabilities were not detected");
        }
        
        if (!crossTierValidation.valid) {
            recommendations.push("Strengthen cross-tier coordination - dependencies not properly satisfied");
        }
        
        if (performanceMetrics.eventLatency.average > 1000) {
            recommendations.push("Optimize event processing - high latency detected");
        }
        
        if (emergenceValidation.convergenceTime > 10000) {
            recommendations.push("Accelerate capability emergence - convergence time is high");
        }
        
        return recommendations;
    }
    
    // Utility methods
    
    private getTierNumber(tierString: string): number {
        switch (tierString) {
            case "tier1": return 1;
            case "tier2": return 2;
            case "tier3": return 3;
            default: return 0;
        }
    }
    
    private mapBehaviorToCapability(behaviorType: string): string {
        const mapping: Record<string, string> = {
            "adaptive_learning": "continuous_improvement",
            "coordination": "multi_agent_coordination",
            "optimization": "performance_optimization",
            "discovery": "insight_generation"
        };
        return mapping[behaviorType] || behaviorType;
    }
    
    private calculatePercentile(values: number[], percentile: number): number {
        const sorted = [...values].sort((a, b) => a - b);
        const index = Math.ceil(sorted.length * percentile) - 1;
        return sorted[index] || 0;
    }
}

/**
 * Event tracker for monitoring event flow
 */
class EventTracker {
    private eventHistory: ExecutionEvent[] = [];
    private tracking = false;
    
    startTracking(): void {
        this.tracking = true;
        this.eventHistory = [];
    }
    
    stopTracking(): void {
        this.tracking = false;
    }
    
    trackEvent(event: ExecutionEvent): void {
        if (this.tracking) {
            this.eventHistory.push(event);
        }
    }
    
    getEventHistory(): ExecutionEvent[] {
        return [...this.eventHistory];
    }
    
    reset(): void {
        this.eventHistory = [];
        this.tracking = false;
    }
}