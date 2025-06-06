import { v4 as uuidv4 } from "uuid";
import { logger } from "@local/server/src/utils/logger.js";
import { EventBus } from "@local/shared/src/execution/events/eventBus.js";
import { 
    SwarmState, 
    SwarmPhase, 
    AgentState, 
    BlackboardItem,
    ResourceAllocation,
    SwarmConfig,
    AgentRole,
    TeamStructure,
    ChatConfigObject,
} from "@local/shared/src/execution/types/swarm.js";
import { 
    SystemEvent,
    SwarmEvent,
    RunEvent,
    ExecutionEvent,
} from "@local/shared/src/execution/types/events.js";
import { SessionUser } from "@local/server/src/types.js";
import { getEventBus } from "../../cross-cutting/events/eventBus.js";
import { SwarmStateStore } from "../state/swarmStateStore.js";
import { MetacognitiveReasoner } from "../intelligence/metacognitiveReasoner.js";
import { ResourceManager } from "../organization/resourceManager.js";
import { TeamManager } from "../organization/teamManager.js";
import { CoordinationProtocol } from "../communication/coordinationProtocol.js";

/**
 * Tier 1 Swarm State Machine
 * 
 * Coordinates intelligent swarms through metacognitive reasoning and natural language.
 * This is the strategic intelligence layer that:
 * - Forms teams of agents based on capabilities
 * - Allocates resources across the swarm
 * - Monitors progress and adapts strategies
 * - Communicates with Tier 2 for routine execution
 * - Evolves swarm behavior through experience
 */
export class TierOneSwarmStateMachine {
    private readonly swarmId: string;
    private state: SwarmState;
    private readonly eventBus: EventBus;
    private readonly stateStore: SwarmStateStore;
    private readonly metacognitiveReasoner: MetacognitiveReasoner;
    private readonly resourceManager: ResourceManager;
    private readonly teamManager: TeamManager;
    private readonly coordinationProtocol: CoordinationProtocol;
    
    // Event processing
    private processingLock = false;
    private eventQueue: SwarmEvent[] = [];
    private disposed = false;
    
    // User association
    private initiatingUser: SessionUser | null = null;
    private conversationId: string | null = null;

    constructor(
        swarmId: string,
        private readonly config: SwarmConfig,
    ) {
        this.swarmId = swarmId;
        this.eventBus = getEventBus();
        this.stateStore = new SwarmStateStore();
        this.metacognitiveReasoner = new MetacognitiveReasoner();
        this.resourceManager = new ResourceManager();
        this.teamManager = new TeamManager();
        this.coordinationProtocol = new CoordinationProtocol();
        
        // Initialize state
        this.state = this.createInitialState();
    }

    /**
     * Creates the initial swarm state
     */
    private createInitialState(): SwarmState {
        return {
            id: this.swarmId,
            phase: SwarmPhase.FORMING,
            config: this.config,
            goal: this.config.goal || "",
            teams: [],
            agents: [],
            blackboard: [],
            resources: {
                allocated: [],
                available: this.config.resources || [],
                usage: {
                    memoryMB: 0,
                    cpuPercent: 0,
                    ioOps: 0,
                    networkKB: 0,
                },
            },
            performance: {
                startTime: new Date(),
                tasksCompleted: 0,
                tasksTotal: 0,
                avgResponseTime: 0,
                errorRate: 0,
            },
            createdAt: new Date(),
            updatedAt: new Date(),
        };
    }

    /**
     * Starts the swarm with a goal
     */
    async start(
        conversationId: string, 
        goal: string, 
        initiatingUser: SessionUser,
        chatConfig?: ChatConfigObject,
    ): Promise<void> {
        if (this.state.phase !== SwarmPhase.FORMING) {
            logger.warn(`TierOneSwarmStateMachine ${this.swarmId} already started. Current phase: ${this.state.phase}`);
            return;
        }

        this.conversationId = conversationId;
        this.initiatingUser = initiatingUser;
        this.state.goal = goal;
        
        logger.info(`Starting TierOneSwarmStateMachine ${this.swarmId} for conversation ${conversationId} with goal: "${goal}"`);

        try {
            // Save initial state
            await this.stateStore.saveSwarmState(this.state);

            // Form initial teams based on goal
            await this.formInitialTeams(goal, chatConfig);

            // Transition to planning phase
            await this.transitionToPhase(SwarmPhase.PLANNING);

            // Emit swarm started event
            await this.eventBus.publish({
                type: "swarm.started",
                source: { tier: 1, component: "swarm", id: this.swarmId },
                data: {
                    swarmId: this.swarmId,
                    conversationId,
                    goal,
                    userId: initiatingUser.id,
                },
                metadata: {
                    timestamp: new Date(),
                    correlationId: uuidv4(),
                    version: "1.0",
                },
            });

            // Start metacognitive planning
            await this.startMetacognitivePlanning();

        } catch (error) {
            logger.error(`Failed to start swarm ${this.swarmId}`, error);
            await this.transitionToPhase(SwarmPhase.FAILED);
            throw error;
        }
    }

