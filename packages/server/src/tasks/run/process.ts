import { generatePK, type RoutineExecutionInput, type Success, type TierExecutionRequest } from "@vrooli/shared";
import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { SwarmContextManager } from "../../services/execution/shared/SwarmContextManager.js";
import { RoutineStateMachine } from "../../services/execution/tier2/routineStateMachine.js";
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
    // Add any routine-specific methods here
}
export const activeRunRegistry = new RunRegistry();

/**
 * Process routine execution directly through RoutineStateMachine
 * 
 * This uses RoutineStateMachine directly with RunTask structure
 * for clean integration with the three-tier architecture.
 */
async function processRoutineExecution(payload: RunTask) {
    logger.info("[processRoutineExecution] Starting routine execution", {
        runId: payload.input.runId,
        resourceVersionId: payload.input.resourceVersionId,
        userId: payload.context.userData.id,
    });

    try {
        // Extract run ID or generate new one if not provided
        const runId = payload.input.runId || generatePK().toString();

        const contextManager = new SwarmContextManager();

        // Create state machine directly with RunTask
        const stateMachine = new RoutineStateMachine(
            runId,
            contextManager,
            undefined, // runContextManager
            payload.context.userData.id,
        );

        // Create tier execution request from RunTask
        const tierRequest: TierExecutionRequest<RoutineExecutionInput> = {
            context: payload.context,
            input: {
                routineId: payload.input.resourceVersionId,
                parameters: payload.input.formValues || {},
                workflow: {
                    steps: [], // TODO: Load from database
                    dependencies: [],
                },
            },
            allocation: payload.allocation,
            options: {
                strategy: "sequential",
                timeout: 300000, // 5 minutes
            },
        };

        // Start routine execution using the state machine
        const result = await stateMachine.executeInForeground(tierRequest);
        if (!result.success) {
            throw new Error(result.error?.message || "Failed to start routine");
        }

        // Create registry record
        const record: ActiveRunRecord = {
            id: runId,
            userId: payload.context.userData.id,
            hasPremium: payload.context.userData.hasPremium || false,
            startTime: Date.now(),
        };

        // Add state machine to registry
        activeRunRegistry.add(record, stateMachine);

        logger.info("[processRoutineExecution] Successfully started routine", {
            runId,
        });

        return { runId };

    } catch (error) {
        logger.error("[processRoutineExecution] Failed to start routine execution", {
            runId: payload.input.runId,
            error: error instanceof Error ? error.message : String(error),
        });
        throw error;
    }
}

export async function runProcess({ data }: Job<RunTask>): Promise<Success> {
    switch (data.type) {
        case QueueTaskType.RUN_START:
            await processRoutineExecution(data as RunTask);
            return { __typename: "Success" as const, success: true };

        default:
            throw new CustomError("0331", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}
