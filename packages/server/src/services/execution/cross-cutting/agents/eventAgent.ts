import { type Logger } from "winston";
import { type AgentResponse, type IntelligentEvent } from "../events/eventBus.js";

/**
 * Event Agent - Intelligent agent for processing events
 * 
 * Event agents are specialized AI entities that subscribe to specific event patterns
 * and provide domain-specific intelligence, learning, and adaptation capabilities.
 */
export class EventAgent {
    private readonly agentId: string;
    private readonly capabilities: string[];
    private readonly logger: Logger;
    private learningData: Map<string, EventLearningData> = new Map();
    private performanceMetrics: AgentPerformanceMetrics;

    constructor(agentId: string, capabilities: string[], logger?: Logger) {
        this.agentId = agentId;
        this.capabilities = capabilities;
        this.logger = logger || console as any;
        this.performanceMetrics = {
            eventsProcessed: 0,
            averageResponseTime: 0,
            accuracyScore: 0,
            lastUpdated: new Date(),
        };
    }

    /**
     * Process an event and generate an intelligent response
     */
    async processEvent(event: IntelligentEvent): Promise<AgentResponse> {
        const startTime = Date.now();

        try {
            // Analyze event context and patterns
            const analysis = await this.analyzeEvent(event);

            // Generate response based on agent's capabilities and learning
            const response = await this.generateResponse(event, analysis);

            // Update performance metrics
            const processingTime = Date.now() - startTime;
            this.updatePerformanceMetrics(processingTime);

            this.logger.debug(`[EventAgent:${this.agentId}] Processed event`, {
                eventId: event.id,
                eventType: event.type,
                responseStatus: response.status,
                confidence: response.confidence,
                processingTime,
            });

            return response;

        } catch (error) {
            this.logger.error(`[EventAgent:${this.agentId}] Error processing event`, {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                status: "ESCALATE",
                confidence: 0,
                reasoning: `Agent processing failed: ${error instanceof Error ? error.message : String(error)}`,
                suggestedActions: ["manual_review", "error_investigation"],
            };
        }
    }

    /**
     * Get agent insights and recommendations
     */
    getInsights(): AgentInsights {
        const patterns = Array.from(this.learningData.values())
            .sort((a, b) => b.occurrences - a.occurrences)
            .slice(0, 10); // Top 10 patterns

        const recommendations: string[] = [];

        // Analyze patterns for recommendations
        for (const pattern of patterns) {
            if (pattern.successRate < 0.7 && pattern.occurrences > 5) {
                recommendations.push(
                    `Improve handling for pattern '${pattern.pattern}' (success rate: ${Math.round(pattern.successRate * 100)}%)`,
                );
            }

            if (pattern.averageConfidence < 0.6 && pattern.occurrences > 10) {
                recommendations.push(
                    `Low confidence for pattern '${pattern.pattern}' (avg: ${Math.round(pattern.averageConfidence * 100)}%) - consider additional training`,
                );
            }
        }

        return {
            agentId: this.agentId,
            capabilities: this.capabilities,
            performance: this.performanceMetrics,
            topPatterns: patterns,
            recommendations,
            totalPatternsLearned: this.learningData.size,
            generatedAt: new Date(),
        };
    }

    /**
     * Check if agent can handle specific event type
     */
    canHandle(event: IntelligentEvent): boolean {
        // Check if event requires capabilities this agent has
        if (event.agentContext?.requiredCapabilities) {
            const hasRequiredCapabilities = event.agentContext.requiredCapabilities.every(
                capability => this.capabilities.includes(capability),
            );
            if (!hasRequiredCapabilities) return false;
        }

        // Check if agent is explicitly excluded
        if (event.agentContext?.excludedAgents?.includes(this.agentId)) {
            return false;
        }

        // Check if agent is specifically targeted
        if (event.agentContext?.targetAgents) {
            return event.agentContext.targetAgents.includes(this.agentId);
        }

        return true;
    }

    /**
     * Private helper methods
     */
    private async analyzeEvent(event: IntelligentEvent): Promise<EventAnalysis> {
        const pattern = this.extractEventPattern(event);
        const historicalData = this.learningData.get(pattern);

        return {
            pattern,
            confidence: this.calculateConfidence(event, historicalData),
            riskLevel: this.assessRiskLevel(event),
            similarity: this.findSimilarEvents(event),
            contextFactors: this.extractContextFactors(event),
        };
    }

    private async generateResponse(
        event: IntelligentEvent,
        analysis: EventAnalysis,
    ): Promise<AgentResponse> {
        // Default response based on agent capabilities and analysis
        let status: AgentResponse["status"] = "OK";
        let confidence = analysis.confidence;
        let reasoning = "Event processed successfully";
        const suggestedActions: string[] = [];

        // Risk-based decision making
        if (analysis.riskLevel === "high") {
            if (this.capabilities.includes("security_analysis")) {
                status = "ALARM";
                reasoning = "High risk event detected requiring security review";
                suggestedActions.push("security_review", "immediate_attention");
            } else {
                status = "ESCALATE";
                reasoning = "High risk event requires specialized security agent";
                suggestedActions.push("route_to_security_agent");
            }
        } else if (analysis.riskLevel === "medium") {
            if (confidence < 0.7) {
                status = "DEFER";
                reasoning = "Medium risk event with low confidence requires additional analysis";
                suggestedActions.push("gather_additional_context", "consult_other_agents");
            }
        }

        // Capability-specific logic
        if (this.capabilities.includes("performance_optimization")) {
            this.addPerformanceInsights(event, suggestedActions);
        }

        if (this.capabilities.includes("quality_assurance")) {
            this.addQualityInsights(event, suggestedActions);
        }

        if (this.capabilities.includes("compliance_monitoring")) {
            this.addComplianceInsights(event, suggestedActions);
        }

        return {
            status,
            confidence,
            reasoning,
            suggestedActions: suggestedActions.length > 0 ? suggestedActions : undefined,
            metadata: {
                agentId: this.agentId,
                capabilities: this.capabilities,
                analysisPattern: analysis.pattern,
                riskLevel: analysis.riskLevel,
            },
        };
    }

    private extractEventPattern(event: IntelligentEvent): string {
        return `${event.category}/${event.subcategory || "*"}/${event.tier}`;
    }

    private calculateConfidence(
        event: IntelligentEvent,
        historicalData?: EventLearningData,
    ): number {
        let baseConfidence = 0.5;

        // Increase confidence based on historical success
        if (historicalData) {
            baseConfidence = Math.min(0.9, 0.3 + (historicalData.successRate * 0.6));
        }

        // Adjust based on event complexity
        const complexity = this.assessEventComplexity(event);
        baseConfidence *= (1 - complexity * 0.3);

        // Ensure confidence is within reasonable bounds
        return Math.max(0.1, Math.min(0.95, baseConfidence));
    }

    private assessRiskLevel(event: IntelligentEvent): "low" | "medium" | "high" {
        if (event.priority === "EMERGENCY" || event.priority === "CRITICAL") {
            return "high";
        }

        if (event.securityLevel === "confidential" || event.securityLevel === "secret") {
            return "high";
        }

        if (event.humanApprovalRequired || event.complianceRequired) {
            return "medium";
        }

        if (event.category === "safety" || event.category === "security") {
            return "medium";
        }

        return "low";
    }

    private assessEventComplexity(event: IntelligentEvent): number {
        let complexity = 0;

        // More complex if it has many related events
        if (event.relatedEvents && event.relatedEvents.length > 3) {
            complexity += 0.2;
        }

        // More complex if it requires specific capabilities
        if (event.agentContext?.requiredCapabilities && event.agentContext.requiredCapabilities.length > 2) {
            complexity += 0.3;
        }

        // More complex if it's cross-tier
        if (event.correlationChain && event.correlationChain.length > 1) {
            complexity += 0.2;
        }

        return Math.min(1, complexity);
    }

    private findSimilarEvents(event: IntelligentEvent): string[] {
        const similar: string[] = [];
        const currentPattern = this.extractEventPattern(event);

        for (const [pattern, data] of this.learningData) {
            if (pattern !== currentPattern && this.patternsAreSimilar(pattern, currentPattern)) {
                similar.push(pattern);
            }
        }

        return similar.slice(0, 5); // Top 5 similar patterns
    }

    private patternsAreSimilar(pattern1: string, pattern2: string): boolean {
        const parts1 = pattern1.split("/");
        const parts2 = pattern2.split("/");

        // Similar if they share category or tier
        return parts1[0] === parts2[0] || parts1[2] === parts2[2];
    }

    private extractContextFactors(event: IntelligentEvent): Record<string, unknown> {
        return {
            tier: event.tier,
            category: event.category,
            subcategory: event.subcategory,
            priority: event.priority,
            deliveryGuarantee: event.deliveryGuarantee,
            hasCorrelationChain: !!event.correlationChain?.length,
            hasRelatedEvents: !!event.relatedEvents?.length,
            requiresApproval: event.humanApprovalRequired,
            securityLevel: event.securityLevel,
        };
    }

    private addPerformanceInsights(event: IntelligentEvent, actions: string[]): void {
        if (event.category === "routine" && event.type.includes("completed")) {
            actions.push("analyze_performance_metrics", "identify_optimization_opportunities");
        }
    }

    private addQualityInsights(event: IntelligentEvent, actions: string[]): void {
        if (event.category === "step" || event.category === "routine") {
            actions.push("validate_output_quality", "check_compliance_standards");
        }
    }

    private addComplianceInsights(event: IntelligentEvent, actions: string[]): void {
        if (event.complianceRequired || event.securityLevel === "confidential") {
            actions.push("audit_trail_update", "compliance_verification");
        }
    }

    private updatePerformanceMetrics(processingTime: number): void {
        this.performanceMetrics.eventsProcessed++;

        // Update rolling average response time
        const alpha = 0.1; // Exponential moving average factor
        this.performanceMetrics.averageResponseTime =
            (1 - alpha) * this.performanceMetrics.averageResponseTime + alpha * processingTime;

        this.performanceMetrics.lastUpdated = new Date();
    }
}

/**
 * Supporting interfaces
 */
interface EventLearningData {
    pattern: string;
    occurrences: number;
    responses: Array<{
        status: string;
        confidence: number;
        timestamp: Date;
    }>;
    averageConfidence: number;
    successRate: number;
    lastSeen: Date;
}

interface AgentPerformanceMetrics {
    eventsProcessed: number;
    averageResponseTime: number;
    accuracyScore: number;
    lastUpdated: Date;
}

interface EventAnalysis {
    pattern: string;
    confidence: number;
    riskLevel: "low" | "medium" | "high";
    similarity: string[];
    contextFactors: Record<string, unknown>;
}

interface AgentInsights {
    agentId: string;
    capabilities: string[];
    performance: AgentPerformanceMetrics;
    topPatterns: EventLearningData[];
    recommendations: string[];
    totalPatternsLearned: number;
    generatedAt: Date;
}