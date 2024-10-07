import { ChatMessage, ChatMessageCreateWithTaskInfoInput, ChatMessageSearchInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageUpdateWithTaskInfoInput, FindByIdInput, RegenerateResponseInput, Success, getTranslation, uuidValidate } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { ModelMap } from "../../models/base";
import { ChatMessageModelLogic, ChatModelInfo } from "../../models/base/types";
import { emitSocketEvent } from "../../sockets/events";
import { requestBotResponse } from "../../tasks/llm/queue";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";
import { ChatMessagePre, PreMapChatData, PreMapMessageDataUpdate, PreMapUserData, getChatParticipantData } from "../../utils/chat";
import { getSingleTypePermissions } from "../../validators/permissions";

export type EndpointsChatMessage = {
    Query: {
        chatMessage: GQLEndpoint<FindByIdInput, FindOneResult<ChatMessage>>;
        chatMessages: GQLEndpoint<ChatMessageSearchInput, FindManyResult<ChatMessage>>;
        chatMessageTree: GQLEndpoint<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>;
    },
    Mutation: {
        chatMessageCreate: GQLEndpoint<ChatMessageCreateWithTaskInfoInput, CreateOneResult<ChatMessage>>;
        chatMessageUpdate: GQLEndpoint<ChatMessageUpdateWithTaskInfoInput, UpdateOneResult<ChatMessage>>;
        regenerateResponse: GQLEndpoint<RegenerateResponseInput, Success>;
    }
}

const objectType = "ChatMessage";
export const ChatMessageEndpoints: EndpointsChatMessage = {
    Query: {
        chatMessage: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        chatMessages: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req });
        },
        chatMessageTree: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return ModelMap.get<ChatMessageModelLogic>("ChatMessage").query.searchTree(req, input, info);
        },
    },
    Mutation: {
        chatMessageCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            const { message, task, taskContexts } = input;
            const additionalData = { task, taskContexts };
            return createOneHelper({ additionalData, info, input: message, objectType, req });
        },
        chatMessageUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            const { message, task, taskContexts } = input;
            const additionalData = { task, taskContexts };
            return updateOneHelper({ additionalData, info, input: message, objectType, req });
        },
        //TODO remove if possible
        regenerateResponse: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            const { messageId, task, taskContexts } = input;
            if (!uuidValidate(messageId)) {
                throw new CustomError("0423", "InvalidArgs", userData.languages, { input });
            }
            const { canDelete: canRegenerateResponse } = await getSingleTypePermissions<ChatModelInfo["GqlPermission"]>("ChatMessage", [input.messageId], userData);
            // Use delete permissions to determine if we can regenerate a response, 
            // even though we keep the old message
            if (!Array.isArray(canRegenerateResponse) || !canRegenerateResponse.every(Boolean)) {
                throw new CustomError("0424", "Unauthorized", userData.languages, { input });
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
                throw new CustomError("0426", "InternalError", userData.languages, { messageId, parentMessageData, userId });
            }
            // Determine which bot should respond. 
            // The message should have already been created by a bot, 
            // but if not we can default to the first bot in the chat
            const respondingBotId = bots.find(b => b.id === regneratingMessageData.userId)?.id ?? bots[0].id;
            // Send typing indicator while bots are responding
            emitSocketEvent("typing", chatId, { starting: [respondingBotId] });
            // Call LLM for bot response
            const parentMessageText = getTranslation({ translations: parentMessageData.translations }, userData.languages).text || null;
            requestBotResponse({
                chatId,
                mode: "text",
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
    },
};
