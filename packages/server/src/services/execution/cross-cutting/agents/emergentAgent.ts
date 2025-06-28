import { type Logger } from "winston";
import { type AgentResponse, type IntelligentEvent } from "../../../events/types.js";

/**
 * Emergent Agent - Base infrastructure for goal-driven intelligent agents
 * 
 * This is NOT a hard-coded agent type. Instead, it provides the infrastructure
 * for agents that emerge based on specific goals and routines deployed by teams.
 * Agents learn from event patterns and propose routine improvements.
 */
export class EmergentAgent {
    protected readonly agentId: string;
    protected readonly goal: string;
    protected readonly initialRoutine: string;
    protected readonly logger: Logger;

    // Learning and pattern recognition
    private eventPatterns: Map<string, EventPattern> = new Map();
    private routineProposals: Map<string, RoutineProposal> = new Map();
    private learningData: Map<string, AgentLearningData> = new Map();

    // Agent state
    private performanceMetrics: AgentPerformanceMetrics;
    private lastActivity: Date;
    private proposalCounter = 0;

    constructor(
        agentId: string,
        goal: string,
        initialRoutine: string,
        logger?: Logger,
    ) {
        this.agentId = agentId;
        this.goal = goal;
        this.initialRoutine = initialRoutine;
        this.logger = logger || console as any;
        this.lastActivity = new Date();

        this.performanceMetrics = {
            eventsProcessed: 0,
            patternsLearned: 0,
            routinesProposed: 0,
            routinesAccepted: 0,
            averageConfidence: 0,
            lastUpdated: new Date(),
        };

        this.logger.info(`[EmergentAgent:${this.agentId}] Deployed with goal: ${this.goal}`);
    }

    /**
     * Process events and learn patterns to achieve the agent's goal
     */
    async processEvent(event: IntelligentEvent): Promise<AgentResponse> {
        const startTime = Date.now();
        this.lastActivity = new Date();

        try {
            // 1. Analyze event in context of agent's goal
            const goalRelevance = await this.analyzeGoalRelevance(event);

            if (goalRelevance.score < 0.3) {
                // Event not relevant to this agent's goal
                return {
                    status: "OK",
                    confidence: 0.9,
                    reasoning: "Event not relevant to agent goal",
                };
            }

            // 2. Learn from this event pattern
            await this.learnFromEvent(event, goalRelevance);

            // 3. Check if this event suggests a routine improvement opportunity
            const improvementOpportunity = await this.identifyImprovementOpportunity(event);

            // 4. Generate response based on goal-driven analysis
            const response = await this.generateGoalDrivenResponse(event, goalRelevance, improvementOpportunity);

            // 5. Update performance metrics
            this.updatePerformanceMetrics(response, Date.now() - startTime);

            return response;

        } catch (error) {
            this.logger.error(`[EmergentAgent:${this.agentId}] Error processing event`, {
                eventId: event.id,
                goal: this.goal,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                status: "ESCALATE",
                confidence: 0,
                reasoning: `Agent processing failed: ${error instanceof Error ? error.message : String(error)}`,
                suggestedActions: ["agent_debug", "manual_review"],
            };
        }
    }

    /**
     * Propose a routine improvement based on learned patterns
     */
    async proposeRoutineImprovement(
        targetRoutine: string,
        improvementType: "optimization" | "quality" | "security" | "feature",
        analysis: ImprovementAnalysis,
    ): Promise<string> {
        const proposalId = `${this.agentId}_proposal_${++this.proposalCounter}_${Date.now()}`;

        const proposal: RoutineProposal = {
            id: proposalId,
            agentId: this.agentId,
            targetRoutine,
            improvementType,
            goal: this.goal,
            analysis,
            confidence: analysis.confidence,
            status: "proposed",
            createdAt: new Date(),
            estimatedImpact: this.estimateImpact(analysis),
        };

        this.routineProposals.set(proposalId, proposal);
        this.performanceMetrics.routinesProposed++;

        this.logger.info(`[EmergentAgent:${this.agentId}] Proposed routine improvement`, {
            proposalId,
            targetRoutine,
            improvementType,
            confidence: analysis.confidence,
            estimatedImpact: proposal.estimatedImpact,
        });

        // In a real implementation, this would:
        // 1. Create a new routine version with improvements
        // 2. Generate a pull request
        // 3. Notify the team for review

        return proposalId;
    }

    /**
     * Get the latest routine improvement proposal
     */
    getLatestProposal(): RoutineProposal | null {
        const proposals = Array.from(this.routineProposals.values())
            .filter(p => p.status === "proposed")
            .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());

        return proposals[0] || null;
    }

