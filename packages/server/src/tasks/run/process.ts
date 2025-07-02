import { type Success, type TierExecutionRequest, type RoutineExecutionInput } from "@vrooli/shared";
import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { RoutineOrchestrator } from "../../services/execution/tier2/routineOrchestrator.js";
import type { RoutineStateMachine } from "../../services/execution/tier2/routineStateMachine.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import { QueueTaskType, type RunTask } from "../taskTypes.js";

/**
 * The fields to select for various run-related objects
 * TODO: This may need to be updated for the new execution architecture
 */
export const RunProcessSelect = {
    Run: {
        id: true,
        completedComplexity: true,
        contextSwitches: true,
        isPrivate: true,
        io: {
            id: true,
            data: true,
            nodeInputName: true,
            nodeName: true,
        },
        name: true,
        resourceVersion: {
            id: true,
            complexity: true,
            isDeleted: true,
            isPrivate: true,
            resourceSubType: true,
            root: {
                id: true,
                ownedByTeam: {
                    id: true,
                    isPrivate: true,
                },
                ownedByUser: {
                    id: true,
                    isPrivate: true,
                },
            },
        },
        status: true,
        steps: {
            id: true,
            complexity: true,
            contextSwitches: true,
            name: true,
            nodeId: true,
            order: true,
            resourceInId: true,
            resourceVersionId: true,
            startedAt: true,
            status: true,
            timeElapsed: true,
        },
        timeElapsed: true,
    },
    ResourceVersion: {
        id: true,
        codeLanguage: true,
        config: true,
        complexity: true,
        isAutomatable: true,
        isDeleted: true,
        isPrivate: true,
        relatedVersions: {
            id: true,
            labels: true,
            toVersion: {
                id: true,
                codeLanguage: true,
                config: true,
            },
        },
        resourceSubType: true,
        root: {
            id: true,
            ownedByTeam: {
                id: true,
                isPrivate: true,
            },
            ownedByUser: {
                id: true,
                isPrivate: true,
            },
            resourceType: true,
        },
        translations: {
            id: true,
            language: true,
            description: true,
            details: true,
            instructions: true,
            name: true,
        },
    },
} as const;

export type ActiveRunRecord = BaseActiveTaskRecord;
class RunRegistry extends BaseActiveTaskRegistry<ActiveRunRecord, RoutineStateMachine> {
    // Add any swarm-specific methods here
}
export const activeRunRegistry = new RunRegistry();

/**
 * Process new three-tier run execution
 */
async function processNewRunExecution(payload: RunTask) {
    // Extract fields from the new nested structure
    const { context, input, allocation } = payload;
    
    logger.info("[processNewRunExecution] Starting new run execution", {
        runId: input.runId,
        resourceVersionId: input.resourceVersionId,
        userId: context.userData.id,
        isNewRun: input.isNewRun,
    });

    try {
        // TODO: Fetch routine data from database using input.resourceVersionId
        // For now, we'll need to implement this properly
        const routineData = {
            routineId: input.resourceVersionId,
            parameters: input.formValues || {},
            workflow: {
                steps: [],
                dependencies: [],
            },
        };

        // Create RoutineExecutionInput for the orchestrator
        const routineRequest: TierExecutionRequest<RoutineExecutionInput> = {
            context,
            input: routineData,
            allocation,
            options: {
                // Add any execution options if needed
            },
        };

        // Execute directly through coordinator
        const coordinator = new RoutineOrchestrator();
        const result = await coordinator.execute(routineRequest);
        if (!result.success) {
            throw new Error(result.error?.message || "Failed to start run");
        }

        // Create registry record
        const record: ActiveRunRecord = {
            id: input.runId,
            userId: context.userData.id,
            hasPremium: context.userData.hasPremium || false,
            startTime: Date.now(),
        };

        // TODO: Need to get the actual RoutineStateMachine instance from RoutineOrchestrator
        // For now, we'll comment out the registry add until we can get the proper instance
        // activeRunRegistry.add(record, coordinator);

        logger.info("[processNewRunExecution] Successfully started run", {
            runId: input.runId,
        });

        return { runId: input.runId };

    } catch (error) {
        logger.error("[processNewRunExecution] Failed to start run execution", {
            runId: input.runId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

export async function runProcess({ data }: Job<RunTask>): Promise<Success> {
    switch (data.type) {
        case QueueTaskType.RUN_START:
            await processNewRunExecution(data as RunTask);
            return { __typename: "Success" as const, success: true };

        default:
            throw new CustomError("0331", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}
