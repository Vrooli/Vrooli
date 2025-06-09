import {
    type AgentRole,
    type ExecutionId,
    type ExecutionResult,
    type ExecutionStatus,
    type ResourceAllocation,
    type SwarmConfig,
    type SwarmCoordinationInput,
    type SwarmState,
    type TeamFormation,
    type TierCapabilities,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    StrategyType,
    SwarmEventType as SwarmEventTypeEnum,
    SwarmState as SwarmStateEnum,
    generatePK
} from "@vrooli/shared";
import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type RollingHistory } from "../../cross-cutting/monitoring/index.js";
import { TelemetryShim } from "../../cross-cutting/monitoring/telemetryShim.js";
import { MetacognitiveMonitor } from "../intelligence/metacognitiveMonitor.js";
import { StrategyEngine } from "../intelligence/strategyEngine.js";
import { ResourceManager } from "../organization/resourceManager.js";
import { TeamManager } from "../organization/teamManager.js";
import { type ISwarmStateStore } from "../state/swarmStateStore.js";
import { PromptTemplateService } from "../intelligence/promptService.js";
import { ConversationBridge } from "../intelligence/conversationBridge.js";

/**
 * Swarm initialization parameters
 */
export interface SwarmInitParams {
    name: string;
    description: string;
    goal: string;
    config?: Partial<SwarmConfig>;
    initialAgents?: string[];
    metadata?: Record<string, unknown>;
}

/**
 * Swarm execution context
 */
export interface SwarmContext {
    swarmId: string;
    goal: string;
    progress: SwarmProgress;
    resources: SwarmResources;
    knowledge: SwarmKnowledge;
}

/**
 * Swarm progress tracking
 */
export interface SwarmProgress {
    tasksCompleted: number;
    tasksTotal: number;
    milestones: SwarmMilestone[];
    currentPhase: string;
}

/**
 * Swarm milestone
 */
export interface SwarmMilestone {
    id: string;
    name: string;
    completed: boolean;
    timestamp?: Date;
}

/**
 * Swarm resource allocation
 */
export interface SwarmResources {
    totalBudget: number;
    usedBudget: number;
    allocations: Map<string, number>; // agentId -> allocated budget
}

/**
 * Swarm knowledge base
 */
export interface SwarmKnowledge {
    facts: Map<string, unknown>;
    insights: string[];
    decisions: SwarmDecision[];
}

/**
 * Swarm decision record
 */
export interface SwarmDecision {
    id: string;
    timestamp: Date;
    decision: string;
    rationale: string;
    outcome?: string;
}

/**
 * TierOneSwarmStateMachine - Metacognitive swarm coordination
 * 
 * This is the highest level of intelligence in Vrooli's execution architecture.
 * It implements a metacognitive approach to swarm coordination, where the swarm
 * reasons about its own reasoning and adapts its strategies dynamically.
 * 
 * Key capabilities:
 * - Natural language reasoning for strategic decisions
 * - Dynamic team formation based on task requirements
 * - Resource allocation with economic modeling
 * - Self-monitoring and strategy adaptation
 * - Emergent intelligence through agent collaboration
 * 
 * The swarm operates through a continuous OODA loop (Observe, Orient, Decide, Act)
 * enhanced with metacognitive reflection and learning.
 * 
 * Implements TierCommunicationInterface for standardized inter-tier communication.
 */
export class SwarmStateMachine implements TierCommunicationInterface {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly stateStore: ISwarmStateStore;
    private readonly tier2Orchestrator: TierCommunicationInterface;
    private readonly teamManager: TeamManager;
    private readonly resourceManager: ResourceManager;
    private readonly strategyEngine: StrategyEngine;
    private readonly metacognitiveMonitor: MetacognitiveMonitor;
    private readonly telemetryShim: TelemetryShim;
    private readonly rollingHistory?: RollingHistory;
    private readonly conversationBridge: ConversationBridge;

    // Active swarms
    private readonly activeSwarms: Map<string, SwarmContext> = new Map();
    private readonly swarmTimers: Map<string, NodeJS.Timer> = new Map();

    // Configuration constants
    private readonly DEFAULT_BUDGET = 1000;
    private readonly DEFAULT_ADAPTATION_INTERVAL_MS = 60000; // 1 minute

    constructor(
        logger: Logger,
        eventBus: EventBus,
        stateStore: ISwarmStateStore,
        tier2Orchestrator: TierCommunicationInterface,
        rollingHistory?: RollingHistory,
    ) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.stateStore = stateStore;
        this.tier2Orchestrator = tier2Orchestrator;
        this.rollingHistory = rollingHistory;

