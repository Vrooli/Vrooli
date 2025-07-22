import { generatePK, type RoutineExecutionInput, type Success, type TierExecutionRequest } from "@vrooli/shared";
import { type Job } from "bullmq";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { SwarmContextManager } from "../../services/execution/shared/SwarmContextManager.js";
import { RoutineExecutor } from "../../services/execution/tier2/routineExecutor.js";
import { type RoutineStateMachine } from "../../services/execution/tier2/routineStateMachine.js";
import { StepExecutor } from "../../services/execution/tier3/stepExecutor.js";
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
    // Starting routine execution

    try {
        // Extract run ID or generate new one if not provided
        const runId = payload.input.runId || generatePK().toString();

        // Check if run is already active (resumption check)
        if (!payload.input.isNewRun) {
            const existingStateMachine = activeRunRegistry.get(runId);
            if (existingStateMachine) {
                // Resuming existing execution
                await existingStateMachine.resume(); // Uses existing resume() infrastructure!
                return { runId };
            }
        }

        const contextManager = new SwarmContextManager(payload);

        // Create execution components for proper architecture
        const stepExecutor = new StepExecutor();
        const routineExecutor = new RoutineExecutor(
            contextManager,
            stepExecutor,
            runId, // contextId
            undefined, // runContextManager
            payload.context.userData.id,
            payload.context.parentSwarmId,
        );

        // Create tier execution request from RunTask with new interface
        const tierRequest: TierExecutionRequest<RoutineExecutionInput> = {
            context: payload.context,
            input: {
                resourceVersionId: payload.input.resourceVersionId,
                runId: payload.input.isNewRun ? undefined : runId, // Support resumption
                parameters: payload.input.formValues || {},
                workflow: {
                    steps: [], // TODO: Load from database
                    dependencies: [],
                },
            },
            allocation: payload.allocation,
            options: {
                timeout: 300000, // 5 minutes
                priority: "medium",
            },
        };

        // Start routine execution using the RoutineExecutor
        const result = await routineExecutor.execute(tierRequest);
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

        // Add state machine to registry (get it from the RoutineExecutor)
        activeRunRegistry.add(record, routineExecutor.getStateMachine());

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
