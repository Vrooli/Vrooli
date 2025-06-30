import {
    RunStatus,
    SwarmStatus,
    nanoid,
    type RoutineExecutionInput,
    type SwarmCoordinationInput,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { type Logger } from "winston";
import { DbProvider } from "../../db/provider.js";
import { CacheService } from "../../redisConn.js";
import { getUnifiedEventSystem } from "../events/index.js";
import type { IEventBus } from "../events/types.js";
import { AuthIntegrationService } from "./integration/authIntegrationService.js";
import { RoutineStorageService } from "./integration/routineStorageService.js";
import { RunPersistenceService } from "./integration/runPersistenceService.js";
import { SwarmContextManager } from "./shared/SwarmContextManager.js";
import { SwarmCoordinator } from "./tier1/coordination/index.js";
import { createConversationBridge } from "./tier1/intelligence/conversationBridge.js";
import { TierTwoOrchestrator } from "./tier2/index.js";
import { TierThreeExecutor } from "./tier3/index.js";

/**
 * Main service entry point for the three-tier execution architecture
 * 
 * This service coordinates all three tiers and provides a unified interface
 * for starting and managing swarms and runs.
 */
export class SwarmExecutionService {
    private readonly logger: Logger;
    private readonly eventBus: IEventBus | null;
    private readonly tierOne: SwarmCoordinator;
    private readonly tierTwo: TierTwoOrchestrator;
    private readonly tierThree: TierThreeExecutor;
    private readonly persistenceService: RunPersistenceService;
    private readonly routineService: RoutineStorageService;
    private readonly authService: AuthIntegrationService;
    private readonly contextManager: SwarmContextManager;
    private readonly conversationBridge: ReturnType<typeof createConversationBridge>;

    constructor(logger: Logger) {
        this.logger = logger;

        // Get unified event bus for cross-tier communication
        this.eventBus = getUnifiedEventSystem();

        // Get PrismaClient from DbProvider
        const prisma = DbProvider.get();

        // Initialize integration services with proper dependencies
        this.persistenceService = new RunPersistenceService(prisma, logger);
        this.routineService = new RoutineStorageService(prisma, logger);
        this.authService = new AuthIntegrationService(prisma, logger, this.eventBus);

        // Initialize SwarmContextManager for modern state management
        this.contextManager = new SwarmContextManager(
            CacheService.get().raw() as any, // Redis client
            logger,
        );

        // Initialize conversation bridge for AI coordination
        this.conversationBridge = createConversationBridge(logger);

        // Initialize tiers in dependency order (tier 3 -> tier 2 -> tier 1)
        this.tierThree = new TierThreeExecutor(logger, this.eventBus);
        this.tierTwo = new TierTwoOrchestrator(
            logger,
            this.eventBus,
            this.tierThree,
            this.contextManager, // Pass SwarmContextManager to Tier 2
        );
        this.tierOne = new SwarmCoordinator(
            logger,
            this.contextManager,
            this.conversationBridge,
            this.tierTwo,
        );

        // Start all services
        this.initialize();
    }

    /**
     * Initialize all services and set up event handlers
     */
    private initialize(): void {
        // Subscribe to tier events
        this.setupEventHandlers();

        this.logger.info("[SwarmExecutionService] Initialized three-tier execution architecture");
    }

    /**
     * Start a new swarm execution
     */
    async startSwarm(config: {
        swarmId: string;
        name: string;
        description: string;
        goal: string;
        resources: {
            maxCredits: number;
            maxTokens: number;
            maxTime: number;
            tools: Array<{ name: string; description: string }>;
        };
        config: {
            model: string;
            temperature: number;
            autoApproveTools: boolean;
            parallelExecutionLimit: number;
        };
        userId: string;
        organizationId?: string;
        parentSwarmId?: string; // NEW: For child swarms
        leaderBotId?: string; // Optional bot ID, defaults to Valyxa
    }): Promise<{ swarmId: string }> {
        this.logger.info("[SwarmExecutionService] Starting new swarm", {
            swarmId: config.swarmId,
            name: config.name,
            userId: config.userId,
        });

        try {
            // Check user permissions - for now, just verify user exists and has basic permissions
            const userData = await this.authService.getUserData(config.userId);
            if (!userData) {
                throw new Error("User not found");
            }

            if (!userData.permissions.canCreateSwarms) {
                throw new Error("User does not have permission to create swarms");
            }

            // Create SwarmCoordinationInput
            const swarmInput: SwarmCoordinationInput = {
                goal: config.goal,
                availableAgents: config.resources.tools.map((tool, index) => ({
                    id: `agent-${index}`,
                    name: tool.name,
                    capabilities: [tool.description],
                    currentLoad: 0,
                    maxConcurrentTasks: config.config.parallelExecutionLimit,
                })),
                teamConfiguration: {
                    preferredTeamSize: Math.min(config.resources.tools.length, 5),
                    requiredSkills: config.resources.tools.map(t => t.name),
                    collaborationStyle: "collaborative",
                },
            };

            // Create execution request for Tier 1
            const request: TierExecutionRequest<SwarmCoordinationInput> = {
                context: {
                    executionId: config.swarmId,
                    swarmId: config.parentSwarmId || config.swarmId,
                    userId: config.userId,
                    timestamp: new Date(),
                    correlationId: nanoid(),
                    organizationId: config.organizationId,
                },
                input: swarmInput,
                allocation: {
                    maxCredits: config.resources.maxCredits.toString(),
                    maxDurationMs: config.resources.maxTime,
                    maxMemoryMB: 1024,
                    maxConcurrentSteps: config.config.parallelExecutionLimit,
                },
                options: {
                    priority: "medium",
                    timeout: config.resources.maxTime,
                    emergentCapabilities: true,
                },
            };

            // Execute through Tier 1
            const result = await this.tierOne.execute(request);

            if (result.status !== "completed") {
                throw new Error(result.error?.message || "Failed to start swarm");
            }

            return { swarmId: config.swarmId };

        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to start swarm", {
                swarmId: config.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Start a new run within a swarm
     */
    async startRun(config: {
        runId: string;
        swarmId: string;
        routineVersionId: string;
        inputs: Record<string, unknown>;
        config: {
            strategy?: string;
            model: string;
            maxSteps: number;
            timeout: number;
        };
        userId: string;
    }): Promise<{ runId: string }> {
        this.logger.info("[SwarmExecutionService] Starting new run", {
            runId: config.runId,
            swarmId: config.swarmId,
            routineVersionId: config.routineVersionId,
        });

        try {
            // Load routine from database
            const routine = await this.routineService.loadRoutine(config.routineVersionId);
            if (!routine) {
                throw new Error(`Routine version not found: ${config.routineVersionId}`);
            }

            // Check user permissions
            const permissionResult = await this.authService.canUserExecuteRoutine(
                config.userId,
                config.routineVersionId,
            );

            if (!permissionResult.allowed) {
                throw new Error(`User does not have permission to run this routine: ${permissionResult.reason}`);
            }

            // Create run record in database
            await this.persistenceService.createRun({
                id: config.runId,
                routineId: config.routineVersionId,
                userId: config.userId,
                inputs: config.inputs,
                metadata: {
                    swarmId: config.swarmId,
                    executionArchitecture: "three-tier",
                    routineName: routine.name || `Run of ${routine.root.name}`,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            });

            // Create RoutineExecutionInput
            const routineInput: RoutineExecutionInput = {
                routineId: config.routineVersionId,
                parameters: config.inputs,
                workflow: {
                    steps: [], // Will be loaded by Tier 2
                    dependencies: [],
                },
            };

            // Get swarm context to determine resource allocation
            const swarmContext = await this.contextManager.getContext(config.swarmId);
            if (!swarmContext) {
                throw new Error(`Swarm ${config.swarmId} not found`);
            }

            // Create execution request for routine
            const request: TierExecutionRequest<RoutineExecutionInput> = {
                context: {
                    executionId: config.runId,
                    swarmId: config.swarmId,
                    userId: config.userId,
                    timestamp: new Date(),
                    correlationId: nanoid(),
                    routineId: config.routineVersionId,
                },
                input: routineInput,
                allocation: {
                    maxCredits: swarmContext.resources.available.credits,
                    maxDurationMs: config.config.timeout,
                    maxMemoryMB: 512,
                    maxConcurrentSteps: config.config.maxSteps,
                },
                options: {
                    priority: "medium",
                    timeout: config.config.timeout,
                    strategy: config.config.strategy,
                },
            };

            // Execute through Tier 1 (which will delegate to Tier 2)
            const result = await this.tierOne.execute(request);

            if (result.status !== "completed") {
                throw new Error(result.error?.message || "Failed to execute run");
            }

            return { runId: config.runId };

        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to start run", {
                runId: config.runId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Update run status to failed
            await this.persistenceService.updateRunStatus(config.runId, RunStatus.Failed);

            throw error;
        }
    }

    /**
     * Get swarm status
     */
    async getSwarmStatus(swarmId: string): Promise<{
        status: SwarmStatus;
        progress?: number;
        currentPhase?: string;
        activeRuns?: number;
        completedRuns?: number;
        errors?: string[];
    }> {
        try {
            const executionStatus = await this.tierOne.getExecutionStatus(swarmId);

            // Map ExecutionStatus to SwarmStatus
            const statusMap: Record<string, SwarmStatus> = {
                "pending": SwarmStatus.Pending,
                "running": SwarmStatus.Running,
                "paused": SwarmStatus.Paused,
                "completed": SwarmStatus.Completed,
                "failed": SwarmStatus.Failed,
                "cancelled": SwarmStatus.Cancelled,
            };

            return {
                status: statusMap[executionStatus.status] || SwarmStatus.Unknown,
                progress: executionStatus.progress,
                currentPhase: executionStatus.metadata?.currentPhase as string,
                activeRuns: executionStatus.metadata?.activeRuns as number,
                completedRuns: executionStatus.metadata?.completedRuns as number,
                errors: executionStatus.error ? [executionStatus.error.message] : undefined,
            };
        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to get swarm status", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                status: SwarmStatus.Unknown,
                errors: [error instanceof Error ? error.message : "Unknown error"],
            };
        }
    }

    /**
     * Get run status
     */
    async getRunStatus(runId: string): Promise<{
        status: RunStatus;
        progress?: number;
        currentStep?: string;
        outputs?: Record<string, unknown>;
        errors?: string[];
    }> {
        try {
            // Get run from database
            const run = await this.persistenceService.loadRun(runId);
            if (!run) {
                return {
                    status: RunStatus.Unknown,
                    errors: ["Run not found"],
                };
            }

            // Get detailed status from Tier 2
            const detailedStatus = await this.tierTwo.getRunStatus(runId);

            // Map database status to RunStatus enum
            const statusMap: Record<string, RunStatus> = {
                "Scheduled": RunStatus.Scheduled,
                "InProgress": RunStatus.InProgress,
                "Completed": RunStatus.Completed,
                "Failed": RunStatus.Failed,
                "Cancelled": RunStatus.Cancelled,
                "Paused": RunStatus.Paused,
            };

            return {
                status: statusMap[run.status] || RunStatus.InProgress,
                progress: detailedStatus?.progress,
                currentStep: detailedStatus?.currentStep,
                outputs: run.outputs,
                errors: detailedStatus?.errors,
            };

        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to get run status", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                status: RunStatus.Unknown,
                errors: [error instanceof Error ? error.message : "Unknown error"],
            };
        }
    }

    /**
     * Cancel a swarm
     */
    async cancelSwarm(swarmId: string, userId: string, reason?: string): Promise<{
        success: boolean;
        message?: string;
    }> {
        try {
            await this.tierOne.cancelExecution(swarmId);
            return { success: true, message: "Swarm cancelled successfully" };
        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to cancel swarm", {
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                success: false,
                message: error instanceof Error ? error.message : "Failed to cancel swarm",
            };
        }
    }

    /**
     * Cancel a run
     */
    async cancelRun(runId: string, userId: string, reason?: string): Promise<{
        success: boolean;
        message?: string;
    }> {
        try {
            // Update run status
            await this.persistenceService.updateRunState(runId, "CANCELLED");

            // Cancel through Tier 2
            await this.tierTwo.cancelRun(runId, reason);

            return { success: true, message: "Run cancelled successfully" };
        } catch (error) {
            this.logger.error("[SwarmExecutionService] Failed to cancel run", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            return {
                success: false,
                message: error instanceof Error ? error.message : "Failed to cancel run",
            };
        }
    }

    /**
     * Setup event handlers for cross-tier communication
     */
    private setupEventHandlers(): void {
        // Handle run completion events
        this.eventBus.on("run.completed", async (event) => {
            const { runId, outputs } = event.data;

            // Update run status AND outputs
            await this.persistenceService.updateRunState(runId, "COMPLETED");
            await this.persistenceService.updateRunOutputs(runId, outputs);
        });

        // Handle run failure events
        this.eventBus.on("run.failed", async (event) => {
            const { runId, error } = event.data;
            await this.persistenceService.updateRunState(runId, "FAILED");
        });

        // Handle step completion events
        this.eventBus.on("step.completed", async (event) => {
            const { runId, stepId, outputs, resourceUsage } = event.data;
            await this.persistenceService.recordStepExecution(runId, {
                stepId,
                status: "completed",
                result: outputs,
                resourceUsage: resourceUsage || { tokens: 0, credits: 0, duration: 0 },
                startedAt: new Date(),
                completedAt: new Date(),
            });
        });
    }

    /**
     * Shutdown the service
     */
    async shutdown(): Promise<void> {
        this.logger.info("[SwarmExecutionService] Shutting down three-tier execution architecture");

        // Shutdown all tiers
        await Promise.all([
            this.tierOne.shutdown(),
            this.tierTwo.shutdown(),
            this.tierThree.shutdown(),
        ]);
    }
}
