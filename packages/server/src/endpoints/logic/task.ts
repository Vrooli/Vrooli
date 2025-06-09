import { type CancelTaskInput, type CheckTaskStatusesInput, type CheckTaskStatusesResult, nanoid, PendingToolCallStatus, type RespondToToolApprovalInput, RunTriggeredFrom, type StartRunTaskInput, type StartSwarmTaskInput, type Success, TaskType } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { completionService } from "../../services/conversation/responseEngine.js";
import { SwarmExecutionService } from "../../services/execution/swarmExecutionService.js";
import { changeRunTaskStatus, getRunTaskStatuses, processRun } from "../../tasks/run/queue.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue.js";
import { activeSwarmRegistry } from "../../tasks/swarm/process.js";
import { changeSwarmTaskStatus, getSwarmTaskStatuses, processSwarm } from "../../tasks/swarm/queue.js";
import { QueueTaskType } from "../../tasks/taskTypes.js";
import type { ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

// Initialize the new three-tier execution service
const swarmExecutionService = new SwarmExecutionService(logger);

export type EndpointsTask = {
    checkStatuses: ApiEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    startSwarmTask: ApiEndpoint<StartSwarmTaskInput, Success>;
    startRunTask: ApiEndpoint<StartRunTaskInput, Success>;
    cancelTask: ApiEndpoint<CancelTaskInput, Success>;
    respondToToolApproval: ApiEndpoint<RespondToToolApprovalInput, Success>;
}

export const task: EndpointsTask = createStandardCrudEndpoints({
    objectType: "Task",
    endpoints: {},
    customEndpoints: {
        checkStatuses: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const result: CheckTaskStatusesResult = {
            __typename: "CheckTaskStatusesResult",
            statuses: [],
        };
        if (!input) return { __typename: "CheckTaskStatusesResult", statuses: [] };
        
        // Try to get status from new architecture first
        for (const taskId of input.taskIds) {
            try {
                // Check if it's a new architecture swarm/run
                if (taskId.startsWith("swarm-")) {
                    const status = await swarmExecutionService.getSwarmStatus(taskId);
                    // Convert to old format for compatibility
                    result.statuses.push({
                        id: taskId,
                        status: status.status,
                        progress: status.progress,
                        errors: status.errors,
                    });
                    continue;
                } else if (taskId.startsWith("run-")) {
                    const status = await swarmExecutionService.getRunStatus(taskId);
                    // Convert to old format for compatibility
                    result.statuses.push({
                        id: taskId,
                        status: status.status,
                        progress: status.progress,
                        errors: status.errors,
                    });
                    continue;
                }
            } catch (error) {
                // Fall through to old system
            }
        }
        
        // Fall back to old system for remaining tasks
        switch (input.taskType) {
            case TaskType.Llm:
                const oldSwarmStatuses = await getSwarmTaskStatuses(input.taskIds);
                result.statuses.push(...oldSwarmStatuses.filter(s => !result.statuses.find(r => r.id === s.id)));
                break;
            case TaskType.Run:
                const oldRunStatuses = await getRunTaskStatuses(input.taskIds);
                result.statuses.push(...oldRunStatuses.filter(s => !result.statuses.find(r => r.id === s.id)));
                break;
            case TaskType.Sandbox:
                result.statuses.push(...await getSandboxTaskStatuses(input.taskIds));
                break;
        }
        return result;
    },
    startSwarmTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        if (!input) return { __typename: "Success", success: false, error: "Input is required." };

        try {
            const swarmId = `swarm-${nanoid()}`;
            
            // Start swarm using the new three-tier architecture
            await swarmExecutionService.startSwarm({
                swarmId,
                name: `Chat Swarm ${swarmId}`,
                description: "Conversational AI swarm for chat completion",
                goal: "Provide helpful responses to user queries",
                resources: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000, // 1 hour
                    tools: input.taskContexts?.map(ctx => ({
                        name: ctx.toolName || "unknown",
                        description: ctx.description || "Tool from context",
                    })) || [],
                },
                config: {
                    model: input.model || "gpt-4o-mini",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 5,
                },
                userId: userData.id,
                organizationId: userData.organizationId,
            });

            // For backward compatibility, still use the old system for now
            // TODO: Fully migrate to new architecture
            const taskId = `task-${nanoid()}`;
            return processSwarm({
                type: QueueTaskType.LLM_COMPLETION,
                id: taskId,
                chatId: input.chatId,
                messageId: input.messageId,
                model: input.model,
                respondingBot: input.respondingBot,
                taskContexts: input.taskContexts,
                userData,
            });
        } catch (error) {
            logger.error("[task.startSwarmTask] Failed to start swarm with new architecture", {
                error: error instanceof Error ? error.message : String(error),
                userId: userData.id,
            });
            
            // Fallback to old system
            const taskId = `task-${nanoid()}`;
            return processSwarm({
                type: QueueTaskType.LLM_COMPLETION,
                id: taskId,
                chatId: input.chatId,
                messageId: input.messageId,
                model: input.model,
                respondingBot: input.respondingBot,
                taskContexts: input.taskContexts,
                userData,
            });
        }
    },
    startRunTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        if (!input) return { __typename: "Success", success: false, error: "Input is required." };

        try {
            const runId = `run-${nanoid()}`;
            
            // Start run using the new three-tier architecture
            // First, we need a swarm context for the run
            const swarmId = input.swarmId || `swarm-${nanoid()}`;
            
            // If no swarm exists, create one
            if (!input.swarmId) {
                await swarmExecutionService.startSwarm({
                    swarmId,
                    name: `Routine Execution Swarm ${swarmId}`,
                    description: "Swarm for executing routine runs",
                    goal: "Execute routine to completion",
                    resources: {
                        maxCredits: 50000,
                        maxTokens: 500000,
                        maxTime: 7200000, // 2 hours
                        tools: [], // Will be determined by routine
                    },
                    config: {
                        model: "gpt-4o-mini",
                        temperature: 0.3, // Lower temperature for routine execution
                        autoApproveTools: true, // Auto-approve for routine runs
                        parallelExecutionLimit: 10,
                    },
                    userId: userData.id,
                    organizationId: userData.organizationId,
                });
            }
            
            // Start the run
            await swarmExecutionService.startRun({
                runId,
                swarmId,
                routineVersionId: input.routineVersionId,
                inputs: input.inputsData || {},
                config: {
                    strategy: "reasoning", // Default to reasoning strategy for routines
                    model: "gpt-4o-mini",
                    maxSteps: 1000,
                    timeout: 3600000, // 1 hour
                },
                userId: userData.id,
            });

            // For backward compatibility, still process through old system
            // TODO: Fully migrate to new architecture
            const taskId = `task-${nanoid()}`;
            return processRun({
                ...input,
                runFrom: RunTriggeredFrom.RunView,
                startedById: userData.id,
                id: taskId,
                userData,
            });
        } catch (error) {
            logger.error("[task.startRunTask] Failed to start run with new architecture", {
                error: error instanceof Error ? error.message : String(error),
                userId: userData.id,
                routineVersionId: input.routineVersionId,
            });
            
            // Fallback to old system
            const taskId = `task-${nanoid()}`;
            return processRun({
                ...input,
                runFrom: RunTriggeredFrom.RunView,
                startedById: userData.id,
                id: taskId,
                userData,
            });
        }
    },
    cancelTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        if (!input) return { __typename: "Success", success: false, error: "Input is required for cancelTask" };

        switch (input.taskType) {
            case TaskType.Llm:
                return changeSwarmTaskStatus(input.taskId, "Suggested", userData.id);
            case TaskType.Run:
                return changeRunTaskStatus(input.taskId, "Suggested", userData.id);
            case TaskType.Sandbox:
                return changeSandboxTaskStatus(input.taskId, "Suggested", userData.id);
            default:
                return { __typename: "Success", success: false, error: "Unsupported task type for cancellation or task type not provided." };
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

            const pendingCall = conversationState.config.pendingToolCalls![pendingCallIndex];

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
                const swarmStateMachineInstance = activeSwarmRegistry.get(conversationId);
                if (swarmStateMachineInstance) {
                    logger.info(`Notifying SwarmStateMachine for conversation ${conversationId} about approved tool call ${pendingId}.`);
                    // The actual execution of the tool will be handled by the SwarmStateMachine
                    // in response to this notification, likely by queueing an internal event.
                    await swarmStateMachineInstance.handleToolApproval(pendingCall);
                } else {
                    logger.warn(`SwarmStateMachine instance not found in activeSwarmRegistry for conversation ${conversationId} after tool approval. The tool will not be executed immediately via this path.`);
                }
            } else {
                // If the tool call was rejected
                const swarmStateMachineInstance = activeSwarmRegistry.get(conversationId);
                if (swarmStateMachineInstance) {
                    logger.info(`Notifying SwarmStateMachine for conversation ${conversationId} about rejected tool call ${pendingId}. Reason: ${reason || "No reason provided"}`);
                    // Notify the SwarmStateMachine about the rejection.
                    await swarmStateMachineInstance.handleToolRejection(pendingCall, reason);
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
