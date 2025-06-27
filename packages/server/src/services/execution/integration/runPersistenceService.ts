import { type PrismaClient } from "@prisma/client";
import {
    type ResourceUsage,
    type RunState,
    type StepStatus,
    generatePK,
} from "@vrooli/shared";
import { type Logger } from "winston";

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

const STEP_STATUS_MAP = {
    "pending": "InProgress",
    "running": "InProgress",
    "completed": "Completed",
    "failed": "Failed",
    "skipped": "Skipped",
} as const;

/**
 * Run persistence service that stores execution state in Vrooli's database
 * 
 * This service manages the persistence of run data, integrating the execution
 * architecture with Vrooli's existing run tracking system.
 */
export class RunPersistenceService {
    private readonly prisma: PrismaClient;
    private readonly logger: Logger;

    constructor(prisma: PrismaClient, logger: Logger) {
        this.prisma = prisma;
        this.logger = logger;
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
        this.logger.debug("[RunPersistenceService] Creating run", {
            runId: runData.id,
            routineId: runData.routineId,
            userId: runData.userId,
        });

        try {
            // Find the resource version
            const resourceVersion = await this.prisma.resource_version.findFirst({
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
            await this.prisma.run.create({
                data: {
                    id: BigInt(runData.id),
                    name: `Execution ${runData.id.slice(-8)}`,
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
                await this.prisma.run_io.create({
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

            this.logger.info("[RunPersistenceService] Run created", {
                runId: runData.id,
                resourceVersionId: resourceVersion?.id,
            });

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to create run", {
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
        this.logger.debug("[RunPersistenceService] Updating run state", {
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

            await this.prisma.run.update({
                where: {
                    id: BigInt(runId),
                },
                data: updateData,
            });

            this.logger.debug("[RunPersistenceService] Run state updated", {
                runId,
                state,
                dbStatus,
            });

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to update run state", {
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
        state: StepStatus;
        startedAt: Date;
        completedAt?: Date;
        result?: Record<string, unknown>;
        error?: string;
        resourceUsage?: ResourceUsage;
    }): Promise<void> {
        this.logger.debug("[RunPersistenceService] Recording step execution", {
            runId,
            stepId: stepData.stepId,
            state: stepData.state,
        });

        try {
            const dbStatus = STEP_STATUS_MAP[stepData.state] || "InProgress";

            // Calculate time elapsed if completed
            let timeElapsed: number | undefined;
            if (stepData.completedAt) {
                timeElapsed = stepData.completedAt.getTime() - stepData.startedAt.getTime();
            }

            // Prepare step data
            const stepPersistenceData = {
                result: stepData.result,
                error: stepData.error,
                resourceUsage: stepData.resourceUsage,
            };

            // Try to find existing step record
            const existingStep = await this.prisma.run_step.findFirst({
                where: {
                    runId: BigInt(runId),
                    nodeId: stepData.stepId,
                },
            });

            if (existingStep) {
                // Update existing step
                await this.prisma.run_step.update({
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
                await this.prisma.run_step.create({
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

            this.logger.debug("[RunPersistenceService] Step execution recorded", {
                runId,
                stepId: stepData.stepId,
                dbStatus,
                timeElapsed,
            });

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to record step execution", {
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
        this.logger.debug("[RunPersistenceService] Loading run", { runId });

        try {
            const run = await this.prisma.run.findUnique({
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
                this.logger.warn("[RunPersistenceService] Failed to parse run data", {
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

            this.logger.debug("[RunPersistenceService] Run loaded", {
                runId,
                status: result.status,
                stepCount: run.steps.length,
            });

            return result;

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to load run", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Gets run execution history for a user
     */
    async getUserRunHistory(userId: string, limit = 20, offset = 0): Promise<Array<{
        id: string;
        routineId: string;
        routineName: string;
        status: string;
        createdAt: Date;
        completedAt?: Date;
        duration?: number;
    }>> {
        this.logger.debug("[RunPersistenceService] Getting user run history", {
            userId,
            limit,
            offset,
        });

        try {
            const runs = await this.prisma.run.findMany({
                where: {
                    userId: BigInt(userId),
                },
                include: {
                    resourceVersion: {
                        include: {
                            translations: {
                                where: {
                                    language: {
                                        code: "en",
                                    },
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
                const routineName = run.resourceVersion?.translations?.[0]?.name ||
                    `Routine ${run.resourceVersion?.publicId || "Unknown"}`;

                let duration: number | undefined;
                if (run.startedAt && run.completedAt) {
                    duration = run.completedAt.getTime() - run.startedAt.getTime();
                }

                return {
                    id: run.id.toString(),
                    routineId: run.resourceVersion?.publicId || "unknown",
                    routineName,
                    status: run.status,
                    createdAt: run.createdAt,
                    completedAt: run.completedAt || undefined,
                    duration,
                };
            });

            this.logger.debug("[RunPersistenceService] User run history loaded", {
                userId,
                count: history.length,
            });

            return history;

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to get user run history", {
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
        this.logger.debug("[RunPersistenceService] Updating run outputs", {
            runId,
            outputKeys: Object.keys(outputs),
        });

        try {
            // Get current run data
            const run = await this.prisma.run.findUnique({
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
            await this.prisma.run.update({
                where: { id: BigInt(runId) },
                data: {
                    data: JSON.stringify(parsedData),
                    completedAt: new Date(),
                    updatedAt: new Date(),
                },
            });

            this.logger.info("[RunPersistenceService] Run outputs updated", {
                runId,
                outputCount: Object.keys(outputs).length,
            });

        } catch (error) {
            this.logger.error("[RunPersistenceService] Failed to update run outputs", {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Private helper methods
     */
    private estimateStepComplexity(resourceUsage?: ResourceUsage): number {
        if (!resourceUsage) return 1;

        let complexity = 1;

        if (resourceUsage.tokens && resourceUsage.tokens > 1000) complexity += 1;
        if (resourceUsage.apiCalls && resourceUsage.apiCalls > 5) complexity += 1;
        if (resourceUsage.computeTime && resourceUsage.computeTime > 30000) complexity += 2;

        return Math.min(complexity, 10);
    }
}