    /**
     * Accept a routine improvement proposal
     */
    async acceptProposal(proposalId: string): Promise<boolean> {
        const proposal = this.routineProposals.get(proposalId);
        if (!proposal || proposal.status !== "proposed") {
            return false;
        }

        proposal.status = "accepted";
        proposal.acceptedAt = new Date();
        this.performanceMetrics.routinesAccepted++;

        this.logger.info(`[EmergentAgent:${this.agentId}] Proposal accepted`, {
            proposalId,
            targetRoutine: proposal.targetRoutine,
            improvementType: proposal.improvementType,
        });

        // In a real implementation, this would:
        // 1. Deploy the improved routine
        // 2. Update the agent's learning based on success
        // 3. Monitor the improvement's performance

        return true;
    }

    /**
     * Reject a routine improvement proposal
     */
    async rejectProposal(proposalId: string, reason: string): Promise<boolean> {
        const proposal = this.routineProposals.get(proposalId);
        if (!proposal || proposal.status !== "proposed") {
            return false;
        }

        proposal.status = "rejected";
        proposal.rejectedAt = new Date();
        proposal.rejectionReason = reason;

        // Learn from rejection to improve future proposals
        await this.learnFromRejection(proposal, reason);

        this.logger.info(`[EmergentAgent:${this.agentId}] Proposal rejected`, {
            proposalId,
            reason,
        });

        return true;
    }

    /**
     * Private helper methods for goal-driven intelligence
     */
    private async analyzeGoalRelevance(event: IntelligentEvent): Promise<GoalRelevance> {
        // Analyze how relevant this event is to the agent's goal
        const goalKeywords = this.extractGoalKeywords();
        const eventContext = this.extractEventContext(event);

        // Simple keyword-based relevance (could be enhanced with ML)
        const overlap = goalKeywords.filter(keyword =>
            eventContext.some(context => context.toLowerCase().includes(keyword.toLowerCase())),
        );

        const score = overlap.length / Math.max(goalKeywords.length, 1);

        return {
            score,
            relevantAspects: overlap,
            eventContext,
            reasoning: `Found ${overlap.length}/${goalKeywords.length} goal-relevant aspects`,
        };
    }

    /**
     * Extracts meaningful keywords from the agent's goal string
     * Used for simple relevance analysis
     */
    private extractGoalKeywords(): string[] {
        // Split goal into words and filter out common words
        const commonWords = new Set([
            "the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
            "of", "with", "by", "from", "up", "about", "into", "through", "during",
            "before", "after", "above", "below", "between", "same", "as", "is", "are",
            "was", "were", "been", "be", "have", "has", "had", "do", "does", "did",
            "will", "would", "should", "could", "may", "might", "must", "can", "shall",
        ]);

        // Extract keywords from goal
        const words = this.goal
            .toLowerCase()
            .replace(/[^\w\s]/g, " ") // Remove punctuation
            .split(/\s+/)
            .filter(word => word.length > 2 && !commonWords.has(word));

        // Add domain-specific keywords based on goal type
        const domainKeywords: string[] = [];

        if (this.goal.toLowerCase().includes("security")) {
            domainKeywords.push("threat", "vulnerability", "attack", "risk", "protection");
        }
        if (this.goal.toLowerCase().includes("performance")) {
            domainKeywords.push("speed", "latency", "throughput", "optimization", "efficiency");
        }
        if (this.goal.toLowerCase().includes("quality")) {
            domainKeywords.push("accuracy", "reliability", "consistency", "validation", "correctness");
        }
        if (this.goal.toLowerCase().includes("cost")) {
            domainKeywords.push("budget", "expense", "savings", "reduction", "efficiency");
        }

        return [...new Set([...words, ...domainKeywords])];
    }

