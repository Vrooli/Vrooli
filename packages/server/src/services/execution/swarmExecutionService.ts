import { type Logger } from "winston";
import {
    type SwarmStatus,
    type RunStatus,
    nanoid,
} from "@vrooli/shared";
import { TierOneCoordinator } from "./tier1/index.js";
import { TierTwoOrchestrator } from "./tier2/index.js";
import { TierThreeExecutor } from "./tier3/index.js";
import { EventBus } from "./cross-cutting/events/eventBus.js";
import { RunPersistenceService } from "./integration/runPersistenceService.js";
import { RoutineStorageService } from "./integration/routineStorageService.js";
import { AuthIntegrationService } from "./integration/authIntegrationService.js";

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
        
        // Initialize integration services
        this.persistenceService = new RunPersistenceService(logger);
        this.routineService = new RoutineStorageService(logger);
        this.authService = new AuthIntegrationService(logger);
        
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
    }): Promise<{ swarmId: string }> {
        this.logger.info("[SwarmExecutionService] Starting new swarm", {
            swarmId: config.swarmId,
            name: config.name,
            userId: config.userId,
        });

        try {
            // Check user permissions
            const hasPermission = await this.authService.checkPermission(
                config.userId,
                "swarm:create",
                config.organizationId,
            );

            if (!hasPermission) {
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
            const hasPermission = await this.authService.checkPermission(
                config.userId,
                "routine:run",
                routine.root.ownerId,
            );

            if (!hasPermission) {
                throw new Error("User does not have permission to run this routine");
            }

            // Create run record in database
            const run = await this.persistenceService.createRun({
                id: config.runId,
                routineVersionId: config.routineVersionId,
                userId: config.userId,
                name: routine.name || `Run of ${routine.root.name}`,
                status: RunStatus.InProgress,
                inputs: config.inputs,
                completedComplexity: 0,
                contextSwitches: 0,
                timeElapsed: "0",
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
            const run = await this.persistenceService.getRun(runId);
            if (!run) {
                return {
                    status: RunStatus.Unknown,
                    errors: ["Run not found"],
                };
            }

            // Get detailed status from Tier 2
            const detailedStatus = await this.tierTwo.getRunStatus(runId);

            return {
                status: run.status,
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
            await this.persistenceService.updateRunStatus(runId, RunStatus.Cancelled);
            
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
            await this.persistenceService.completeRun(runId, outputs);
        });

        // Handle run failure events
        this.eventBus.on("run.failed", async (event) => {
            const { runId, error } = event.data;
            await this.persistenceService.updateRunStatus(runId, RunStatus.Failed);
        });

        // Handle step completion events
        this.eventBus.on("step.completed", async (event) => {
            const { runId, stepId, outputs } = event.data;
            await this.persistenceService.updateRunProgress(runId, {
                lastCompletedStep: stepId,
                outputs,
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