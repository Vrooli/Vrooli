import { type CancelTaskInput, type CheckTaskStatusesInput, type CheckTaskStatusesResult, nanoid, PendingToolCallStatus, type RespondToToolApprovalInput, type RunTask, RunTriggeredFrom, type StartRunTaskInput, type StartSwarmTaskInput, type Success, type SwarmExecutionTask, TaskType } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { completionService } from "../../services/conversation/responseEngine.js";
import { QueueService } from "../../tasks/queues.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue.js";
import { QueueTaskType } from "../../tasks/taskTypes.js";
import type { ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

// Lazy imports to avoid circular dependencies
async function getSwarmQueue() {
    const { processNewSwarmExecution, getSwarmTaskStatuses, changeSwarmTaskStatus } = await import("../../tasks/swarm/queue.js");
    return { processNewSwarmExecution, getSwarmTaskStatuses, changeSwarmTaskStatus };
}

async function getRunQueue() {
    const { processRun, getRunTaskStatuses, changeRunTaskStatus } = await import("../../tasks/run/queue.js");
    return { processRun, getRunTaskStatuses, changeRunTaskStatus };
}

async function getSwarmRegistry() {
    const { activeSwarmRegistry } = await import("../../tasks/swarm/process.js");
    return { activeSwarmRegistry };
}

export type EndpointsTask = {
    checkStatuses: ApiEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    startSwarmTask: ApiEndpoint<StartSwarmTaskInput, Success>;
    startRunTask: ApiEndpoint<StartRunTaskInput, Success>;
    cancelTask: ApiEndpoint<CancelTaskInput, Success>;
    respondToToolApproval: ApiEndpoint<RespondToToolApprovalInput, Success>;
}

export const task: EndpointsTask = createStandardCrudEndpoints({
    objectType: "Task" as const,
    endpoints: {},
    customEndpoints: {
        checkStatuses: async (data, { req }) => {
            const input = data?.input;
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            const result: CheckTaskStatusesResult = {
                __typename: "CheckTaskStatusesResult",
                statuses: [],
            };
            if (!input) return { __typename: "CheckTaskStatusesResult", statuses: [] };

            // Get task statuses from queue system
            switch (input.taskType) {
                case TaskType.Llm: {
                    const { getSwarmTaskStatuses } = await getSwarmQueue();
                    const swarmStatuses = await getSwarmTaskStatuses(input.taskIds, QueueService.get());
                    result.statuses.push(...swarmStatuses);
                    break;
                }
                case TaskType.Run: {
                    const { getRunTaskStatuses } = await getRunQueue();
                    const runStatuses = await getRunTaskStatuses(input.taskIds, QueueService.get());
                    result.statuses.push(...runStatuses);
                    break;
                }
                case TaskType.Sandbox:
                    result.statuses.push(...await getSandboxTaskStatuses(input.taskIds, QueueService.get()));
                    break;
            }
            return result;
        },
        startSwarmTask: async (data, { req }) => {
            const input = data?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            if (!input) return { __typename: "Success", success: false, error: "Input is required." };

            try {
                const swarmId = `swarm-${nanoid()}`;

                // Create SwarmExecutionTask for the three-tier architecture
                const swarmTask: Omit<SwarmExecutionTask, "status"> = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    context: {
                        swarmId,
                        userData,
                        timestamp: new Date(),
                    },
                    input: {
                        goal: "Provide helpful responses to user queries",
                        chatId: input.chatId,
                        messageId: input.messageId,
                        model: input.model || "gpt-4o-mini",
                        respondingBot: input.respondingBot,
                        taskContexts: input.taskContexts || [],
                    },
                    allocation: {
                        maxCredits: "10000",
                        maxDurationMs: 3600000, // 1 hour
                        maxMemoryMB: 512,
                        maxConcurrentSteps: 5,
                    },
                };

                // Queue the swarm task
                const { processNewSwarmExecution } = await getSwarmQueue();
                return await processNewSwarmExecution(swarmTask, QueueService.get());
            } catch (error) {
                logger.error("[task.startSwarmTask] Failed to queue swarm task", {
                    error: error instanceof Error ? error.message : String(error),
                    userId: userData.id,
                });
                return { __typename: "Success", success: false, error: `Failed to start swarm task: ${error instanceof Error ? error.message : String(error)}` };
            }
        },
        startRunTask: async (data, { req }) => {
            const input = data?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            if (!input) return { __typename: "Success", success: false, error: "Input is required." };

            try {
                const runId = input.runId || `run-${nanoid()}`;
                const swarmId = input.swarmId || `swarm-${nanoid()}`;

                // Create RunTask for the three-tier architecture
                const runTask: Omit<RunTask, "status"> = {
                    type: QueueTaskType.RUN_START,
                    context: {
                        swarmId,
                        userData,
                        timestamp: new Date(),
                    },
                    input: {
                        runId,
                        resourceVersionId: input.routineVersionId,
                        config: input.config,
                        formValues: input.formValues,
                        isNewRun: input.isNewRun,
                        runFrom: RunTriggeredFrom.RunView,
                        startedById: userData.id,
                        status: "Scheduled",
                    },
                    allocation: {
                        maxCredits: "50000",
                        maxDurationMs: 7200000, // 2 hours
                        maxMemoryMB: 1024,
                        maxConcurrentSteps: 10,
                    },
                };

                // Queue the run task
                const { processRun } = await getRunQueue();
                return await processRun(runTask, QueueService.get());
            } catch (error) {
                logger.error("[task.startRunTask] Failed to queue run task", {
                    error: error instanceof Error ? error.message : String(error),
                    userId: userData.id,
                    routineVersionId: input.routineVersionId,
                });
                return { __typename: "Success", success: false, error: `Failed to start run task: ${error instanceof Error ? error.message : String(error)}` };
            }
        },
        cancelTask: async (data, { req }) => {
            const input = data?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            if (!input) return { __typename: "Success", success: false, error: "Input is required for cancelTask" };

            try {
                switch (input.taskType) {
                    case TaskType.Llm: {
                        const { changeSwarmTaskStatus } = await getSwarmQueue();
                        return await changeSwarmTaskStatus(input.taskId, "Suggested", userData.id, QueueService.get());
                    }
                    case TaskType.Run: {
                        const { changeRunTaskStatus } = await getRunQueue();
                        return await changeRunTaskStatus(input.taskId, "Suggested", userData.id, QueueService.get());
                    }
                    case TaskType.Sandbox:
                        return await changeSandboxTaskStatus(input.taskId, "Suggested", userData.id, QueueService.get());
                    default:
                        return { __typename: "Success", success: false, error: "Unsupported task type for cancellation or task type not provided." };
                }
            } catch (error) {
                logger.error("[task.cancelTask] Failed to cancel task", {
                    error: error instanceof Error ? error.message : String(error),
                    taskId: input.taskId,
                    taskType: input.taskType,
                    userId: userData.id,
                });
                return { __typename: "Success", success: false, error: `Failed to cancel task: ${error instanceof Error ? error.message : String(error)}` };
            }
        },
        respondToToolApproval: async (payload, { req }) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });

            if (!payload || !payload.input) {
                logger.error("RespondToToolApproval endpoint called with no input.");
                return { __typename: "Success" as const, success: false, error: "Request input is missing." };
            }
            const { input } = payload;

            const { conversationId, pendingId, approved, reason } = input;

            // TODO: UI update - The frontend needs to be updated to correctly route tool approval responses to this endpoint
            // and handle the outcome (e.g., update UI based on success/failure of processing the approval,
            // display pending tool calls, and allow interaction).
            // TODO: Shared package update - Ensure `RespondToToolApprovalInput` is defined in `@vrooli/shared`.

            try {
                logger.info(`Processing tool approval response for conversation ${conversationId}, pendingId ${pendingId}, approved: ${approved}`);

                const conversationState = await completionService.getConversationState(conversationId);
                if (!conversationState) {
                    logger.error(`Conversation state not found for ID: ${conversationId} while responding to tool approval.`);
                    return { __typename: "Success" as const, success: false, error: "Conversation not found." };
                }

                const pendingCallIndex = conversationState.config.pendingToolCalls?.findIndex(call => call.pendingId === pendingId);

                if (pendingCallIndex === undefined || pendingCallIndex === -1) {
                    logger.warn(`Pending tool call with ID ${pendingId} not found in conversation ${conversationId}. It might have timed out or been processed already.`);
                    return { __typename: "Success" as const, success: false, error: "Pending tool call not found or already processed." };
                }

                const pendingCall = conversationState.config.pendingToolCalls?.[pendingCallIndex];
                if (!pendingCall) {
                    logger.warn(`Pending tool call with ID ${pendingId} not found in conversation ${conversationId}.`);
                    return { __typename: "Success" as const, success: false, error: "Pending tool call not found." };
                }

                if (pendingCall.status !== PendingToolCallStatus.PENDING_APPROVAL) {
                    logger.warn(`Pending tool call ${pendingId} in conversation ${conversationId} is not in PENDING_APPROVAL state. Current state: ${pendingCall.status}.`);
                    return { __typename: "Success" as const, success: false, error: `Tool call is not awaiting approval (current state: ${pendingCall.status}).` };
                }

                // Update the pending call status
                pendingCall.status = approved ? PendingToolCallStatus.APPROVED_READY_FOR_EXECUTION : PendingToolCallStatus.REJECTED_BY_USER;
                pendingCall.decisionTime = Date.now();
                pendingCall.approvedOrRejectedByUserId = userData.id;
                pendingCall.statusReason = reason;

                completionService.updateConversationConfig(conversationId, conversationState.config);

                logger.info(`Tool call ${pendingId} in conversation ${conversationId} was ${approved ? "approved" : "rejected"} by user ${userData.id}.`);

                if (approved) {
                    const { activeSwarmRegistry } = await getSwarmRegistry();
                    const swarmStateMachineInstance = activeSwarmRegistry.get(conversationId);
                    if (swarmStateMachineInstance) {
                        logger.info(`Notifying SwarmStateMachine for conversation ${conversationId} about approved tool call ${pendingId}.`);
                        // The actual execution of the tool will be handled by the SwarmStateMachine
                        // in response to this notification, likely by queueing an internal event.
                        // TODO: handleToolApproval method doesn't exist on SwarmStateMachine
                        // await swarmStateMachineInstance.handleToolApproval(pendingCall);
                    } else {
                        logger.warn(`SwarmStateMachine instance not found in activeSwarmRegistry for conversation ${conversationId} after tool approval. The tool will not be executed immediately via this path.`);
                    }
                } else {
                    // If the tool call was rejected
                    const { activeSwarmRegistry } = await getSwarmRegistry();
                    const swarmStateMachineInstance = activeSwarmRegistry.get(conversationId);
                    if (swarmStateMachineInstance) {
                        logger.info(`Notifying SwarmStateMachine for conversation ${conversationId} about rejected tool call ${pendingId}. Reason: ${reason || "No reason provided"}`);
                        // Notify the SwarmStateMachine about the rejection.
                        // TODO: handleToolRejection method doesn't exist on SwarmStateMachine
                        // await swarmStateMachineInstance.handleToolRejection(pendingCall, reason);
                    } else {
                        logger.warn(`SwarmStateMachine instance not found in activeSwarmRegistry for conversation ${conversationId} after tool rejection. The rejection may not be fully processed by the swarm logic immediately.`);
                        // Even if the swarm instance isn't active, the config was updated. 
                        // The swarm might pick up the rejection when it next processes the conversation state.
                    }
                }

                return { __typename: "Success" as const, success: true };
            } catch (error) {
                logger.error("Error processing tool approval response:", { error, conversationId, pendingId });
                const errorMessage = error instanceof Error ? error.message : "An unknown error occurred.";
                return { __typename: "Success" as const, success: false, error: `Failed to process approval: ${errorMessage}` };
            }
        },
    },
});