    /**
     * Extracts contextual information from an event for relevance analysis
     */
    private extractEventContext(event: IntelligentEvent): string[] {
        const context: string[] = [];

        // Extract from event type
        context.push(event.type.toLowerCase());

        // Extract from event tier and metadata
        if (event.tier) {
            context.push(`tier${event.tier}`);
        }

        // Extract from event data
        if (event.data && typeof event.data === "object") {
            // Extract keys and string values
            for (const [key, value] of Object.entries(event.data)) {
                context.push(key.toLowerCase());
                if (typeof value === "string") {
                    context.push(value.toLowerCase());
                }
            }

            // Extract specific fields that are commonly relevant
            const relevantFields = ["operation", "action", "status", "result", "error", "type", "category"];
            for (const field of relevantFields) {
                const value = (event.data as any)[field];
                if (value && typeof value === "string") {
                    context.push(value.toLowerCase());
                }
            }
        }

        // Extract from any text content in the event
        const extractTextFromObject = (obj: unknown): string[] => {
            const texts: string[] = [];
            if (typeof obj === "string") {
                texts.push(obj);
            } else if (Array.isArray(obj)) {
                obj.forEach(item => texts.push(...extractTextFromObject(item)));
            } else if (obj && typeof obj === "object") {
                Object.values(obj).forEach(value => texts.push(...extractTextFromObject(value)));
            }
            return texts;
        };

        const allTexts = extractTextFromObject(event);
        context.push(...allTexts.map(t => t.toLowerCase()).filter(t => t.length > 2));

        // Remove duplicates and return
        return [...new Set(context)];
    }

    private async learnFromEvent(event: IntelligentEvent, relevance: GoalRelevance): Promise<void> {
        const pattern = this.extractEventPattern(event);

        let learningData = this.learningData.get(pattern);
        if (!learningData) {
            learningData = {
                pattern,
                occurrences: 0,
                averageRelevance: 0,
                goalAlignment: [],
                lastSeen: new Date(),
            };
            this.learningData.set(pattern, learningData);
        }

        // Update learning data
        learningData.occurrences++;
        learningData.averageRelevance =
            (learningData.averageRelevance * (learningData.occurrences - 1) + relevance.score) / learningData.occurrences;
        learningData.goalAlignment.push({
            score: relevance.score,
            aspects: relevance.relevantAspects,
            timestamp: new Date(),
        });
        learningData.lastSeen = new Date();

        // Keep only recent alignment data
        if (learningData.goalAlignment.length > 50) {
            learningData.goalAlignment = learningData.goalAlignment.slice(-50);
        }

        // Update event patterns for improvement detection
        await this.updateEventPattern(event, relevance);

        this.performanceMetrics.patternsLearned = this.learningData.size;
    }

    private async identifyImprovementOpportunity(event: IntelligentEvent): Promise<ImprovementOpportunity | null> {
        // Look for patterns that suggest routine improvements aligned with the agent's goal

        // Check for performance issues if goal includes optimization
        if (this.goal.toLowerCase().includes("performance") || this.goal.toLowerCase().includes("optimize")) {
            const performanceData = this.extractPerformanceData(event);
            if (performanceData && performanceData.executionTime > 1000) { // Slow execution
                return {
                    type: "optimization",
                    confidence: 0.8,
                    description: "Slow execution detected - optimization opportunity",
                    data: performanceData,
                };
            }
        }

        // Check for quality issues if goal includes quality
        if (this.goal.toLowerCase().includes("quality") || this.goal.toLowerCase().includes("accuracy")) {
            const qualityData = this.extractQualityData(event);
            if (qualityData && qualityData.qualityScore < 0.8) {
                return {
                    type: "quality",
                    confidence: 0.7,
                    description: "Quality degradation detected - improvement opportunity",
                    data: qualityData,
                };
            }
        }

        // Check for security issues if goal includes security
        if (this.goal.toLowerCase().includes("security") || this.goal.toLowerCase().includes("threat")) {
            const securityData = this.extractSecurityData(event);
            if (securityData && securityData.riskLevel === "high") {
                return {
                    type: "security",
                    confidence: 0.9,
                    description: "Security risk detected - mitigation opportunity",
                    data: securityData,
                };
            }
        }

        return null;
    }

