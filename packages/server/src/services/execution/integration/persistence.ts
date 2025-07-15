import {
    type ExecutionResourceUsage,
    type RunState,
    type StepStatus,
    RunStepStatus,
    generatePK,
} from "@vrooli/shared";
import { DbProvider } from "../../../db/provider.js";
import { logger } from "../../../events/logger.js";

/**
 * Execution status mapping between shared types and database enums
 */
const RUN_STATUS_MAP = {
    "UNINITIALIZED": "Scheduled",
    "LOADING": "InProgress",
    "READY": "InProgress",
    "RUNNING": "InProgress",
    "PAUSED": "InProgress",
    "SUSPENDED": "InProgress",
    "COMPLETED": "Completed",
    "FAILED": "Failed",
    "CANCELLED": "Cancelled",
} as const;

const STEP_STATUS_MAP: Record<StepStatus["state"], RunStepStatus> = {
    "pending": RunStepStatus.InProgress,
    "ready": RunStepStatus.InProgress,
    "running": RunStepStatus.InProgress,
    "completed": RunStepStatus.Completed,
    "failed": RunStepStatus.Skipped,  // Map failed to Skipped since Failed doesn't exist in RunStepStatus
    "skipped": RunStepStatus.Skipped,
};

/**
 * Run persistence service that stores execution state in Vrooli's database
 * 
 * This service manages the persistence of run data, integrating the execution
 * architecture with Vrooli's existing run tracking system.
 */
export class RunPersistenceService {

    // eslint-disable-next-line no-magic-numbers
    private static readonly NAME_SUFFIX_LENGTH = 8;
    // eslint-disable-next-line no-magic-numbers
    private static readonly DEFAULT_HISTORY_LIMIT = 20;
    // eslint-disable-next-line no-magic-numbers
    private static readonly HIGH_TOOL_CALLS_THRESHOLD = 5;
    // eslint-disable-next-line no-magic-numbers
    private static readonly LONG_DURATION_THRESHOLD_MS = 30000;
    // eslint-disable-next-line no-magic-numbers
    private static readonly HIGH_MEMORY_THRESHOLD_MB = 100;
    // eslint-disable-next-line no-magic-numbers
    private static readonly MAX_COMPLEXITY = 10;

    constructor() {
        // Add constructor logic here if needed
    }

