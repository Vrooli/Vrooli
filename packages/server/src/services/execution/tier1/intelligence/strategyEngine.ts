import { type Logger } from "winston";
import {
    type SwarmProgress,
    type SwarmKnowledge,
    type SwarmDecision,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { LLMService } from "../llm/llmService.js";

/**
 * Situation analysis input
 */
export interface SituationAnalysisInput {
    goal: string;
    observations: any;
    knowledge: SwarmKnowledge;
    progress: SwarmProgress;
}

/**
 * Decision generation input
 */
export interface DecisionGenerationInput {
    goal: string;
    orientation: any;
    constraints: {
        budget?: number;
        timeLimit?: number;
        resources?: Record<string, number>;
    };
}

/**
 * StrategyEngine - Natural language reasoning interface for Tier 1
 * 
 * This component provides a simple interface to LLM-based reasoning.
 * All intelligence comes from the LLM's natural language understanding,
 * not from hard-coded patterns or algorithms.
 * 
 * The engine:
 * - Formats prompts for LLM reasoning
 * - Sends reasoning requests to LLM service
 * - Parses LLM responses into structured decisions
 * - Emits events for agent analysis
 */
export class StrategyEngine {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly llmService: LLMService;

    constructor(logger: Logger, eventBus: EventBus, llmService: LLMService) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.llmService = llmService;
    }

    /**
     * Analyzes the current situation using LLM reasoning
     */
    async analyzeSituation(
        input: SituationAnalysisInput,
    ): Promise<string> {
        this.logger.debug("[StrategyEngine] Analyzing situation", {
            goal: input.goal,
            observationCount: Object.keys(input.observations).length,
        });

        // Build natural language prompt
        const prompt = `
You are a strategic coordinator analyzing a swarm execution.

Goal: ${input.goal}

Current Progress:
- Tasks Completed: ${input.progress.tasksCompleted}/${input.progress.tasksTotal}
- Current Phase: ${input.progress.currentPhase}
- Milestones: ${input.progress.milestones.filter(m => m.completed).length}/${input.progress.milestones.length} completed

Observations:
${JSON.stringify(input.observations, null, 2)}

Provide a strategic assessment of the current situation.`;

        // Emit analysis request for monitoring agents
        await this.eventBus.publish("swarm.events", {
            type: "STRATEGY_ANALYSIS_REQUEST",
            swarmId: input.knowledge.swarmId,
            timestamp: new Date(),
            metadata: { prompt, input },
        });

        // Use LLM service for strategic analysis
        return await this.llmService.analyzeStrategically(
            `Goal: ${input.goal}\nProgress: ${JSON.stringify(input.progress)}\nObservations: ${JSON.stringify(input.observations)}`,
            "Provide a strategic assessment of the current situation"
        );
    }

    /**
     * Generates strategic decisions using LLM reasoning
     */
    async generateDecisions(
        input: DecisionGenerationInput,
    ): Promise<SwarmDecision[]> {
        this.logger.debug("[StrategyEngine] Generating decisions", {
            goal: input.goal,
            constraints: input.constraints,
        });

        // Build natural language prompt
        const prompt = `
You are a strategic coordinator making decisions for a swarm.

Goal: ${input.goal}

Current Orientation: ${JSON.stringify(input.orientation, null, 2)}

Constraints:
- Budget: ${input.constraints.budget || "unlimited"}
- Time Limit: ${input.constraints.timeLimit ? `${input.constraints.timeLimit}ms` : "none"}
- Resources: ${JSON.stringify(input.constraints.resources || {})}

Generate strategic decisions in the format:
action_name(parameters)

Provide 1-3 high-priority decisions with brief rationale.`;

        // Emit decision request for monitoring agents
        await this.eventBus.publish("swarm.events", {
            type: "STRATEGY_DECISION_REQUEST",
            timestamp: new Date(),
            metadata: { prompt, input },
        });

        // Use LLM service for decision generation
        const response = await this.llmService.generateDecision(
            input.goal,
            input.constraints,
            ["allocate_resources", "execute_routine", "form_team", "adapt_strategy"]
        );
        
        return this.parseDecisionsFromText(response);
    }

    /**
     * Requests strategy adaptation through LLM reasoning
     */
    async requestAdaptation(
        swarmId: string,
        context: any,
    ): Promise<string> {
        this.logger.info("[StrategyEngine] Requesting strategy adaptation", {
            swarmId,
        });

        const prompt = `
You are a strategic coordinator evaluating swarm performance.

Context: ${JSON.stringify(context, null, 2)}

Based on the current performance and challenges, what strategic adaptations would you recommend?

Focus on high-level strategic changes, not implementation details.`;

        // Emit adaptation request for monitoring agents
        await this.eventBus.publish("swarm.events", {
            type: "STRATEGY_ADAPTATION_REQUEST",
            swarmId,
            timestamp: new Date(),
            metadata: { prompt, context },
        });

        // Use LLM service for adaptation reasoning
        return await this.llmService.analyzeStrategically(
            JSON.stringify(context),
            "What strategic adaptations would you recommend based on current performance?"
        );
    }

    /**
     * Records decision outcome for agent analysis
     */
    async recordOutcome(
        decision: SwarmDecision,
        outcome: "success" | "failure" | "partial",
    ): Promise<void> {
        this.logger.debug("[StrategyEngine] Recording outcome", {
            decision: decision.decision,
            outcome,
        });

        // Emit outcome event for learning agents
        await this.eventBus.publish("swarm.events", {
            type: "STRATEGY_DECISION_OUTCOME",
            timestamp: new Date(),
            metadata: {
                decision,
                outcome,
            },
        });
    }

    /**
     * Private helper methods
     */
    private parseDecisionsFromText(text: string): SwarmDecision[] {
        // Simple parser for LLM output
        // In production, use more sophisticated parsing
        const decisions: SwarmDecision[] = [];
        const lines = text.split('\n').filter(line => line.trim());
        
        for (const line of lines) {
            // Look for action_name(parameters) format
            const match = line.match(/^(\w+)\(([^)]*)\)/);
            if (match) {
                decisions.push({
                    id: `decision-${Date.now()}-${Math.random()}`,
                    decision: match[0],
                    timestamp: new Date(),
                    confidence: 0.8, // LLM decisions have high confidence
                });
            }
        }
        
        return decisions;
    }

}