    /**
     * Forms initial teams based on the goal and available agents
     */
    private async formInitialTeams(goal: string, chatConfig?: ChatConfigObject): Promise<void> {
        // Use team manager to analyze goal and form teams
        const teams = await this.teamManager.formTeamsForGoal(
            goal,
            this.config,
            chatConfig,
        );

        // Update state with teams
        this.state.teams = teams;

        // Create agents for each team
        for (const team of teams) {
            const agents = await this.teamManager.createAgentsForTeam(team);
            this.state.agents.push(...agents);
        }

        // Save updated state
        await this.stateStore.saveSwarmState(this.state);
    }

    /**
     * Starts metacognitive planning process
     */
    private async startMetacognitivePlanning(): Promise<void> {
        // Use metacognitive reasoner to create initial plan
        const plan = await this.metacognitiveReasoner.createExecutionPlan(
            this.state.goal,
            this.state.teams,
            this.state.resources.available,
        );

        // Store plan in blackboard
        const planItem: BlackboardItem = {
            id: uuidv4(),
            type: "plan",
            content: plan,
            author: "metacognitive_reasoner",
            timestamp: new Date(),
            visibility: "swarm",
        };
        
        this.state.blackboard.push(planItem);
        await this.stateStore.saveBlackboardItem(this.swarmId, planItem);

        // Request routine execution from Tier 2
        await this.requestRoutineExecution(plan);
    }

    /**
     * Requests routine execution from Tier 2
     */
    private async requestRoutineExecution(plan: any): Promise<void> {
        await this.eventBus.publish({
            type: "routine.create",
            source: { tier: 1, component: "swarm", id: this.swarmId },
            target: { tier: 2 },
            data: {
                swarmId: this.swarmId,
                plan,
                userId: this.initiatingUser?.id,
            },
            metadata: {
                timestamp: new Date(),
                correlationId: uuidv4(),
                version: "1.0",
            },
        });
    }

    /**
     * Handles incoming events
     */
    async handleEvent(event: SwarmEvent): Promise<void> {
        if (this.disposed) return;

        this.eventQueue.push(event);
        
        if (!this.processingLock) {
            await this.processEventQueue();
        }
    }

    /**
     * Processes queued events
     */
    private async processEventQueue(): Promise<void> {
        if (this.processingLock || this.disposed) return;
        
        this.processingLock = true;

        try {
            while (this.eventQueue.length > 0) {
                const event = this.eventQueue.shift()!;
                await this.processEvent(event);
            }
        } finally {
            this.processingLock = false;
        }
    }

    /**
     * Processes a single event
     */
    private async processEvent(event: SwarmEvent): Promise<void> {
        logger.info(`Processing event ${event.type} for swarm ${this.swarmId}`);

        try {
            switch (event.type) {
                case "routine.progress":
                    await this.handleRoutineProgress(event);
                    break;
                case "routine.completed":
                    await this.handleRoutineCompleted(event);
                    break;
                case "routine.failed":
                    await this.handleRoutineFailed(event);
                    break;
                case "agent.message":
                    await this.handleAgentMessage(event);
                    break;
                case "resource.request":
                    await this.handleResourceRequest(event);
                    break;
                default:
                    logger.warn(`Unknown event type: ${event.type}`);
            }

            // Update state after processing
            await this.stateStore.saveSwarmState(this.state);

        } catch (error) {
            logger.error(`Error processing event ${event.type} for swarm ${this.swarmId}`, error);
        }
    }

    /**
     * Handles routine progress updates from Tier 2
     */
    private async handleRoutineProgress(event: RunEvent): Promise<void> {
        // Update performance metrics
        if (event.data.progress) {
            this.state.performance.tasksCompleted = event.data.progress.completed || 0;
            this.state.performance.tasksTotal = event.data.progress.total || 0;
        }

        // Metacognitive monitoring
        const shouldAdapt = await this.metacognitiveReasoner.evaluateProgress(
            this.state,
            event.data.progress,
        );

        if (shouldAdapt) {
            await this.adaptStrategy();
        }
    }

