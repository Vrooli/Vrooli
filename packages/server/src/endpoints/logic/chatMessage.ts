import { ChatMessage, ChatMessageCreateWithTaskInfoInput, ChatMessageSearchInput, ChatMessageSearchResult, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageUpdateWithTaskInfoInput, FindByIdInput, RegenerateResponseInput, Success, getTranslation, validatePK } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { ChatMessageModelLogic, ChatModelInfo } from "../../models/base/types.js";
import { SocketService } from "../../sockets/io.js";
import { requestBotResponse } from "../../tasks/llm/queue.js";
import { ApiEndpoint } from "../../types.js";
import { ChatMessagePre, PreMapChatData, PreMapMessageDataUpdate, PreMapUserData, getChatParticipantData } from "../../utils/chat.js";
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
    //TODO remove if possible
    regenerateResponse: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });

        const { messageId, model, task, taskContexts } = input;
        if (!validatePK(messageId)) {
            throw new CustomError("0423", "InvalidArgs", { input });
        }
        const { canDelete: canRegenerateResponse } = await getSingleTypePermissions<ChatModelInfo["ApiPermission"]>("ChatMessage", [input.messageId], userData);
        // Use delete permissions to determine if we can regenerate a response, 
        // even though we keep the old message
        if (!Array.isArray(canRegenerateResponse) || !canRegenerateResponse.every(Boolean)) {
            throw new CustomError("0424", "Unauthorized", { input });
        }
        // Initialize object to store queried information
        const preMap: ChatMessagePre = { chatData: {}, messageData: {}, userData: {} };
        // Collect chat and participant information
        const userId = userData.id;
        await getChatParticipantData({
            includeMessageInfo: true,
            includeMessageParentInfo: true,
            messageIds: [messageId],
            preMap,
            userData,
        });
        const parentMessageData = Object.values(preMap.messageData).find(m => m.messageId !== messageId) as PreMapMessageDataUpdate | undefined;
        const regneratingMessageData = preMap.messageData[messageId] as PreMapMessageDataUpdate | undefined;
        const chatId = parentMessageData?.chatId;
        const chat: PreMapChatData | undefined = chatId ? preMap.chatData[chatId] : undefined;
        const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMap.userData[id]).filter(b => b) ?? [];
        if (
            !parentMessageData ||
            !parentMessageData.userId ||
            !regneratingMessageData ||
            !regneratingMessageData.userId ||
            !chatId ||
            !chat ||
            bots.length === 0
        ) {
            throw new CustomError("0426", "InternalError", { messageId, parentMessageData, userId });
        }
        // Determine which bot should respond. 
        // The message should have already been created by a bot, 
        // but if not we can default to the first bot in the chat
        const respondingBotId = bots.find(b => b.id === regneratingMessageData.userId)?.id ?? bots[0].id;
        // Send typing indicator while bots are responding
        SocketService.get().emitSocketEvent("typing", chatId, { starting: [respondingBotId] });
        // Call LLM for bot response
        const parentMessageText = getTranslation({ translations: parentMessageData.translations }, userData.languages).text || null;
        requestBotResponse({
            chatId,
            mode: "text",
            model,
            parentId: parentMessageData.messageId,
            parentMessage: parentMessageText,
            respondingBotId,
            shouldNotRunTasks: false,
            task,
            taskContexts,
            participantsData: preMap.userData,
            userData,
        });
        return { __typename: "Success", success: true };
    },
};
