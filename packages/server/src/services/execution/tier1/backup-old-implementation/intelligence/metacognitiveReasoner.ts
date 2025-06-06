import { v4 as uuidv4 } from "uuid";
import { logger } from "@local/server/src/utils/logger.js";
import { 
    SwarmState,
    TeamStructure,
    AgentRole,
    BlackboardItem,
    ResourceAvailability,
} from "@local/shared/src/execution/types/swarm.js";
import { LlmService } from "@local/server/src/services/llm/index.js";

/**
 * Metacognitive Reasoner for Tier 1
 * 
 * Provides natural language reasoning about:
 * - Goal decomposition and planning
 * - Progress monitoring and evaluation
 * - Strategy adaptation and learning
 * - Failure analysis and recovery
 * 
 * This component embodies the "thinking about thinking" capability
 * that makes the swarm truly intelligent and adaptive.
 */
export class MetacognitiveReasoner {
    private llmService: LlmService;
    private monitoringInterval: NodeJS.Timeout | null = null;

    constructor() {
        this.llmService = new LlmService();
    }

    /**
     * Creates an execution plan for achieving the goal
     */
    async createExecutionPlan(
        goal: string,
        teams: TeamStructure[],
        availableResources: ResourceAvailability[],
    ): Promise<any> {
        const prompt = this.buildPlanningPrompt(goal, teams, availableResources);
        
        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        plan: {
                            type: "object",
                            properties: {
                                overview: { type: "string" },
                                phases: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            name: { type: "string" },
                                            objective: { type: "string" },
                                            tasks: {
                                                type: "array",
                                                items: {
                                                    type: "object",
                                                    properties: {
                                                        id: { type: "string" },
                                                        description: { type: "string" },
                                                        assignedTeam: { type: "string" },
                                                        dependencies: {
                                                            type: "array",
                                                            items: { type: "string" },
                                                        },
                                                        estimatedDuration: { type: "string" },
                                                        requiredResources: {
                                                            type: "array",
                                                            items: { type: "string" },
                                                        },
                                                    },
                                                },
                                            },
                                            successCriteria: { type: "string" },
                                        },
                                    },
                                },
                                risks: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            description: { type: "string" },
                                            likelihood: { type: "string" },
                                            impact: { type: "string" },
                                            mitigation: { type: "string" },
                                        },
                                    },
                                },
                                reasoning: { type: "string" },
                            },
                        },
                    },
                },
            });

            logger.info("Created execution plan", { goal, phaseCount: response.plan.phases.length });
            return response.plan;

        } catch (error) {
            logger.error("Failed to create execution plan", error);
            throw error;
        }
    }

    /**
     * Evaluates progress towards the goal
     */
    async evaluateProgress(
        state: SwarmState,
        progress: any,
    ): Promise<boolean> {
        const prompt = this.buildProgressEvaluationPrompt(state, progress);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        evaluation: {
                            type: "object",
                            properties: {
                                onTrack: { type: "boolean" },
                                progressPercentage: { type: "number" },
                                concerns: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            issue: { type: "string" },
                                            severity: { type: "string" },
                                            recommendation: { type: "string" },
                                        },
                                    },
                                },
                                shouldAdapt: { type: "boolean" },
                                reasoning: { type: "string" },
                            },
                        },
                    },
                },
            });

            // Log significant concerns
            if (response.evaluation.concerns.length > 0) {
                logger.warn("Progress evaluation concerns", {
                    swarmId: state.id,
                    concerns: response.evaluation.concerns,
                });
            }

            return response.evaluation.shouldAdapt;

        } catch (error) {
            logger.error("Failed to evaluate progress", error);
            return false;
        }
    }

    /**
     * Evaluates results to determine if goal was achieved
     */
    async evaluateResults(
        state: SwarmState,
        results: any,
    ): Promise<any> {
        const prompt = this.buildResultEvaluationPrompt(state, results);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        goalAchieved: { type: "boolean" },
                        completionPercentage: { type: "number" },
                        achievements: {
                            type: "array",
                            items: { type: "string" },
                        },
                        gaps: {
                            type: "array",
                            items: { type: "string" },
                        },
                        lessons: {
                            type: "array",
                            items: {
                                type: "object",
                                properties: {
                                    insight: { type: "string" },
                                    applicability: { type: "string" },
                                },
                            },
                        },
                        nextSteps: {
                            type: "array",
                            items: { type: "string" },
                        },
                        reasoning: { type: "string" },
                    },
                },
            });

            logger.info("Evaluated results", {
                swarmId: state.id,
                goalAchieved: response.goalAchieved,
                completionPercentage: response.completionPercentage,
            });

            return response;

        } catch (error) {
            logger.error("Failed to evaluate results", error);
            throw error;
        }
    }

    /**
     * Analyzes failure to determine recovery options
     */
    async analyzeFailure(
        state: SwarmState,
        error: any,
    ): Promise<any> {
        const prompt = this.buildFailureAnalysisPrompt(state, error);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        recoverable: { type: "boolean" },
                        rootCause: { type: "string" },
                        contributingFactors: {
                            type: "array",
                            items: { type: "string" },
                        },
                        recoveryOptions: {
                            type: "array",
                            items: {
                                type: "object",
                                properties: {
                                    strategy: { type: "string" },
                                    likelihood: { type: "string" },
                                    requirements: {
                                        type: "array",
                                        items: { type: "string" },
                                    },
                                },
                            },
                        },
                        preventionMeasures: {
                            type: "array",
                            items: { type: "string" },
                        },
                        reasoning: { type: "string" },
                    },
                },
            });

            logger.info("Analyzed failure", {
                swarmId: state.id,
                recoverable: response.recoverable,
                rootCause: response.rootCause,
            });

            return response;

        } catch (error) {
            logger.error("Failed to analyze failure", error);
            return { recoverable: false, rootCause: "Analysis failed" };
        }
    }

    /**
     * Recommends adaptations based on current state
     */
    async recommendAdaptations(state: SwarmState): Promise<any[]> {
        const prompt = this.buildAdaptationPrompt(state);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        adaptations: {
                            type: "array",
                            items: {
                                type: "object",
                                properties: {
                                    type: { 
                                        type: "string",
                                        enum: ["reorganize_teams", "reallocate_resources", "change_approach"],
                                    },
                                    priority: {
                                        type: "string",
                                        enum: ["high", "medium", "low"],
                                    },
                                    description: { type: "string" },
                                    params: { type: "object" },
                                    expectedImpact: { type: "string" },
                                },
                            },
                        },
                        reasoning: { type: "string" },
                    },
                },
            });

            logger.info("Recommended adaptations", {
                swarmId: state.id,
                count: response.adaptations.length,
            });

            return response.adaptations;

        } catch (error) {
            logger.error("Failed to recommend adaptations", error);
            return [];
        }
    }

    /**
     * Plans next steps after evaluating results
     */
    async planNextSteps(
        state: SwarmState,
        evaluation: any,
    ): Promise<any> {
        const prompt = this.buildNextStepsPrompt(state, evaluation);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        plan: {
                            type: "object",
                            properties: {
                                objective: { type: "string" },
                                approach: { type: "string" },
                                tasks: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            id: { type: "string" },
                                            description: { type: "string" },
                                            priority: { type: "string" },
                                            estimatedDuration: { type: "string" },
                                        },
                                    },
                                },
                                successCriteria: { type: "string" },
                                reasoning: { type: "string" },
                            },
                        },
                    },
                },
            });

            logger.info("Planned next steps", {
                swarmId: state.id,
                taskCount: response.plan.tasks.length,
            });

            return response.plan;

        } catch (error) {
            logger.error("Failed to plan next steps", error);
            throw error;
        }
    }

    /**
     * Creates a recovery plan after failure
     */
    async createRecoveryPlan(
        state: SwarmState,
        failureAnalysis: any,
    ): Promise<any> {
        const prompt = this.buildRecoveryPlanPrompt(state, failureAnalysis);

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        plan: {
                            type: "object",
                            properties: {
                                strategy: { type: "string" },
                                immediateActions: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            action: { type: "string" },
                                            responsible: { type: "string" },
                                            deadline: { type: "string" },
                                        },
                                    },
                                },
                                checkpoints: {
                                    type: "array",
                                    items: {
                                        type: "object",
                                        properties: {
                                            milestone: { type: "string" },
                                            criteria: { type: "string" },
                                        },
                                    },
                                },
                                fallbackOption: { type: "string" },
                                reasoning: { type: "string" },
                            },
                        },
                    },
                },
            });

            logger.info("Created recovery plan", {
                swarmId: state.id,
                strategy: response.plan.strategy,
            });

            return response.plan;

        } catch (error) {
            logger.error("Failed to create recovery plan", error);
            throw error;
        }
    }

    /**
     * Starts continuous monitoring of swarm execution
     */
    async startMonitoring(state: SwarmState): Promise<void> {
        // Clear any existing monitoring
        this.stopMonitoring();

        // Set up periodic monitoring
        this.monitoringInterval = setInterval(async () => {
            try {
                await this.performMonitoringCycle(state);
            } catch (error) {
                logger.error("Monitoring cycle failed", error);
            }
        }, 30000); // Every 30 seconds

        logger.info(`Started monitoring for swarm ${state.id}`);
    }

    /**
     * Stops monitoring
     */
    stopMonitoring(): void {
        if (this.monitoringInterval) {
            clearInterval(this.monitoringInterval);
            this.monitoringInterval = null;
        }
    }

    /**
     * Performs a monitoring cycle
     */
    private async performMonitoringCycle(state: SwarmState): Promise<void> {
        // Analyze recent blackboard entries
        const recentItems = state.blackboard
            .filter(item => {
                const itemTime = new Date(item.timestamp).getTime();
                const fiveMinutesAgo = Date.now() - 5 * 60 * 1000;
                return itemTime > fiveMinutesAgo;
            });

        if (recentItems.length === 0) {
            return; // No recent activity
        }

        const prompt = `
As a metacognitive monitor, analyze the recent swarm activity and identify any patterns or concerns.

Recent blackboard items:
${JSON.stringify(recentItems, null, 2)}

Identify:
1. Activity patterns
2. Potential issues
3. Optimization opportunities
4. Team dynamics
`;

        try {
            const response = await this.llmService.generateStructuredResponse({
                prompt,
                schema: {
                    type: "object",
                    properties: {
                        patterns: {
                            type: "array",
                            items: { type: "string" },
                        },
                        concerns: {
                            type: "array",
                            items: {
                                type: "object",
                                properties: {
                                    issue: { type: "string" },
                                    severity: { type: "string" },
                                },
                            },
                        },
                        opportunities: {
                            type: "array",
                            items: { type: "string" },
                        },
                        requiresIntervention: { type: "boolean" },
                    },
                },
            });

            if (response.requiresIntervention) {
                logger.warn("Monitoring detected issues requiring intervention", {
                    swarmId: state.id,
                    concerns: response.concerns,
                });
            }

        } catch (error) {
            logger.error("Monitoring analysis failed", error);
        }
    }

    // Prompt building methods

    private buildPlanningPrompt(
        goal: string,
        teams: TeamStructure[],
        resources: ResourceAvailability[],
    ): string {
        return `
You are a metacognitive planner for an AI swarm. Create a comprehensive execution plan.

Goal: ${goal}

Available Teams:
${JSON.stringify(teams, null, 2)}

Available Resources:
${JSON.stringify(resources, null, 2)}

Create a detailed plan that:
1. Breaks down the goal into phases
2. Assigns tasks to appropriate teams
3. Identifies dependencies and risks
4. Estimates time and resource requirements
5. Defines clear success criteria

Focus on leveraging team strengths and managing resources efficiently.
`;
    }

    private buildProgressEvaluationPrompt(state: SwarmState, progress: any): string {
        return `
You are a metacognitive evaluator. Assess the swarm's progress towards its goal.

Goal: ${state.goal}
Current Phase: ${state.phase}
Progress Data: ${JSON.stringify(progress, null, 2)}
Performance Metrics: ${JSON.stringify(state.performance, null, 2)}

Evaluate:
1. Whether progress is on track
2. Any emerging concerns or bottlenecks
3. If strategy adaptation is needed
4. Overall health of the swarm execution

Be analytical and identify both positive trends and potential issues.
`;
    }

    private buildResultEvaluationPrompt(state: SwarmState, results: any): string {
        return `
You are a metacognitive evaluator. Assess whether the swarm achieved its goal.

Goal: ${state.goal}
Results: ${JSON.stringify(results, null, 2)}
Performance: ${JSON.stringify(state.performance, null, 2)}

Determine:
1. If the goal was fully achieved
2. What was accomplished successfully
3. Any gaps or incomplete aspects
4. Lessons learned for future swarms
5. Recommended next steps

Be thorough and objective in your assessment.
`;
    }

    private buildFailureAnalysisPrompt(state: SwarmState, error: any): string {
        return `
You are a metacognitive analyst. Analyze this swarm failure to determine recovery options.

Goal: ${state.goal}
Error: ${JSON.stringify(error, null, 2)}
Current State: Phase ${state.phase}, ${state.agents.length} agents
Recent Activity: ${state.blackboard.slice(-5).map(item => item.type).join(", ")}

Analyze:
1. Root cause of the failure
2. Contributing factors
3. Whether recovery is possible
4. Recovery strategies if applicable
5. Prevention measures for future

Be thorough in identifying both immediate and systemic issues.
`;
    }

    private buildAdaptationPrompt(state: SwarmState): string {
        return `
You are a metacognitive strategist. Recommend adaptations to improve swarm performance.

Current State:
- Goal: ${state.goal}
- Phase: ${state.phase}
- Teams: ${state.teams.length}
- Active Agents: ${state.agents.filter(a => a.state === "active").length}
- Resource Usage: ${JSON.stringify(state.resources.usage, null, 2)}
- Performance: ${JSON.stringify(state.performance, null, 2)}

Recent Blackboard Items:
${state.blackboard.slice(-10).map(item => `- ${item.type}: ${item.author}`).join("\n")}

Recommend adaptations that could improve:
1. Team organization and coordination
2. Resource allocation and efficiency
3. Overall approach to the goal
4. Communication patterns

Focus on actionable changes with clear expected benefits.
`;
    }

    private buildNextStepsPrompt(state: SwarmState, evaluation: any): string {
        return `
You are a metacognitive planner. Based on the evaluation, plan the next steps.

Goal: ${state.goal}
Evaluation Summary:
- Goal Achieved: ${evaluation.goalAchieved}
- Completion: ${evaluation.completionPercentage}%
- Gaps: ${evaluation.gaps.join(", ")}

Current Resources:
- Available: ${state.resources.available.length} resources
- Teams: ${state.teams.length} active

Plan the next phase that:
1. Addresses identified gaps
2. Builds on achievements
3. Uses resources efficiently
4. Has clear success criteria

Focus on concrete, achievable tasks.
`;
    }

    private buildRecoveryPlanPrompt(state: SwarmState, analysis: any): string {
        return `
You are a metacognitive recovery specialist. Create a recovery plan based on the failure analysis.

Goal: ${state.goal}
Failure Analysis:
- Root Cause: ${analysis.rootCause}
- Recoverable: ${analysis.recoverable}
- Options: ${analysis.recoveryOptions.map((o: any) => o.strategy).join(", ")}

Current State:
- Teams: ${state.teams.length}
- Resources: ${state.resources.available.length}

Create a recovery plan that:
1. Addresses the root cause
2. Implements immediate stabilization
3. Defines clear checkpoints
4. Includes a fallback option

Make the plan concrete and actionable.
`;
    }
}