    /**
     * Creates a new run record in the database
     */
    async createRun(runData: {
        id: string;
        routineId: string;
        userId: string;
        inputs: Record<string, unknown>;
        metadata?: Record<string, unknown>;
        createdAt: Date;
        updatedAt: Date;
    }): Promise<void> {
        logger.debug("[RunPersistenceService] Creating run", {
            runId: runData.id,
            routineId: runData.routineId,
            userId: runData.userId,
        });

        try {
            // Find the resource version
            const resourceVersion = await DbProvider.get().resource_version.findFirst({
                where: {
                    OR: [
                        { publicId: runData.routineId },
                        { root: { publicId: runData.routineId } },
                    ],
                },
                orderBy: {
                    versionIndex: "desc",
                },
            });

            // Create the run record
            await DbProvider.get().run.create({
                data: {
                    id: BigInt(runData.id),
                    name: `Execution ${runData.id.slice(-RunPersistenceService.NAME_SUFFIX_LENGTH)}`,
                    data: JSON.stringify({
                        inputs: runData.inputs,
                        metadata: runData.metadata || {},
                        executionArchitecture: "three-tier",
                    }),
                    status: "Scheduled",
                    isPrivate: false,
                    wasRunAutomatically: true,
                    resourceVersionId: resourceVersion?.id,
                    userId: BigInt(runData.userId),
                    createdAt: runData.createdAt,
                    updatedAt: runData.updatedAt,
                },
            });

            // Store inputs as run_io records
            for (const [key, value] of Object.entries(runData.inputs)) {
                await DbProvider.get().run_io.create({
                    data: {
                        id: BigInt(generatePK()),
                        runId: BigInt(runData.id),
                        nodeInputName: key,
                        nodeName: "root",
                        data: JSON.stringify(value),
                        createdAt: new Date(),
                        updatedAt: new Date(),
                    },
                });
            }

            logger.info("[RunPersistenceService] Run created", {
                runId: runData.id,
                resourceVersionId: resourceVersion?.id,
            });

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to create run", {
                runId: runData.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Updates run state in the database
     */
    async updateRunState(runId: string, state: RunState): Promise<void> {
        logger.debug("[RunPersistenceService] Updating run state", {
            runId,
            state,
        });

        try {
            const dbStatus = RUN_STATUS_MAP[state] || "InProgress";
            const updateData: any = {
                status: dbStatus,
                updatedAt: new Date(),
            };

            // Set timestamps based on state
            if (state === "RUNNING" || state === "READY") {
                updateData.startedAt = new Date();
            } else if (state === "COMPLETED" || state === "FAILED" || state === "CANCELLED") {
                updateData.completedAt = new Date();
            }

            await DbProvider.get().run.update({
                where: {
                    id: BigInt(runId),
                },
                data: updateData,
            });

            logger.debug("[RunPersistenceService] Run state updated", {
                runId,
                state,
                dbStatus,
            });

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to update run state", {
                runId,
                state,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Records step execution in the database
     */
    async recordStepExecution(runId: string, stepData: {
        stepId: string;
        state: StepStatus["state"];
        startedAt: Date;
        completedAt?: Date;
        result?: Record<string, unknown>;
        error?: string;
        resourceUsage?: ExecutionResourceUsage;
    }): Promise<void> {
        logger.debug("[RunPersistenceService] Recording step execution", {
            runId,
            stepId: stepData.stepId,
            state: stepData.state,
        });

        try {
            const dbStatus = STEP_STATUS_MAP[stepData.state] || RunStepStatus.InProgress;

            // Calculate time elapsed if completed
            let timeElapsed: number | undefined;
            if (stepData.completedAt) {
                timeElapsed = stepData.completedAt.getTime() - stepData.startedAt.getTime();
            }

            // Try to find existing step record
            const existingStep = await DbProvider.get().run_step.findFirst({
                where: {
                    runId: BigInt(runId),
                    nodeId: stepData.stepId,
                },
            });

            if (existingStep) {
                // Update existing step
                await DbProvider.get().run_step.update({
                    where: {
                        id: existingStep.id,
                    },
                    data: {
                        status: dbStatus,
                        startedAt: stepData.startedAt,
                        completedAt: stepData.completedAt,
                        timeElapsed,
                        // Store execution details in a custom field (would need schema update)
                        // For now, we'll store minimal data
                    },
                });
            } else {
                // Create new step record
                await DbProvider.get().run_step.create({
                    data: {
                        id: BigInt(generatePK()),
                        runId: BigInt(runId),
                        nodeId: stepData.stepId,
                        name: `Step ${stepData.stepId}`,
                        status: dbStatus,
                        startedAt: stepData.startedAt,
                        completedAt: stepData.completedAt,
                        timeElapsed,
                        complexity: this.estimateStepComplexity(stepData.resourceUsage),
                        order: 0, // Would need to track step order
                        resourceInId: stepData.stepId, // Using stepId as resource identifier
                    },
                });
            }

            logger.debug("[RunPersistenceService] Step execution recorded", {
                runId,
                stepId: stepData.stepId,
                dbStatus,
                timeElapsed,
            });

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to record step execution", {
                runId,
                stepId: stepData.stepId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Loads run state from the database
     */
    async loadRun(runId: string): Promise<{
        id: string;
        routineId: string;
        userId: string;
        status: string;
        inputs: Record<string, unknown>;
        outputs: Record<string, unknown>;
        metadata: Record<string, unknown>;
        createdAt: Date;
        startedAt?: Date;
        completedAt?: Date;
    } | null> {
        logger.debug("[RunPersistenceService] Loading run", { runId });

        try {
            const run = await DbProvider.get().run.findUnique({
                where: {
                    id: BigInt(runId),
                },
                include: {
                    resourceVersion: true,
                    io: true,
                    steps: {
                        orderBy: {
                            order: "asc",
                        },
                    },
                },
            });

            if (!run) {
                return null;
            }

            // Parse stored data
            let parsedData: any = {};
            try {
                parsedData = JSON.parse(run.data || "{}");
            } catch (error) {
                logger.warn("[RunPersistenceService] Failed to parse run data", {
                    runId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }

            // Reconstruct inputs from run_io
            const inputs: Record<string, unknown> = {};
            for (const io of run.io) {
                try {
                    inputs[io.nodeInputName] = JSON.parse(io.data);
                } catch (error) {
                    inputs[io.nodeInputName] = io.data;
                }
            }

            const result = {
                id: run.id.toString(),
                routineId: run.resourceVersion?.publicId || "unknown",
                userId: run.userId?.toString() || "unknown",
                status: run.status,
                inputs: { ...inputs, ...parsedData.inputs },
                outputs: parsedData.outputs || {},
                metadata: parsedData.metadata || {},
                createdAt: run.createdAt,
                startedAt: run.startedAt || undefined,
                completedAt: run.completedAt || undefined,
            };

            logger.debug("[RunPersistenceService] Run loaded", {
                runId,
                status: result.status,
                stepCount: run.steps.length,
            });

            return result;

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to load run", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Gets run execution history for a user
     */
    async getUserRunHistory(userId: string, limit = RunPersistenceService.DEFAULT_HISTORY_LIMIT, offset = 0): Promise<Array<{
        id: string;
        routineId: string;
        routineName: string;
        status: string;
        createdAt: Date;
        completedAt?: Date;
        duration?: number;
    }>> {
        logger.debug("[RunPersistenceService] Getting user run history", {
            userId,
            limit,
            offset,
        });

        try {
            const runs = await DbProvider.get().run.findMany({
                where: {
                    userId: BigInt(userId),
                },
                include: {
                    resourceVersion: {
                        include: {
                            translations: {
                                where: {
                                    language: "en",
                                },
                            },
                        },
                    },
                },
                orderBy: {
                    createdAt: "desc",
                },
                take: limit,
                skip: offset,
            });

            const history = runs.map(run => {
                // Type assertion needed due to Prisma include typing issue
                const runWithRelations = run as any;
                const routineName = runWithRelations.resourceVersion?.translations?.[0]?.name ||
                    `Routine ${runWithRelations.resourceVersion?.publicId || "Unknown"}`;

                let duration: number | undefined;
                if (run.startedAt && run.completedAt) {
                    duration = run.completedAt.getTime() - run.startedAt.getTime();
                }

                return {
                    id: run.id.toString(),
                    routineId: runWithRelations.resourceVersion?.publicId || "unknown",
                    routineName,
                    status: run.status,
                    createdAt: run.createdAt,
                    completedAt: run.completedAt || undefined,
                    duration,
                };
            });

            logger.debug("[RunPersistenceService] User run history loaded", {
                userId,
                count: history.length,
            });

            return history;

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to get user run history", {
                userId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Updates run outputs in the database
     */
    async updateRunOutputs(runId: string, outputs: Record<string, unknown>): Promise<void> {
        logger.debug("[RunPersistenceService] Updating run outputs", {
            runId,
            outputKeys: Object.keys(outputs),
        });

        try {
            // Get current run data
            const run = await DbProvider.get().run.findUnique({
                where: { id: BigInt(runId) },
                select: { data: true },
            });

            if (!run) {
                throw new Error(`Run not found: ${runId}`);
            }

            // Parse existing data
            const parsedData = JSON.parse(run.data || "{}");

            // Add outputs to the data
            parsedData.outputs = outputs;

            // Update run with new data
            await DbProvider.get().run.update({
                where: { id: BigInt(runId) },
                data: {
                    data: JSON.stringify(parsedData),
                    completedAt: new Date(),
                    updatedAt: new Date(),
                },
            });

            logger.info("[RunPersistenceService] Run outputs updated", {
                runId,
                outputCount: Object.keys(outputs).length,
            });

        } catch (error) {
            logger.error("[RunPersistenceService] Failed to update run outputs", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Private helper methods
     */
    private estimateStepComplexity(resourceUsage?: ExecutionResourceUsage): number {
        if (!resourceUsage) return 1;

        let complexity = 1;

        if (resourceUsage.toolCalls > RunPersistenceService.HIGH_TOOL_CALLS_THRESHOLD) complexity += 1;
        if (resourceUsage.durationMs > RunPersistenceService.LONG_DURATION_THRESHOLD_MS) complexity += 2;
        if (resourceUsage.memoryUsedMB > RunPersistenceService.HIGH_MEMORY_THRESHOLD_MB) complexity += 1;

        return Math.min(complexity, RunPersistenceService.MAX_COMPLEXITY);
    }
}
