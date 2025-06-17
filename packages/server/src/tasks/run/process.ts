import { type Job } from "bullmq";
import { generatePK } from "@vrooli/shared";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { TierTwoOrchestrator } from "../../services/execution/tier2/tierTwoOrchestrator.js";
import { EventBus } from "../../services/execution/cross-cutting/events/eventBus.js";
import { TierThreeExecutor } from "../../services/execution/tier3/TierThreeExecutor.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord, type ManagedTaskStateMachine } from "../activeTaskRegistry.js";
import { QueueTaskType, type RunTask } from "../taskTypes.js";
import { type Success } from "@vrooli/shared";

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

/**
 * Adapter to make the new TierTwoOrchestrator work with the existing ActiveRunRegistry
 */
export class NewRunStateMachineAdapter implements ManagedTaskStateMachine {
    constructor(
        private readonly runId: string,
        private readonly tierTwoOrchestrator: TierTwoOrchestrator,
        private readonly userId?: string,
    ) {}

    getTaskId(): string {
        return this.runId;
    }

    getCurrentSagaStatus(): string {
        // This is synchronous but we need to fetch status async
        // Return a default status for now - the registry will call this periodically
        return "RUNNING";
    }

    async requestPause(): Promise<boolean> {
        // For now, run execution doesn't support pause
        // This could be implemented in the future
        return false;
    }

    async requestStop(reason: string): Promise<boolean> {
        try {
            await this.tierTwoOrchestrator.cancelRun(this.runId, reason);
            return true;
        } catch (error) {
            logger.error("[NewRunStateMachineAdapter] Failed to stop run", {
                runId: this.runId,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    getAssociatedUserId?(): string | undefined {
        return this.userId;
    }
}

export type ActiveRunRecord = BaseActiveTaskRecord;
export class ActiveRunRegistry extends BaseActiveTaskRegistry<ActiveRunRecord, NewRunStateMachineAdapter> {
    // Add run-specific registry setup here
}
export const activeRunRegistry = new ActiveRunRegistry();

// Initialize the new execution services
// Lazy initialize to avoid database/cache initialization issues
let eventBus: EventBus | null = null;
let tierThreeExecutor: TierThreeExecutor | null = null;
let tierTwoOrchestrator: TierTwoOrchestrator | null = null;

function getTierTwoOrchestrator(): TierTwoOrchestrator {
    if (!eventBus) {
        eventBus = new EventBus(logger);
    }
    if (!tierThreeExecutor) {
        tierThreeExecutor = new TierThreeExecutor(logger, eventBus);
    }
    if (!tierTwoOrchestrator) {
        tierTwoOrchestrator = new TierTwoOrchestrator(logger, eventBus, tierThreeExecutor);
    }
    return tierTwoOrchestrator;
}

/**
 * Process new three-tier run execution
 */
async function processNewRunExecution(payload: RunTask) {
    logger.info("[processNewRunExecution] Starting new run execution", {
        runId: payload.runId,
        resourceVersionId: payload.resourceVersionId,
        userId: payload.userData.id,
        isNewRun: payload.isNewRun,
    });

    try {
        // Extract run ID - should always be provided in the payload
        const runId = payload.runId;

        // For new runs, we need to load the routine data from the database
        // For existing runs, we can resume from stored state
        if (payload.isNewRun) {
            // TODO: Load routine data from database using payload.resourceVersionId
            // For now, we'll use a placeholder
            const routineData = { workflow: {} }; // This should be loaded from DB

            // Start run through new three-tier architecture
            await getTierTwoOrchestrator().startRun({
                runId,
                swarmId: generatePK(), // Generate swarm ID for the run context
                routineVersionId: payload.resourceVersionId,
                routine: routineData,
                inputs: payload.formValues || {},
                config: {
                    strategy: payload.config?.botConfig?.strategy || 'reasoning',
                    model: payload.config?.botConfig?.model || 'gpt-4',
                    maxSteps: payload.config?.limits?.maxSteps || 50,
                    timeout: payload.config?.limits?.maxTime || 300000,
                },
                userId: payload.userData.id,
            });
        } else {
            // For existing runs, resume from current state
            // This would typically involve loading the run state and continuing
            logger.info("[processNewRunExecution] Resuming existing run", { runId });
            // TODO: Implement resume functionality
        }

        // Create adapter for registry management
        const adapter = new NewRunStateMachineAdapter(
            runId,
            getTierTwoOrchestrator(),
            payload.userData.id,
        );

        // Add to active run registry
        const record: ActiveRunRecord = {
            id: runId,
            userId: payload.userData.id,
            hasPremium: payload.userData.hasPremium || false,
            startTime: Date.now(),
        };

        activeRunRegistry.add(record, adapter);

        logger.info("[processNewRunExecution] Successfully started run", {
            runId,
        });

        return { success: true };

    } catch (error) {
        logger.error("[processNewRunExecution] Failed to start run execution", {
            runId: payload.runId,
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