    private async generateGoalDrivenResponse(
        event: IntelligentEvent,
        relevance: GoalRelevance,
        opportunity: ImprovementOpportunity | null,
    ): Promise<AgentResponse> {
        let status: AgentResponse["status"] = "OK";
        let confidence = relevance.score;
        let reasoning = `Event processed with ${Math.round(relevance.score * 100)}% goal relevance`;
        const suggestedActions: string[] = [];

        // Handle improvement opportunities
        if (opportunity) {
            if (opportunity.confidence > 0.8) {
                status = "DEFER";
                confidence = opportunity.confidence;
                reasoning = `${opportunity.description} - considering routine improvement`;
                suggestedActions.push("propose_routine_improvement");

                // Actually propose the improvement
                await this.proposeRoutineImprovement(
                    event.source || "unknown_routine",
                    opportunity.type,
                    {
                        opportunity,
                        eventAnalysis: {
                            relevance,
                            patterns: this.getRelevantPatterns(event),
                        },
                        confidence: opportunity.confidence,
                        reasoning: opportunity.description,
                    },
                );
            } else {
                suggestedActions.push("monitor_for_improvement_opportunity");
            }
        }

        // Goal-specific actions
        if (this.goal.toLowerCase().includes("monitor")) {
            suggestedActions.push("continue_monitoring");
        }

        if (relevance.score > 0.8) {
            suggestedActions.push("deep_analysis", "pattern_learning");
        }

        return {
            status,
            confidence,
            reasoning,
            suggestedActions: suggestedActions.length > 0 ? suggestedActions : undefined,
            metadata: {
                agentId: this.agentId,
                goal: this.goal,
                goalRelevance: relevance.score,
                improvementOpportunity: !!opportunity,
                patternsLearned: this.performanceMetrics.patternsLearned,
            },
        };
    }

    // Helper methods for data extraction and pattern analysis are defined above

    private extractEventPattern(event: IntelligentEvent): string {
        return `${event.category || "unknown"}/${event.subcategory || "*"}/${event.tier}`;
    }

    private extractKeywords(text: string): string[] {
        return text.toLowerCase()
            .replace(/[^\w\s]/g, "")
            .split(/\s+/)
            .filter(word => word.length > 3)
            .slice(0, 5); // Top 5 keywords
    }

    private extractPerformanceData(event: IntelligentEvent): any {
        if (!event.data) return null;

        const data = event.data as any;
        return {
            executionTime: data.executionTime || data.duration || data.responseTime,
            memoryUsage: data.memoryUsage,
            cpuUsage: data.cpuUsage,
            cost: data.cost,
        };
    }

    private extractQualityData(event: IntelligentEvent): any {
        if (!event.data) return null;

        const data = event.data as any;
        return {
            qualityScore: data.qualityScore || data.accuracy,
            errorCount: data.errorCount || data.errors?.length,
            validationStatus: data.validationStatus,
        };
    }

    private extractSecurityData(event: IntelligentEvent): any {
        if (!event.data) return null;

        const data = event.data as any;
        return {
            riskLevel: data.riskLevel || (event.securityLevel === "secret" ? "high" : "low"),
            threats: data.threats || [],
            violations: data.violations || [],
        };
    }

    private async updateEventPattern(event: IntelligentEvent, relevance: GoalRelevance): Promise<void> {
        const patternKey = this.extractEventPattern(event);

        let pattern = this.eventPatterns.get(patternKey);
        if (!pattern) {
            pattern = {
                pattern: patternKey,
                occurrences: 0,
                averageRelevance: 0,
                improvementOpportunities: [],
                lastSeen: new Date(),
            };
            this.eventPatterns.set(patternKey, pattern);
        }

        pattern.occurrences++;
        pattern.averageRelevance =
            (pattern.averageRelevance * (pattern.occurrences - 1) + relevance.score) / pattern.occurrences;
        pattern.lastSeen = new Date();
    }

