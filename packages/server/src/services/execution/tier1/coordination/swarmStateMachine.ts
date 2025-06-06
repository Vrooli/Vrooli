import { type Logger } from "winston";
import {
    type SwarmState,
    type SwarmConfig,
    type SwarmEventType,
    type AgentRole,
    type TeamFormation,
    SwarmState as SwarmStateEnum,
    SwarmEventType as SwarmEventTypeEnum,
    generatePk,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { TeamManager } from "../organization/teamManager.js";
import { ResourceManager } from "../organization/resourceManager.js";
import { StrategyEngine } from "../intelligence/strategyEngine.js";
import { MetacognitiveMonitor } from "../intelligence/metacognitiveMonitor.js";
import { type ISwarmStateStore } from "../state/swarmStateStore.js";

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
 */
export class TierOneSwarmStateMachine {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly stateStore: ISwarmStateStore;
    private readonly teamManager: TeamManager;
    private readonly resourceManager: ResourceManager;
    private readonly strategyEngine: StrategyEngine;
    private readonly metacognitiveMonitor: MetacognitiveMonitor;

    // Active swarms
    private readonly activeSwarms: Map<string, SwarmContext> = new Map();
    private readonly swarmTimers: Map<string, NodeJS.Timer> = new Map();

    constructor(
        logger: Logger,
        eventBus: EventBus,
        stateStore: ISwarmStateStore,
    ) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.stateStore = stateStore;

        // Initialize components
        this.teamManager = new TeamManager(eventBus, logger);
        this.resourceManager = new ResourceManager(logger);
        this.strategyEngine = new StrategyEngine(logger);
        this.metacognitiveMonitor = new MetacognitiveMonitor(eventBus, logger);

        // Subscribe to relevant events
        this.subscribeToEvents();
    }

    /**
     * Creates and initializes a new swarm
     */
    async createSwarm(params: SwarmInitParams): Promise<string> {
        const swarmId = generatePk();
        
        this.logger.info("[SwarmStateMachine] Creating new swarm", {
            swarmId,
            name: params.name,
            goal: params.goal,
        });

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
                    totalBudget: config.maxBudget || 1000,
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
        }, swarm.config.adaptationInterval || 60000);

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
                id: generatePk(),
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
                consensus.results[i] >= consensus.threshold
            );

            this.logger.info("[SwarmStateMachine] Decisions made", {
                swarmId,
                totalDecisions: decisions.length,
                approvedDecisions: approvedDecisions.length,
            });

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

            } catch (error) {
                decision.outcome = `failed: ${error instanceof Error ? error.message : String(error)}`;
                
                this.logger.error("[SwarmStateMachine] Decision execution failed", {
                    swarmId,
                    decision: decision.decision,
                    error: decision.outcome,
                });
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
            id: generatePk(),
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