    /**
     * Handles routine completion
     */
    private async handleRoutineCompleted(event: RunEvent): Promise<void> {
        logger.info(`Routine completed for swarm ${this.swarmId}`);

        // Evaluate results
        const evaluation = await this.metacognitiveReasoner.evaluateResults(
            this.state,
            event.data.results,
        );

        // Update blackboard with results
        const resultItem: BlackboardItem = {
            id: uuidv4(),
            type: "result",
            content: evaluation,
            author: "metacognitive_reasoner",
            timestamp: new Date(),
            visibility: "swarm",
        };
        
        this.state.blackboard.push(resultItem);
        await this.stateStore.saveBlackboardItem(this.swarmId, resultItem);

        // Check if goal is achieved
        if (evaluation.goalAchieved) {
            await this.transitionToPhase(SwarmPhase.DISSOLVING);
        } else {
            // Plan next steps
            await this.planNextSteps(evaluation);
        }
    }

    /**
     * Handles routine failure
     */
    private async handleRoutineFailed(event: RunEvent): Promise<void> {
        logger.error(`Routine failed for swarm ${this.swarmId}`, event.data.error);

        // Analyze failure
        const analysis = await this.metacognitiveReasoner.analyzeFailure(
            this.state,
            event.data.error,
        );

        // Determine recovery strategy
        if (analysis.recoverable) {
            await this.recoverFromFailure(analysis);
        } else {
            await this.transitionToPhase(SwarmPhase.FAILED);
        }
    }

    /**
     * Handles messages from agents
     */
    private async handleAgentMessage(event: SwarmEvent): Promise<void> {
        // Route message through coordination protocol
        await this.coordinationProtocol.routeMessage(
            event.data.agentId,
            event.data.message,
            this.state,
        );
    }

    /**
     * Handles resource requests
     */
    private async handleResourceRequest(event: SwarmEvent): Promise<void> {
        const allocation = await this.resourceManager.allocateResources(
            event.data.agentId,
            event.data.requested,
            this.state.resources,
        );

        if (allocation) {
            // Update state with new allocation
            this.state.resources.allocated.push(allocation);
            
            // Notify agent of allocation
            await this.eventBus.publish({
                type: "resource.allocated",
                source: { tier: 1, component: "swarm", id: this.swarmId },
                target: { tier: 1, component: "agent", id: event.data.agentId },
                data: { allocation },
                metadata: {
                    timestamp: new Date(),
                    correlationId: event.metadata?.correlationId || uuidv4(),
                    version: "1.0",
                },
            });
        }
    }

    /**
     * Adapts strategy based on metacognitive evaluation
     */
    private async adaptStrategy(): Promise<void> {
        logger.info(`Adapting strategy for swarm ${this.swarmId}`);

        // Get adaptation recommendations
        const adaptations = await this.metacognitiveReasoner.recommendAdaptations(this.state);

        // Apply adaptations
        for (const adaptation of adaptations) {
            switch (adaptation.type) {
                case "reorganize_teams":
                    await this.reorganizeTeams(adaptation.params);
                    break;
                case "reallocate_resources":
                    await this.reallocateResources(adaptation.params);
                    break;
                case "change_approach":
                    await this.changeApproach(adaptation.params);
                    break;
            }
        }
    }

    /**
     * Plans next steps after routine completion
     */
    private async planNextSteps(evaluation: any): Promise<void> {
        const nextPlan = await this.metacognitiveReasoner.planNextSteps(
            this.state,
            evaluation,
        );

        // Store updated plan
        const planItem: BlackboardItem = {
            id: uuidv4(),
            type: "plan",
            content: nextPlan,
            author: "metacognitive_reasoner",
            timestamp: new Date(),
            visibility: "swarm",
        };
        
        this.state.blackboard.push(planItem);
        await this.stateStore.saveBlackboardItem(this.swarmId, planItem);

        // Request execution of next phase
        await this.requestRoutineExecution(nextPlan);
    }

    /**
     * Recovers from failure
     */
    private async recoverFromFailure(analysis: any): Promise<void> {
        logger.info(`Attempting recovery for swarm ${this.swarmId}`);

        // Apply recovery strategy
        const recoveryPlan = await this.metacognitiveReasoner.createRecoveryPlan(
            this.state,
            analysis,
        );

        // Execute recovery
        await this.requestRoutineExecution(recoveryPlan);
    }

    /**
     * Reorganizes teams based on adaptation parameters
     */
    private async reorganizeTeams(params: any): Promise<void> {
        const newTeams = await this.teamManager.reorganizeTeams(
            this.state.teams,
            params,
        );

        this.state.teams = newTeams;
    }

    /**
     * Reallocates resources based on adaptation parameters
     */
    private async reallocateResources(params: any): Promise<void> {
        const newAllocations = await this.resourceManager.reallocateResources(
            this.state.resources,
            params,
        );

        this.state.resources.allocated = newAllocations;
    }

    /**
     * Changes approach based on adaptation parameters
     */
    private async changeApproach(params: any): Promise<void> {
        // Update swarm configuration
        this.state.config = {
            ...this.state.config,
            ...params.configUpdates,
        };

        // Notify all agents of approach change
        await this.coordinationProtocol.broadcastToSwarm(
            "approach_changed",
            params,
            this.state,
        );
    }