    private getRelevantPatterns(event: IntelligentEvent): EventPattern[] {
        const currentPattern = this.extractEventPattern(event);

        return Array.from(this.eventPatterns.values())
            .filter(pattern =>
                pattern.pattern === currentPattern ||
                pattern.averageRelevance > 0.7,
            )
            .slice(0, 5);
    }

    private estimateImpact(analysis: ImprovementAnalysis): ImpactEstimate {
        return {
            performance: analysis.opportunity?.type === "optimization" ? 0.3 : 0,
            quality: analysis.opportunity?.type === "quality" ? 0.2 : 0,
            security: analysis.opportunity?.type === "security" ? 0.4 : 0,
            cost: analysis.opportunity?.type === "optimization" ? 0.15 : 0,
        };
    }

    private async learnFromRejection(proposal: RoutineProposal, reason: string): Promise<void> {
        // Learn from rejection to improve future proposals
        // This could update pattern weights, confidence thresholds, etc.
        this.logger.debug(`[EmergentAgent:${this.agentId}] Learning from rejection`, {
            proposalType: proposal.improvementType,
            reason,
        });
    }

    private identifyEmergentCapabilities(): string[] {
        const capabilities: string[] = [];

        // Identify capabilities that have emerged from learning
        if (this.performanceMetrics.routinesAccepted > 3) {
            capabilities.push(`${this.goal} expertise`);
        }

        if (this.performanceMetrics.patternsLearned > 10) {
            capabilities.push("pattern recognition");
        }

        if (this.routineProposals.size > 5) {
            capabilities.push("routine improvement");
        }

        // Goal-specific capabilities
        const goalLower = this.goal.toLowerCase();
        if (goalLower.includes("security") && this.performanceMetrics.routinesAccepted > 0) {
            capabilities.push("adaptive security");
        }

        if (goalLower.includes("performance") && this.performanceMetrics.routinesAccepted > 0) {
            capabilities.push("performance optimization");
        }

        if (goalLower.includes("quality") && this.performanceMetrics.routinesAccepted > 0) {
            capabilities.push("quality assurance");
        }

        return capabilities;
    }

    private updatePerformanceMetrics(response: AgentResponse, processingTime: number): void {
        this.performanceMetrics.eventsProcessed++;
        this.performanceMetrics.lastUpdated = new Date();

        // Update rolling average confidence
        const alpha = 0.1;
        this.performanceMetrics.averageConfidence =
            (1 - alpha) * this.performanceMetrics.averageConfidence + alpha * response.confidence;
    }
}

/**
 * Supporting interfaces for emergent agent infrastructure
 */
interface GoalRelevance {
    score: number;
    relevantAspects: string[];
    eventContext: string[];
    reasoning: string;
}

interface ImprovementOpportunity {
    type: "optimization" | "quality" | "security" | "feature";
    confidence: number;
    description: string;
    data: any;
}

interface ImprovementAnalysis {
    opportunity: ImprovementOpportunity;
    eventAnalysis: {
        relevance: GoalRelevance;
        patterns: EventPattern[];
    };
    confidence: number;
    reasoning: string;
}

interface RoutineProposal {
    id: string;
    agentId: string;
    targetRoutine: string;
    improvementType: "optimization" | "quality" | "security" | "feature";
    goal: string;
    analysis: ImprovementAnalysis;
    confidence: number;
    status: "proposed" | "accepted" | "rejected" | "implemented";
    createdAt: Date;
    acceptedAt?: Date;
    rejectedAt?: Date;
    rejectionReason?: string;
    estimatedImpact: ImpactEstimate;
}

interface EventPattern {
    pattern: string;
    occurrences: number;
    averageRelevance: number;
    improvementOpportunities: ImprovementOpportunity[];
    lastSeen: Date;
}

interface AgentLearningData {
    pattern: string;
    occurrences: number;
    averageRelevance: number;
    goalAlignment: Array<{
        score: number;
        aspects: string[];
        timestamp: Date;
    }>;
    lastSeen: Date;
}

interface AgentPerformanceMetrics {
    eventsProcessed: number;
    patternsLearned: number;
    routinesProposed: number;
    routinesAccepted: number;
    averageConfidence: number;
    lastUpdated: Date;
}

interface ImpactEstimate {
    performance: number;
    quality: number;
    security: number;
    cost: number;
}
