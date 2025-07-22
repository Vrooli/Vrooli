import { generatePK, validatePK, type ChatMessage, type ChatMessageCreateWithTaskInfoInput, type ChatMessageSearchInput, type ChatMessageSearchResult, type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, type ChatMessageUpdateWithTaskInfoInput, type FindByIdInput, type RegenerateResponseInput, type Success } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ModelMap } from "../../models/base/index.js";
import { type ChatMessageModelLogic, type ChatModelInfo } from "../../models/base/types.js";
import { QueueService } from "../../tasks/queues.js";
import { QueueTaskType, type LLMCompletionTask } from "../../tasks/taskTypes.js";
import { type ApiEndpoint } from "../../types.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsChatMessage = {
    findOne: ApiEndpoint<FindByIdInput, ChatMessage>;
    findMany: ApiEndpoint<ChatMessageSearchInput, ChatMessageSearchResult>;
    findTree: ApiEndpoint<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>;
    createOne: ApiEndpoint<ChatMessageCreateWithTaskInfoInput, ChatMessage>;
    updateOne: ApiEndpoint<ChatMessageUpdateWithTaskInfoInput, ChatMessage>;
    regenerateResponse: ApiEndpoint<RegenerateResponseInput, Success>;
}

const objectType = "ChatMessage";
export const chatMessage: EndpointsChatMessage = createStandardCrudEndpoints({
    objectType,
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        createOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
            customImplementation: async ({ input, req, info }) => {
                const { message, model, task, taskContexts } = input;
                const additionalData = { model, task, taskContexts };
                return createOneHelper({ additionalData, info, input: message, objectType, req });
            },
        },
        updateOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
            customImplementation: async ({ input, req, info }) => {
                const { message, model, task, taskContexts } = input;
                const additionalData = { model, task, taskContexts };
                return updateOneHelper({ additionalData, info, input: message, objectType, req });
            },
        },
    },
    customEndpoints: {
        findTree: async (data, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
            const input = data?.input;
            return ModelMap.get<ChatMessageModelLogic>("ChatMessage").query.searchTree(req, input, info);
        },
        regenerateResponse: async (data, { req }) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });

            const input = data?.input;
            const { messageId, model, taskContexts } = input;
            if (!validatePK(messageId)) {
                throw new CustomError("0423", "InvalidArgs", { input });
            }
            const { canDelete: canRegenerateResponse } = await getSingleTypePermissions<ChatModelInfo["ApiPermission"]>("ChatMessage", [input.messageId], userData);
            // Use delete permissions to determine if we can regenerate a response, 
            // even though we keep the old message
            if (!Array.isArray(canRegenerateResponse) || !canRegenerateResponse.every(Boolean)) {
                throw new CustomError("0424", "Unauthorized", { input });
            }

            // Fetch the original message to get its chatId
            const originalMessage = await readOneHelper<ChatMessageModelLogic>({
                req,
                objectType,
                input: { id: messageId },
                info: { select: { chatId: true } },
            });

            if (!originalMessage || !originalMessage.chatId) {
                logger.error("Original message not found or chatId missing for regeneration", { messageId });
                throw new CustomError("0017", "NotFound", { message: `Original message ${messageId} not found or missing chat ID.` });
            }

            const llmTaskPayload: LLMCompletionTask = {
                id: generatePK().toString(),
                type: QueueTaskType.LLM_COMPLETION,
                chatId: originalMessage.chatId as string,
                messageId,
                userData,
                taskContexts: taskContexts ?? [],
                model,
                allocation: {
                    maxCredits: userData.hasPremium ? "200" : "100",
                    maxDurationMs: 60000,
                    maxMemoryMB: 256,
                    maxConcurrentSteps: 1,
                },
                options: {
                    priority: userData.hasPremium ? "high" : "medium",
                    timeout: 60000,
                    retryPolicy: {
                        maxRetries: 3,
                        backoffMs: 1000,
                        backoffMultiplier: 2,
                        maxBackoffMs: 30000,
                    },
                },
            };

            await QueueService.get().swarm.addTask(llmTaskPayload);

            return { __typename: "Success" as const, success: true };
        },
    },
});
