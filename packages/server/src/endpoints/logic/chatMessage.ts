import { generatePK, validatePK, type ChatMessage, type ChatMessageCreateWithTaskInfoInput, type ChatMessageSearchInput, type ChatMessageSearchResult, type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, type ChatMessageUpdateWithTaskInfoInput, type FindByIdInput, type RegenerateResponseInput, type Success } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
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

export type EndpointsChatMessage = {
    findOne: ApiEndpoint<FindByIdInput, ChatMessage>;
    findMany: ApiEndpoint<ChatMessageSearchInput, ChatMessageSearchResult>;
    findTree: ApiEndpoint<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>;
    createOne: ApiEndpoint<ChatMessageCreateWithTaskInfoInput, ChatMessage>;
    updateOne: ApiEndpoint<ChatMessageUpdateWithTaskInfoInput, ChatMessage>;
    regenerateResponse: ApiEndpoint<RegenerateResponseInput, Success>;
}

const objectType = "ChatMessage";
export const chatMessage: EndpointsChatMessage = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req });
    },
    findTree: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return ModelMap.get<ChatMessageModelLogic>("ChatMessage").query.searchTree(req, input, info);
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        const { message, model, task, taskContexts } = input;
        const additionalData = { model, task, taskContexts };
        return createOneHelper({ additionalData, info, input: message, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        const { message, model, task, taskContexts } = input;
        const additionalData = { model, task, taskContexts };
        return updateOneHelper({ additionalData, info, input: message, objectType, req });
    },
    regenerateResponse: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });

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
        };

        await QueueService.get().swarm.addTask(llmTaskPayload);
        logger.info(`LLM task ${llmTaskPayload.id} enqueued for message regeneration: ${messageId}`);

        return { __typename: "Success", success: true };
    },
};
