import { type Logger } from "winston";
import {
    SwarmStatus,
    RunStatus,
    nanoid,
} from "@vrooli/shared";
import { TierOneCoordinator } from "./tier1/index.js";
import { TierTwoOrchestrator } from "./tier2/index.js";
import { TierThreeExecutor } from "./tier3/index.js";
import { EventBus } from "./cross-cutting/events/eventBus.js";
import { RunPersistenceService } from "./integration/runPersistenceService.js";
import { RoutineStorageService } from "./integration/routineStorageService.js";
import { AuthIntegrationService } from "./integration/authIntegrationService.js";
import { DbProvider } from "../../db/provider.js";

/**
 * Main service entry point for the three-tier execution architecture
 * 
 * This service coordinates all three tiers and provides a unified interface
 * for starting and managing swarms and runs.
 */
export class SwarmExecutionService {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly tierOne: TierOneCoordinator;
    private readonly tierTwo: TierTwoOrchestrator;
    private readonly tierThree: TierThreeExecutor;
    private readonly persistenceService: RunPersistenceService;
    private readonly routineService: RoutineStorageService;
    private readonly authService: AuthIntegrationService;

    constructor(logger: Logger) {
        this.logger = logger;
        
        // Initialize event bus for cross-tier communication
        this.eventBus = new EventBus(logger);
        
        // Get PrismaClient from DbProvider
        const prisma = DbProvider.get();
        
        // Initialize integration services with proper dependencies
        this.persistenceService = new RunPersistenceService(prisma, logger);
        this.routineService = new RoutineStorageService(prisma, logger);
        this.authService = new AuthIntegrationService(prisma, logger);
        
        // Initialize tiers in dependency order (tier 3 -> tier 2 -> tier 1)
        this.tierThree = new TierThreeExecutor(logger, this.eventBus);
        this.tierTwo = new TierTwoOrchestrator(logger, this.eventBus, this.tierThree);
        this.tierOne = new TierOneCoordinator(logger, this.eventBus, this.tierTwo);
        
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

            // Start swarm through Tier 1
            await this.tierOne.startSwarm({
                swarmId: config.swarmId,
                name: config.name,
                description: config.description,
                goal: config.goal,
                resources: config.resources,
                config: config.config,
                userId: config.userId,
                organizationId: config.organizationId,
                parentSwarmId: config.parentSwarmId, // NEW: Pass through parent relationship
                leaderBotId: config.leaderBotId, // Pass through leader bot ID
            });

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

            // Start run through Tier 1 (which will delegate to Tier 2)
            await this.tierOne.requestRunExecution({
                swarmId: config.swarmId,
                runId: config.runId,
                routineVersionId: config.routineVersionId,
                inputs: config.inputs,
                config: config.config,
            });

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
            return await this.tierOne.getSwarmStatus(swarmId);
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
                outputs: run.metadata, // Using metadata for outputs since loadRun doesn't return outputs
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
            await this.tierOne.cancelSwarm(swarmId, userId, reason);
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
            await this.persistenceService.updateRunState(runId, "COMPLETED");
            // TODO: Store outputs in run data - would need to extend persistence service
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
