// AI_CHECK: TYPE_SAFETY=2 | LAST: 2025-07-03
import { type CancelTaskInput, type CheckTaskStatusesInput, type CheckTaskStatusesResult, EventTypes, generatePK, nanoid, PendingToolCallStatus, type RespondToToolApprovalInput, RunTriggeredFrom, type StartRunTaskInput, type StartSwarmTaskInput, type Success, TaskType } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { EventPublisher } from "../../services/events/publisher.js";
import { CachedConversationStateStore, PrismaChatStore } from "../../services/response/chatStore.js";
import { QueueService } from "../../tasks/queues.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue.js";
import { QueueTaskType, type RunTask, type SwarmExecutionTask } from "../../tasks/taskTypes.js";
import type { ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

// Lazy imports to avoid circular dependencies
async function getSwarmQueue() {
    const module = await import("../../tasks/swarm/queue.js");
    return {
        processNewSwarmExecution: module.processNewSwarmExecution,
        getSwarmTaskStatuses: module.getSwarmTaskStatuses,
        changeSwarmTaskStatus: module.changeSwarmTaskStatus,
    };
}

async function getRunQueue() {
    const module = await import("../../tasks/run/queue.js");
    return {
        processRun: module.processRun,
        getRunTaskStatuses: module.getRunTaskStatuses,
        changeRunTaskStatus: module.changeRunTaskStatus,
    };
}

// Removed getSwarmRegistry - using EventBus instead of direct registry access

export type EndpointsTask = {
    checkStatuses: ApiEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    startSwarmTask: ApiEndpoint<StartSwarmTaskInput, Success>;
    startRunTask: ApiEndpoint<StartRunTaskInput, Success>;
    cancelTask: ApiEndpoint<CancelTaskInput, Success>;
    respondToToolApproval: ApiEndpoint<RespondToToolApprovalInput, Success>;
}

export const task: EndpointsTask = createStandardCrudEndpoints({
    endpoints: {},
    customEndpoints: {
        checkStatuses: async (data, { req }) => {
            const input = data?.input;
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            const result: CheckTaskStatusesResult = {
                __typename: "CheckTaskStatusesResult" as const,
                statuses: [],
            };
            if (!input) return { __typename: "CheckTaskStatusesResult" as const, statuses: [] };

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
            if (!input) return { __typename: "Success" as const, success: false, error: "Input is required." };

            const DEFAULT_TEMPERATURE = 0.7;
            const DEFAULT_PARALLEL_LIMIT = 3;

            try {
                const swarmId = generatePK().toString();

                // Create SwarmExecutionTask for the three-tier architecture
                const swarmTask: Omit<SwarmExecutionTask, "status"> = {
                    id: nanoid(),
                    type: QueueTaskType.SWARM_EXECUTION,
                    context: {
                        swarmId,
                        userData,
                        timestamp: new Date(),
                    },
                    input: {
                        goal: input.task.goal,
                        swarmId,
                        chatId: input.chatId,
                        teamConfiguration: input.task.teamConfiguration,
                        availableTools: input.task.availableTools?.map(tool => ({ name: tool, description: "" })),
                        executionConfig: {
                            model: input.model,
                            temperature: input.task.executionConfig?.temperature || DEFAULT_TEMPERATURE,
                            parallelExecutionLimit: input.task.executionConfig?.parallelExecutionLimit || DEFAULT_PARALLEL_LIMIT,
                        },
                        userData,
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
                return { __typename: "Success" as const, success: false, error: `Failed to start swarm task: ${error instanceof Error ? error.message : String(error)}` };
            }
        },
        startRunTask: async (data, { req }) => {
            const input = data?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            if (!input) return { __typename: "Success" as const, success: false, error: "Input is required." };

            try {
                const runId = input.runId || `run-${nanoid()}`;
                const swarmId = input.swarmId || `swarm-${nanoid()}`;

                // Create RunTask for the three-tier architecture
                const runTask: RunTask = {
                    id: nanoid(),
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
                return { __typename: "Success" as const, success: false, error: `Failed to start run task: ${error instanceof Error ? error.message : String(error)}` };
            }
        },
        cancelTask: async (data, { req }) => {
            const input = data?.input;
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            if (!input) return { __typename: "Success" as const, success: false, error: "Input is required for cancelTask" };

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
                        return { __typename: "Success" as const, success: false, error: "Unsupported task type for cancellation or task type not provided." };
                }
            } catch (error) {
                logger.error("[task.cancelTask] Failed to cancel task", {
                    error: error instanceof Error ? error.message : String(error),
                    taskId: input.taskId,
                    taskType: input.taskType,
                    userId: userData.id,
                });
                return { __typename: "Success" as const, success: false, error: `Failed to cancel task: ${error instanceof Error ? error.message : String(error)}` };
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

            try {
                logger.info(`Processing tool approval response for conversation ${conversationId}, pendingId ${pendingId}, approved: ${approved}`);

                // Create chat store instance for loading chat configuration
                const chatStore = new CachedConversationStateStore(new PrismaChatStore());

                // Get conversation state from the chat store
                const conversationState = await chatStore.get(conversationId);
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

                // Update the conversation config in the chat store
                await chatStore.updateConfig(conversationId, conversationState.config);

                logger.info(`Tool call ${pendingId} in conversation ${conversationId} was ${approved ? "approved" : "rejected"} by user ${userData.id}.`);

                // Publish tool approval/rejection event through EventBus
                // SwarmStateMachine listens for these events and handles them appropriately
                try {
                    const { proceed, reason: blockReason } = await EventPublisher.emit(
                        approved ? EventTypes.TOOL.APPROVAL_GRANTED : EventTypes.TOOL.APPROVAL_REJECTED,
                        {
                            chatId: conversationId,
                            pendingId,
                            toolCallId: pendingCall.toolCallId,
                            toolName: pendingCall.toolName,
                            callerBotId: pendingCall.callerBotId,
                            ...(approved ? { approvedBy: userData.id } : { reason: reason || "Tool use rejected by user" }),
                        },
                    );

                    if (!proceed) {
                        logger.error(`Tool ${approved ? "approval" : "rejection"} event was blocked`, {
                            conversationId,
                            pendingId,
                            toolName: pendingCall.toolName,
                            blockReason,
                        });
                        // This is a critical security event - if it's blocked, we should fail the operation
                        throw new CustomError("0701", "ToolApprovalEventBlocked", { reason: blockReason });
                    }

                    logger.info(`Published tool ${approved ? "approval" : "rejection"} event for conversation ${conversationId}`, {
                        pendingId,
                        toolName: pendingCall.toolName,
                    });
                } catch (eventError) {
                    logger.error("Error publishing tool approval event", {
                        conversationId,
                        pendingId,
                        error: eventError instanceof Error ? eventError.message : String(eventError),
                    });
                    // Re-throw for security-critical events
                    throw eventError;
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