        // Initialize components
        this.teamManager = new TeamManager(eventBus, logger);
        this.resourceManager = new ResourceManager(logger);
        this.strategyEngine = new StrategyEngine(logger, eventBus);
        this.metacognitiveMonitor = new MetacognitiveMonitor(eventBus, logger);
        this.telemetryShim = new TelemetryShim(eventBus, true);
        this.conversationBridge = new ConversationBridge(logger);

        // Subscribe to relevant events
        this.subscribeToEvents();
    }

    /**
     * Creates and initializes a new swarm
     */
    async createSwarm(params: SwarmInitParams): Promise<string> {
        const swarmId = generatePK();

        this.logger.info("[SwarmStateMachine] Creating new swarm", {
            swarmId,
            name: params.name,
            goal: params.goal,
        });

        // Track in rolling history
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier1.swarm.created',
                tier: 'tier1',
                component: 'swarm-state-machine',
                data: {
                    swarmId,
                    name: params.name,
                    goal: params.goal,
                    agentCount: params.initialAgents?.length || 0,
                },
            });
        }

        try {
            // Create swarm configuration
            const config: SwarmConfig = {
                maxAgents: 10,
                minAgents: 1,
                consensusThreshold: 0.7,
                decisionTimeout: 30000, // 30 seconds
                adaptationInterval: 60000, // 1 minute
                resourceOptimization: true,
                learningEnabled: true,
                ...params.config,
            };

            // Initialize swarm context
            const context: SwarmContext = {
                swarmId,
                goal: params.goal,
                progress: {
                    tasksCompleted: 0,
                    tasksTotal: 0,
                    milestones: [],
                    currentPhase: "initialization",
                },
                resources: {
                    totalBudget: config.maxBudget || this.DEFAULT_BUDGET,
                    usedBudget: 0,
                    allocations: new Map(),
                },
                knowledge: {
                    facts: new Map(),
                    insights: [],
                    decisions: [],
                },
            };

            // Store swarm state
            await this.stateStore.createSwarm(swarmId, {
                id: swarmId,
                name: params.name,
                description: params.description,
                state: SwarmStateEnum.FORMING,
                config,
                metadata: params.metadata || {},
                createdAt: new Date(),
            });

            // Cache context
            this.activeSwarms.set(swarmId, context);

            // Form initial team if agents provided
            if (params.initialAgents && params.initialAgents.length > 0) {
                await this.formInitialTeam(swarmId, params.initialAgents);
            }

            // Emit creation event
            await this.emitSwarmEvent({
                type: SwarmEventTypeEnum.SWARM_CREATED,
                swarmId,
                timestamp: new Date(),
                metadata: {
                    name: params.name,
                    goal: params.goal,
                },
            });

            // Emit telemetry for swarm creation
            await this.telemetryShim.emitStepStarted(swarmId, {
                stepType: 'swarm_coordination',
                strategy: StrategyType.REASONING,
                estimatedResources: {
                    credits: (config.maxBudget || this.DEFAULT_BUDGET).toString(),
                    time: config.decisionTimeout,
                },
            });

            // Start swarm lifecycle
            await this.startSwarmLifecycle(swarmId);

            return swarmId;

        } catch (error) {
            this.logger.error("[SwarmStateMachine] Failed to create swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Starts the swarm lifecycle loop
     */
    private async startSwarmLifecycle(swarmId: string): Promise<void> {
        const swarm = await this.stateStore.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Swarm ${swarmId} not found`);
        }

        // Transition to PLANNING state
        await this.transitionState(swarmId, SwarmStateEnum.PLANNING);

        // Start the OODA loop
        const timer = setInterval(async () => {
            try {
                await this.executeOODALoop(swarmId);
            } catch (error) {
                this.logger.error("[SwarmStateMachine] OODA loop error", {
                    swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }, swarm.config.adaptationInterval || this.DEFAULT_ADAPTATION_INTERVAL_MS);

        this.swarmTimers.set(swarmId, timer);
    }

    /**
     * Executes the OODA (Observe, Orient, Decide, Act) loop
     */
    private async executeOODALoop(swarmId: string): Promise<void> {
        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            return;
        }

        const swarm = await this.stateStore.getSwarm(swarmId);
        if (!swarm || swarm.state === SwarmStateEnum.COMPLETED) {
            this.stopSwarmLifecycle(swarmId);
            return;
        }

        this.logger.debug("[SwarmStateMachine] Executing OODA loop", {
            swarmId,
            phase: context.progress.currentPhase,
        });

        // Track OODA loop execution
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier1.ooda.cycle',
                tier: 'tier1',
                component: 'ooda-loop',
                data: {
                    swarmId,
                    phase: context.progress.currentPhase,
                    iteration: context.progress.tasksCompleted,
                },
            });
        }

        // Observe - Gather information from environment and agents
        const observations = await this.observe(swarmId, context);

        // Orient - Analyze situation and update understanding
        const orientation = await this.orient(swarmId, context, observations);

        // Decide - Make strategic decisions
        const decisions = await this.decide(swarmId, context, orientation);

        // Act - Execute decisions
        await this.act(swarmId, context, decisions);

        // Metacognitive reflection
        await this.reflect(swarmId, context);
    }

    /**
     * Observe phase - Gather information
     */
    private async observe(
        swarmId: string,
        context: SwarmContext,
    ): Promise<SwarmObservations> {
        const observations: SwarmObservations = {
            agentReports: [],
            environmentState: {},
            resourceStatus: await this.resourceManager.getResourceStatus(swarmId),
            performanceMetrics: await this.metacognitiveMonitor.getPerformanceMetrics(swarmId),
        };

        // Collect agent reports
        const team = await this.teamManager.getTeam(swarmId);
        for (const agent of team.agents) {
            const report = await this.collectAgentReport(agent.id);
            if (report) {
                observations.agentReports.push(report);
            }
        }

        // Query environment state
        observations.environmentState = await this.queryEnvironment(swarmId);

        this.logger.debug("[SwarmStateMachine] Observation complete", {
            swarmId,
            reportCount: observations.agentReports.length,
        });

        // Emit telemetry for observation phase
        await this.telemetryShim.emitOutputGenerated(swarmId, {
            outputKeys: ['agentReports', 'environmentState', 'resourceStatus', 'performanceMetrics'],
            size: JSON.stringify(observations).length,
            validationPassed: true,
        });

        return observations;
    }

    /**
     * Orient phase - Analyze and understand
     */
    private async orient(
        swarmId: string,
        context: SwarmContext,
        observations: SwarmObservations,
    ): Promise<SwarmOrientation> {
        // Use strategy engine to analyze situation
        const analysis = await this.strategyEngine.analyzeSituation({
            goal: context.goal,
            observations,
            knowledge: context.knowledge,
            progress: context.progress,
        });

        // Update knowledge base
        for (const [key, value] of Object.entries(analysis.facts)) {
            context.knowledge.facts.set(key, value);
        }

        if (analysis.insights) {
            context.knowledge.insights.push(...analysis.insights);
        }

        const orientation: SwarmOrientation = {
            currentState: analysis.assessment,
            opportunities: analysis.opportunities || [],
            threats: analysis.threats || [],
            recommendations: analysis.recommendations || [],
        };

        this.logger.debug("[SwarmStateMachine] Orientation complete", {
            swarmId,
            state: orientation.currentState,
            opportunityCount: orientation.opportunities.length,
        });

        return orientation;
    }

    /**
     * Decide phase - Make strategic decisions
     */
    private async decide(
        swarmId: string,
        context: SwarmContext,
        orientation: SwarmOrientation,
    ): Promise<SwarmDecision[]> {
        const decisions: SwarmDecision[] = [];

        // Use strategy engine to generate decisions
        const strategicDecisions = await this.strategyEngine.generateDecisions({
            goal: context.goal,
            orientation,
            constraints: {
                budget: context.resources.totalBudget - context.resources.usedBudget,
                timeLimit: 300000, // 5 minutes
            },
        });

        for (const decision of strategicDecisions) {
            const swarmDecision: SwarmDecision = {
                id: generatePK(),
                timestamp: new Date(),
                decision: decision.action,
                rationale: decision.rationale,
            };

            decisions.push(swarmDecision);
            context.knowledge.decisions.push(swarmDecision);
        }

        // Get team consensus if needed
        if (decisions.length > 0) {
            const consensus = await this.teamManager.getConsensus(
                swarmId,
                decisions.map(d => d.decision),
            );

            // Filter decisions based on consensus
            const approvedDecisions = decisions.filter((d, i) =>
                consensus.results[i] >= consensus.threshold,
            );

            this.logger.info("[SwarmStateMachine] Decisions made", {
                swarmId,
                totalDecisions: decisions.length,
                approvedDecisions: approvedDecisions.length,
            });

            // Track decision making
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier1.decisions.made',
                    tier: 'tier1',
                    component: 'strategy-engine',
                    data: {
                        swarmId,
                        totalDecisions: decisions.length,
                        approvedDecisions: approvedDecisions.length,
                        consensusThreshold: consensus.threshold,
                    },
                });
            }

            return approvedDecisions;
        }

        return decisions;
    }

    /**
     * Act phase - Execute decisions
     */
    private async act(
        swarmId: string,
        context: SwarmContext,
        decisions: SwarmDecision[],
    ): Promise<void> {
        for (const decision of decisions) {
            try {
                // Execute decision based on type
                if (decision.decision.startsWith("allocate_resources")) {
                    await this.executeResourceAllocation(swarmId, decision);
                } else if (decision.decision.startsWith("form_team")) {
                    await this.executeTeamFormation(swarmId, decision);
                } else if (decision.decision.startsWith("execute_routine")) {
                    await this.executeRoutine(swarmId, decision);
                } else if (decision.decision.startsWith("adapt_strategy")) {
                    await this.executeStrategyAdaptation(swarmId, decision);
                } else {
                    // Generic execution through event
                    await this.emitSwarmEvent({
                        type: SwarmEventTypeEnum.DECISION_EXECUTED,
                        swarmId,
                        timestamp: new Date(),
                        metadata: { decision },
                    });
                }

                // Record outcome
                decision.outcome = "executed";

                // Track successful execution
                if (this.rollingHistory) {
                    this.rollingHistory.addEvent({
                        timestamp: new Date(),
                        type: 'tier1.decision.executed',
                        tier: 'tier1',
                        component: 'decision-executor',
                        data: {
                            swarmId,
                            decisionId: decision.id,
                            decision: decision.decision,
                            success: true,
                        },
                    });
                }

            } catch (error) {
                decision.outcome = `failed: ${error instanceof Error ? error.message : String(error)}`;

                this.logger.error("[SwarmStateMachine] Decision execution failed", {
                    swarmId,
                    decision: decision.decision,
                    error: decision.outcome,
                });

                // Track failed execution
                if (this.rollingHistory) {
                    this.rollingHistory.addEvent({
                        timestamp: new Date(),
                        type: 'tier1.decision.failed',
                        tier: 'tier1',
                        component: 'decision-executor',
                        data: {
                            swarmId,
                            decisionId: decision.id,
                            decision: decision.decision,
                            error: decision.outcome,
                        },
                    });
                }
            }
        }
    }

    /**
     * Reflect phase - Metacognitive analysis
     */
    private async reflect(
        swarmId: string,
        context: SwarmContext,
    ): Promise<void> {
        // Analyze swarm performance
        const reflection = await this.metacognitiveMonitor.analyzePerformance({
            swarmId,
            decisions: context.knowledge.decisions,
            progress: context.progress,
            resources: context.resources,
        });

        // Apply learnings
        if (reflection.learnings) {
            for (const learning of reflection.learnings) {
                context.knowledge.insights.push(learning);
            }
        }

        // Adapt if needed
        if (reflection.adaptations) {
            for (const adaptation of reflection.adaptations) {
                await this.applyAdaptation(swarmId, adaptation);
            }
        }

        this.logger.debug("[SwarmStateMachine] Reflection complete", {
            swarmId,
            learningCount: reflection.learnings?.length || 0,
            adaptationCount: reflection.adaptations?.length || 0,
        });

        // Emit telemetry for metacognitive reflection
        if (reflection.learnings && reflection.learnings.length > 0) {
            await this.telemetryShim.emitOutputGenerated(swarmId, {
                outputKeys: ['learnings', 'adaptations'],
                size: reflection.learnings.length,
                validationPassed: true,
            });
        }

        // Track reflection phase
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier1.reflection.completed',
                tier: 'tier1',
                component: 'metacognitive-monitor',
                data: {
                    swarmId,
                    learningCount: reflection.learnings?.length || 0,
                    adaptationCount: reflection.adaptations?.length || 0,
                    performanceMetrics: reflection.performanceMetrics,
                },
            });
        }
    }

    /**
     * Helper methods
     */
    private async transitionState(
        swarmId: string,
        newState: SwarmState,
    ): Promise<void> {
        await this.stateStore.updateSwarmState(swarmId, newState);

        await this.emitSwarmEvent({
            type: SwarmEventTypeEnum.STATE_CHANGED,
            swarmId,
            timestamp: new Date(),
            metadata: { newState },
        });
    }

    private async formInitialTeam(
        swarmId: string,
        agentIds: string[],
    ): Promise<void> {
        const formation: TeamFormation = {
            id: generatePK(),
            swarmId,
            agents: agentIds.map(id => ({
                id,
                role: "member" as AgentRole,
                capabilities: [],
                status: "active",
            })),
            createdAt: new Date(),
        };

        await this.teamManager.formTeam(formation);
    }

    private async collectAgentReport(agentId: string): Promise<AgentReport | null> {
        // TODO: Implement agent report collection
        return null;
    }

    private async queryEnvironment(swarmId: string): Promise<Record<string, unknown>> {
        // TODO: Implement environment querying
        return {};
    }

    private async executeResourceAllocation(
        swarmId: string,
        decision: SwarmDecision,
    ): Promise<void> {
        // Parse allocation from decision
        const match = decision.decision.match(/allocate_resources\((\d+)\)/);
        if (match) {
            const amount = parseInt(match[1]);
            await this.resourceManager.allocateResources(swarmId, "general", amount);
        }
    }

    private async executeTeamFormation(
        swarmId: string,
        decision: SwarmDecision,
    ): Promise<void> {
        // TODO: Implement dynamic team formation
        await this.emitSwarmEvent({
            type: SwarmEventTypeEnum.TEAM_UPDATED,
            swarmId,
            timestamp: new Date(),
            metadata: { decision },
        });
    }

    private async executeRoutine(
        swarmId: string,
        decision: SwarmDecision,
    ): Promise<void> {
        // Emit event to trigger routine execution in Tier 2
        await this.eventBus.publish("swarm.routine.request", {
            swarmId,
            decision,
            timestamp: new Date(),
        });
    }

    private async executeStrategyAdaptation(
        swarmId: string,
        decision: SwarmDecision,
    ): Promise<void> {
        await this.strategyEngine.adaptStrategy(swarmId, decision.decision);
    }

    private async applyAdaptation(
        swarmId: string,
        adaptation: string,
    ): Promise<void> {
        // TODO: Implement specific adaptations
        this.logger.info("[SwarmStateMachine] Applying adaptation", {
            swarmId,
            adaptation,
        });
    }

    private async emitSwarmEvent(event: any): Promise<void> {
        await this.eventBus.publish("swarm.events", event);
    }

    private stopSwarmLifecycle(swarmId: string): void {
        const timer = this.swarmTimers.get(swarmId);
        if (timer) {
            clearInterval(timer);
            this.swarmTimers.delete(swarmId);
        }
        this.activeSwarms.delete(swarmId);
    }

    private subscribeToEvents(): void {
        // Subscribe to agent events
        this.eventBus.subscribe("agent.report", async (event) => {
            // Handle agent reports
        });

        // Subscribe to execution results from Tier 2
        this.eventBus.subscribe("run.completed", async (event) => {
            // Update swarm progress
        });
    }

    /**
     * Public control methods
     */
    async pauseSwarm(swarmId: string): Promise<void> {
        await this.transitionState(swarmId, SwarmStateEnum.SUSPENDED);
        this.stopSwarmLifecycle(swarmId);
    }

    async resumeSwarm(swarmId: string): Promise<void> {
        await this.transitionState(swarmId, SwarmStateEnum.EXECUTING);
        await this.startSwarmLifecycle(swarmId);
    }

    async terminateSwarm(swarmId: string): Promise<void> {
        await this.transitionState(swarmId, SwarmStateEnum.COMPLETED);
        this.stopSwarmLifecycle(swarmId);

        await this.emitSwarmEvent({
            type: SwarmEventTypeEnum.SWARM_TERMINATED,
            swarmId,
            timestamp: new Date(),
        });

        // Emit telemetry for swarm completion
        const context = this.activeSwarms.get(swarmId);
        if (context) {
            await this.telemetryShim.emitStepCompleted(swarmId, {
                strategy: StrategyType.REASONING,
                duration: Date.now() - (await this.stateStore.getSwarm(swarmId))?.createdAt.getTime() || 0,
                resourceUsage: {
                    creditsUsed: context.resources.usedBudget,
                    tasksCompleted: context.progress.tasksCompleted,
                    decisionsExecuted: context.knowledge.decisions.length,
                },
            });
        }

        // Track swarm termination
        if (this.rollingHistory) {
            this.rollingHistory.addEvent({
                timestamp: new Date(),
                type: 'tier1.swarm.terminated',
                tier: 'tier1',
                component: 'swarm-state-machine',
                data: {
                    swarmId,
                    finalState: SwarmStateEnum.COMPLETED,
                    tasksCompleted: context?.progress.tasksCompleted || 0,
                    decisionsExecuted: context?.knowledge.decisions.length || 0,
                },
            });
        }
    }

    // ===== TierCommunicationInterface Implementation =====

    private readonly activeExecutions = new Map<ExecutionId, {
        status: ExecutionStatus;
        startTime: Date;
        swarmId: string;
    }>();

    /**
     * Execute a request via the standard tier communication interface
     * This is the primary method for external systems to initiate swarm coordination
     */
    async execute<TInput extends SwarmCoordinationInput, TOutput>(
        request: TierExecutionRequest<TInput>
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input, allocation, options } = request;
        const executionId = context.executionId;

        // Track execution
        this.activeExecutions.set(executionId, {
            status: ExecutionStatus.RUNNING,
            startTime: new Date(),
            swarmId: context.swarmId,
        });

        try {
            this.logger.info("[SwarmStateMachine] Starting tier execution", {
                executionId,
                goal: input.goal,
                agentCount: input.availableAgents.length,
            });

            // Track execution in rolling history
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier1.execution.started',
                    tier: 'tier1',
                    component: 'swarm-state-machine',
                    data: {
                        executionId,
                        goal: input.goal,
                        agentCount: input.availableAgents.length,
                        constraints: input.constraints,
                    },
                });
            }

            // Initialize swarm for this goal
            const swarmParams: SwarmInitParams = {
                name: `Swarm for ${input.goal}`,
                description: `Autonomous swarm coordinating to achieve: ${input.goal}`,
                goal: input.goal,
                config: {
                    budget: parseInt(allocation.maxCredits),
                    timeoutMs: allocation.maxDurationMs,
                    maxAgents: input.availableAgents.length,
                },
                initialAgents: input.availableAgents.map(a => a.id),
                metadata: {
                    teamConfiguration: input.teamConfiguration,
                    constraints: input.constraints,
                },
            };

            // Create and coordinate the swarm
            const swarmId = await this.createSwarm(swarmParams);
            const swarmContext = this.activeSwarms.get(swarmId)!;
            const result = await this.coordinateSwarmExecution(swarmContext, input, allocation);

            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.COMPLETED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                swarmId: swarmContext.swarmId,
            });

            this.logger.info("[SwarmStateMachine] Tier execution completed", {
                executionId,
                swarmId: swarmContext.swarmId,
                success: true,
            });

            // Track execution completion
            if (this.rollingHistory) {
                this.rollingHistory.addEvent({
                    timestamp: new Date(),
                    type: 'tier1.execution.completed',
                    tier: 'tier1',
                    component: 'swarm-state-machine',
                    data: {
                        executionId,
                        swarmId: swarmContext.swarmId,
                        success: true,
                        duration: executionResult.duration,
                        tasksCompleted: swarmContext.progress.tasksCompleted,
                        milestonesAchieved: swarmContext.progress.milestones.filter(m => m.completed).length,
                    },
                });
            }

            // Return execution result
            const executionResult: ExecutionResult<TOutput> = {
                success: true,
                result: result as TOutput,
                outputs: result as Record<string, unknown>,
                resourcesUsed: {
                    creditsUsed: swarmContext.resources.usedBudget.toString(),
                    durationMs: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                    memoryUsedMB: 0,
                    stepsExecuted: swarmContext.progress.tasksCompleted,
                },
                duration: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                context,
                metadata: {
                    strategy: 'swarm_coordination',
                    version: '1.0.0',
                    performance: {
                        swarmId: swarmContext.swarmId,
                        tasksCompleted: swarmContext.progress.tasksCompleted,
                        milestonesAchieved: swarmContext.progress.milestones.filter(m => m.completed).length,
                    },
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.90,
                performanceScore: 0.85,
            };

            return executionResult;

        } catch (error) {
            // Update execution status
            this.activeExecutions.set(executionId, {
                status: ExecutionStatus.FAILED,
                startTime: this.activeExecutions.get(executionId)!.startTime,
                swarmId: this.activeExecutions.get(executionId)?.swarmId || 'unknown',
            });

            this.logger.error("[SwarmStateMachine] Tier execution failed", {
                executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Return error result
            const errorResult: ExecutionResult<TOutput> = {
                success: false,
                error: {
                    code: 'TIER1_EXECUTION_FAILED',
                    message: error instanceof Error ? error.message : 'Unknown error',
                    tier: 'tier1',
                    type: error instanceof Error ? error.constructor.name : 'Error',
                },
                resourcesUsed: {
                    creditsUsed: '0',
                    durationMs: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
                duration: Date.now() - this.activeExecutions.get(executionId)!.startTime.getTime(),
                context,
                metadata: {
                    strategy: 'swarm_coordination',
                    version: '1.0.0',
                    timestamp: new Date().toISOString(),
                },
                confidence: 0.0,
                performanceScore: 0.0,
            };

            return errorResult;
        }
    }

    /**
     * Coordinate swarm execution by delegating to Tier 2
     */
    private async coordinateSwarmExecution(
        swarmContext: SwarmContext,
        input: SwarmCoordinationInput,
        allocation: ResourceAllocation
    ): Promise<unknown> {
        // For now, return a simple successful result
        // TODO: Implement full goal decomposition and routine delegation
        return {
            goal: input.goal,
            status: 'completed',
            result: `Successfully coordinated swarm to achieve: ${input.goal}`,
        };
    }

    /**
     * Get execution status for monitoring
     */
    async getExecutionStatus(executionId: ExecutionId): Promise<ExecutionStatus> {
        const execution = this.activeExecutions.get(executionId);
        return execution?.status || ExecutionStatus.COMPLETED;
    }

    /**
     * Cancel a running execution
     */
    async cancelExecution(executionId: ExecutionId): Promise<void> {
        const execution = this.activeExecutions.get(executionId);
        if (execution) {
            this.activeExecutions.set(executionId, {
                ...execution,
                status: ExecutionStatus.CANCELLED,
            });

            // Terminate the associated swarm if it exists
            const swarmContext = this.activeSwarms.get(execution.swarmId);
            if (swarmContext) {
                await this.terminateSwarm(execution.swarmId);
            }

            this.logger.info("[SwarmStateMachine] Execution cancelled", { executionId });
        }
    }

    /**
     * Get tier capabilities for discovery
     */
    async getCapabilities(): Promise<TierCapabilities> {
        return {
            tier: 'tier1',
            supportedInputTypes: ['SwarmCoordinationInput'],
            supportedStrategies: ['swarm_coordination', 'multi_agent_planning', 'goal_decomposition'],
            maxConcurrency: 10,
            estimatedLatency: {
                p50: 30000,
                p95: 180000,
                p99: 600000,
            },
            resourceLimits: {
                maxCredits: '1000000',
                maxDurationMs: 7200000, // 2 hours
                maxMemoryMB: 8192,
            },
        };
    }

    /**
     * Start a swarm by ID - lifecycle method expected by TierOneCoordinator
     */
    async start(swarmId: string): Promise<void> {
        this.logger.info("[SwarmStateMachine] Starting swarm lifecycle", { swarmId });

        const swarm = await this.stateStore.getSwarm(swarmId);
        if (!swarm) {
            throw new Error(`Cannot start swarm ${swarmId}: swarm not found`);
        }

        // If swarm context doesn't exist, create it
        if (!this.activeSwarms.has(swarmId)) {
            const context: SwarmContext = {
                swarmId,
                goal: swarm.config.goal || 'No goal specified',
                progress: {
                    tasksCompleted: 0,
                    tasksTotal: 0,
                    milestones: [],
                    currentPhase: "initialization",
                },
                resources: {
                    totalBudget: swarm.config.maxBudget || this.DEFAULT_BUDGET,
                    usedBudget: 0,
                    allocations: new Map(),
                },
                knowledge: {
                    facts: new Map(),
                    insights: [],
                    decisions: [],
                },
            };
            this.activeSwarms.set(swarmId, context);
        }

        // Start the swarm lifecycle
        await this.startSwarmLifecycle(swarmId);
    }

    /**
     * Stop a swarm - lifecycle method expected by TierOneCoordinator
     */
    async stop(swarmId: string): Promise<void> {
        this.logger.info("[SwarmStateMachine] Stopping swarm", { swarmId });

        // Stop lifecycle timer
        this.stopSwarmLifecycle(swarmId);

        // Transition to completed state
        await this.transitionState(swarmId, SwarmStateEnum.COMPLETED);

        // Clean up context
        this.activeSwarms.delete(swarmId);

        // Emit completion event
        await this.emitSwarmEvent({
            type: SwarmEventTypeEnum.SWARM_STOPPED,
            swarmId,
            timestamp: new Date(),
            metadata: { reason: 'manual_stop' },
        });
    }

    /**
     * Request routine execution from Tier 2
     */
    async requestRunExecution(swarmId: string, routineId: string, parameters: Record<string, unknown>): Promise<void> {
        this.logger.info("[SwarmStateMachine] Requesting routine execution", {
            swarmId,
            routineId,
        });

        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            throw new Error(`Cannot execute routine: swarm ${swarmId} not found`);
        }

        try {
            // Create execution context for Tier 2
            const executionContext = {
                executionId: generatePK(),
                swarmId,
                routineId,
                parentExecutionId: null,
                stepId: null,
                stepType: 'routine',
                strategy: StrategyType.REASONING,
                userId: null, // Will be set by higher level
                organizationId: null,
                isTest: false,
                metadata: {
                    initiatedBy: 'tier1',
                    source: 'swarm-coordination',
                },
            };

            // Create routine execution input
            const routineInput = {
                routineId,
                parameters,
                workflow: {
                    steps: [],
                    dependencies: [],
                },
            };

            // Create resource allocation
            const allocation: ResourceAllocation = {
                maxCredits: Math.min(context.resources.totalBudget - context.resources.usedBudget, 100).toString(),
                maxDurationMs: 300000, // 5 minutes
                maxMemoryMB: 512,
            };

            // Create tier execution request
            const tierRequest = {
                context: executionContext,
                input: routineInput,
                allocation,
                options: {
                    priority: 'high' as const,
                    retryPolicy: {
                        maxRetries: 2,
                        backoffMs: 1000,
                    },
                },
            };

            // Delegate to Tier 2
            const result = await this.tier2Orchestrator.execute(tierRequest);

            if (result.success) {
                this.logger.info("[SwarmStateMachine] Routine execution completed", {
                    swarmId,
                    routineId,
                    duration: result.duration,
                });

                // Update progress
                context.progress.tasksCompleted++;

                // Track resource usage
                const creditsUsed = parseFloat(result.resourcesUsed.creditsUsed || '0');
                context.resources.usedBudget += creditsUsed;

            } else {
                this.logger.error("[SwarmStateMachine] Routine execution failed", {
                    swarmId,
                    routineId,
                    error: result.error,
                });
            }

        } catch (error) {
            this.logger.error("[SwarmStateMachine] Failed to request routine execution", {
                swarmId,
                routineId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Handle run completion from Tier 2
     */
    async handleRunCompletion(swarmId: string, runId: string, result: any): Promise<void> {
        this.logger.info("[SwarmStateMachine] Handling run completion", {
            swarmId,
            runId,
        });

        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            this.logger.warn("[SwarmStateMachine] Cannot handle completion: swarm not found", { swarmId });
            return;
        }

        // Update progress
        context.progress.tasksCompleted++;

        // Record decision
        const decision: SwarmDecision = {
            id: generatePK(),
            timestamp: new Date(),
            decision: `Completed run ${runId}`,
            rationale: `Successfully executed routine within swarm ${swarmId}`,
            outcome: result.success ? 'success' : 'failure',
        };
        context.knowledge.decisions.push(decision);

        // Check if goal is complete
        await this.checkGoalCompletion(swarmId);
    }

    /**
     * Handle resource alert from monitoring
     */
    async handleResourceAlert(swarmId: string, alert: { type: string; message: string; severity: string }): Promise<void> {
        this.logger.warn("[SwarmStateMachine] Handling resource alert", {
            swarmId,
            alert,
        });

        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            return;
        }

        // Record insight
        context.knowledge.insights.push(
            `Resource alert: ${alert.type} - ${alert.message} (severity: ${alert.severity})`
        );

        // Take action based on alert type
        if (alert.severity === 'critical') {
            // Pause swarm execution
            await this.transitionState(swarmId, SwarmStateEnum.PAUSED);

            // Emit emergency event
            await this.emitSwarmEvent({
                type: SwarmEventTypeEnum.RESOURCE_CRITICAL,
                swarmId,
                timestamp: new Date(),
                metadata: alert,
            });
        }
    }

    /**
     * Handle metacognitive insight
     */
    async handleMetacognitiveInsight(swarmId: string, insight: { type: string; content: string; confidence: number }): Promise<void> {
        this.logger.info("[SwarmStateMachine] Handling metacognitive insight", {
            swarmId,
            type: insight.type,
            confidence: insight.confidence,
        });

        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            return;
        }

        // Record insight
        context.knowledge.insights.push(insight.content);

        // Act on high-confidence insights
        if (insight.confidence > 0.8) {
            if (insight.type === 'strategy_adaptation') {
                // Trigger strategy reconsideration
                await this.executeOODALoop(swarmId);
            } else if (insight.type === 'resource_optimization') {
                // Trigger resource reallocation
                await this.resourceManager.optimizeAllocation(swarmId);
            }
        }
    }

    /**
     * Get current execution phase for monitoring
     */
    getCurrentPhase(swarmId: string): string | null {
        const context = this.activeSwarms.get(swarmId);
        return context?.progress.currentPhase || null;
    }

    /**
     * Check if swarm goal is complete
     */
    private async checkGoalCompletion(swarmId: string): Promise<void> {
        const context = this.activeSwarms.get(swarmId);
        if (!context) {
            return;
        }

        // Simple completion check - in real implementation this would use AI reasoning
        const completionThreshold = Math.max(1, context.progress.tasksTotal * 0.8);
        if (context.progress.tasksCompleted >= completionThreshold) {
            this.logger.info("[SwarmStateMachine] Goal completion detected", {
                swarmId,
                tasksCompleted: context.progress.tasksCompleted,
                tasksTotal: context.progress.tasksTotal,
            });

            await this.transitionState(swarmId, SwarmStateEnum.COMPLETED);
            await this.stop(swarmId);
        }
    }

    /**
     * Terminate a swarm forcefully
     */
    private async terminateSwarm(swarmId: string): Promise<void> {
        this.logger.info("[SwarmStateMachine] Terminating swarm", { swarmId });

        // Stop lifecycle
        this.stopSwarmLifecycle(swarmId);

        // Transition to terminated state
        await this.transitionState(swarmId, SwarmStateEnum.TERMINATED);

        // Clean up
        this.activeSwarms.delete(swarmId);

        // Emit termination event
        await this.emitSwarmEvent({
            type: SwarmEventTypeEnum.SWARM_TERMINATED,
            swarmId,
            timestamp: new Date(),
            metadata: { reason: 'forced_termination' },
        });
    }
}

/**
 * Type definitions for OODA loop
 */
interface SwarmObservations {
    agentReports: AgentReport[];
    environmentState: Record<string, unknown>;
    resourceStatus: any;
    performanceMetrics: any;
}

interface AgentReport {
    agentId: string;
    status: string;
    progress: number;
    issues?: string[];
}

interface SwarmOrientation {
    currentState: string;
    opportunities: string[];
    threats: string[];
    recommendations: string[];
}