    /**
     * Transitions to a new phase
     */
    private async transitionToPhase(newPhase: SwarmPhase): Promise<void> {
        const oldPhase = this.state.phase;
        this.state.phase = newPhase;
        this.state.updatedAt = new Date();

        logger.info(`Swarm ${this.swarmId} transitioning from ${oldPhase} to ${newPhase}`);

        // Emit phase transition event
        await this.eventBus.publish({
            type: "swarm.phase_changed",
            source: { tier: 1, component: "swarm", id: this.swarmId },
            data: {
                swarmId: this.swarmId,
                oldPhase,
                newPhase,
            },
            metadata: {
                timestamp: new Date(),
                correlationId: uuidv4(),
                version: "1.0",
            },
        });

        // Handle phase-specific actions
        switch (newPhase) {
            case SwarmPhase.EXECUTING:
                await this.onExecutionPhase();
                break;
            case SwarmPhase.DISSOLVING:
                await this.onDissolvingPhase();
                break;
            case SwarmPhase.FAILED:
                await this.onFailedPhase();
                break;
        }

        await this.stateStore.saveSwarmState(this.state);
    }

    /**
     * Handles execution phase entry
     */
    private async onExecutionPhase(): Promise<void> {
        // Start monitoring
        await this.metacognitiveReasoner.startMonitoring(this.state);
    }

    /**
     * Handles dissolving phase entry
     */
    private async onDissolvingPhase(): Promise<void> {
        // Release all resources
        await this.resourceManager.releaseAllResources(this.state.resources);

        // Archive results
        await this.stateStore.archiveSwarm(this.swarmId);

        // Notify completion
        await this.eventBus.publish({
            type: "swarm.completed",
            source: { tier: 1, component: "swarm", id: this.swarmId },
            data: {
                swarmId: this.swarmId,
                results: this.state.blackboard.filter(item => item.type === "result"),
                performance: this.state.performance,
            },
            metadata: {
                timestamp: new Date(),
                correlationId: uuidv4(),
                version: "1.0",
            },
        });
    }

    /**
     * Handles failed phase entry
     */
    private async onFailedPhase(): Promise<void> {
        // Release resources
        await this.resourceManager.releaseAllResources(this.state.resources);

        // Notify failure
        await this.eventBus.publish({
            type: "swarm.failed",
            source: { tier: 1, component: "swarm", id: this.swarmId },
            data: {
                swarmId: this.swarmId,
                error: "Swarm execution failed",
                lastState: this.state,
            },
            metadata: {
                timestamp: new Date(),
                correlationId: uuidv4(),
                version: "1.0",
            },
        });
    }

    /**
     * Pauses the swarm
     */
    async pause(): Promise<void> {
        if (this.state.phase === SwarmPhase.EXECUTING) {
            await this.transitionToPhase(SwarmPhase.PLANNING);
            logger.info(`Swarm ${this.swarmId} paused`);
        }
    }

    /**
     * Resumes the swarm
     */
    async resume(): Promise<void> {
        if (this.state.phase === SwarmPhase.PLANNING) {
            await this.transitionToPhase(SwarmPhase.EXECUTING);
            logger.info(`Swarm ${this.swarmId} resumed`);
        }
    }

    /**
     * Stops the swarm
     */
    async stop(reason: string): Promise<void> {
        logger.info(`Stopping swarm ${this.swarmId}. Reason: ${reason}`);
        
        // Save stop reason
        const stopItem: BlackboardItem = {
            id: uuidv4(),
            type: "control",
            content: { action: "stop", reason },
            author: "system",
            timestamp: new Date(),
            visibility: "swarm",
        };
        
        this.state.blackboard.push(stopItem);
        await this.stateStore.saveBlackboardItem(this.swarmId, stopItem);

        // Transition to dissolving
        await this.transitionToPhase(SwarmPhase.DISSOLVING);
    }

    /**
     * Shuts down the swarm state machine
     */
    async shutdown(): Promise<void> {
        this.disposed = true;
        this.eventQueue = [];
        
        // Final state save
        await this.stateStore.saveSwarmState(this.state);
        
        logger.info(`Swarm ${this.swarmId} shutdown complete`);
    }

    /**
     * Gets the current swarm state
     */
    getState(): SwarmState {
        return { ...this.state };
    }

    /**
     * Gets the swarm ID
     */
    getSwarmId(): string {
        return this.swarmId;
    }

    /**
     * Gets the associated user ID
     */
    getUserId(): string | undefined {
        return this.initiatingUser?.id;
    }

    /**
     * Gets the conversation ID
     */
    getConversationId(): string | null {
        return this.conversationId;
    }
}