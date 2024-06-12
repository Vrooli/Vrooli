import { AutoFillInput, AutoFillResult, CancelTaskInput, ChatMessage, ChatMessageCreateInput, ChatMessageSearchInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageUpdateInput, CheckTaskStatusesInput, CheckTaskStatusesResult, FindByIdInput, RegenerateResponseInput, StartTaskInput, Success, uuidValidate } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { ModelMap } from "../../models/base";
import { ChatMessageModelLogic } from "../../models/base/types";
import { emitSocketEvent } from "../../sockets/events";
import { determineRespondingBots } from "../../tasks/llm/context";
import { llmProcessAutoFill } from "../../tasks/llm/process";
import { requestBotResponse, requestStartTask } from "../../tasks/llm/queue";
import { changeLlmTaskStatus, getLlmTaskStatuses } from "../../tasks/llmTask";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";
import { PreMapChatData, PreMapMessageData, PreMapUserData, getChatParticipantData } from "../../utils/chat";
import { getSingleTypePermissions } from "../../validators/permissions";

export type EndpointsChatMessage = {
    Query: {
        chatMessage: GQLEndpoint<FindByIdInput, FindOneResult<ChatMessage>>;
        chatMessages: GQLEndpoint<ChatMessageSearchInput, FindManyResult<ChatMessage>>;
        chatMessageTree: GQLEndpoint<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>;
    },
    Mutation: {
        chatMessageCreate: GQLEndpoint<ChatMessageCreateInput, CreateOneResult<ChatMessage>>;
        chatMessageUpdate: GQLEndpoint<ChatMessageUpdateInput, UpdateOneResult<ChatMessage>>;
        regenerateResponse: GQLEndpoint<RegenerateResponseInput, Success>;
        autoFill: GQLEndpoint<AutoFillInput, AutoFillResult>;
        startTask: GQLEndpoint<StartTaskInput, Success>;
        cancelTask: GQLEndpoint<CancelTaskInput, Success>;
        checkTaskStatuses: GQLEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
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
            return createOneHelper({ info, input, objectType, req });
        },
        chatMessageUpdate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        regenerateResponse: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            if (!uuidValidate(input.messageId)) {
                throw new CustomError("0423", "InvalidArgs", userData.languages, { input });
            }
            const { canDelete: canRegenerateResponse } = await getSingleTypePermissions("ChatMessage", [input.messageId], userData);
            // Use delete permissions to determine if we can regenerate a response, 
            // even though we keep the old message
            if (!canRegenerateResponse) {
                throw new CustomError("0424", "Unauthorized", userData.languages, { input });
            }
            // Initialize objects to store queried information
            const preMapChatData: Record<string, PreMapChatData> = {};
            const preMapMessageData: Record<string, PreMapMessageData> = {};
            const preMapUserData: Record<string, PreMapUserData> = {};
            // Collect chat and participant information
            const messageId = input.messageId;
            const userId = userData.id;
            await getChatParticipantData({
                includeMessageInfo: true,
                includeMessageParentInfo: true,
                messageIds: [messageId],
                preMapChatData,
                preMapMessageData,
                preMapUserData,
                userData,
            });
            let messageData = preMapMessageData[messageId];
            const chatId = messageData?.chatId;
            const chat: PreMapChatData | undefined = chatId ? preMapChatData[chatId] : undefined;
            const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMapUserData[id]).filter(b => b) ?? [];
            if (
                !messageData ||
                !messageData.userId ||
                !chatId ||
                !chat ||
                bots.length === 0
            ) {
                throw new CustomError("0426", "InternalError", userData.languages, { messageId, messageData, userId });
            }
            // Determine which bot should respond.
            let respondingBotId: string | null = null;
            // If the message is already a bot message
            if (bots.some(b => b.id === messageData.userId)) {
                // Use the same bot to respond
                respondingBotId = messageData.userId;
                // Set the parent as the message we're responding to, if it exists
                if (messageData.parentId && preMapMessageData[messageData.parentId]) {
                    messageData = preMapMessageData[messageData.parentId];
                }
            }
            // Otherwise, use helper function determine which bots should respond.
            else {
                const botsToRespond = determineRespondingBots(messageData, chat, bots, userData.id);
                if (botsToRespond.length) {
                    // For now, we'll limit regenerations to a single bot
                    respondingBotId = botsToRespond[0];
                }
            }
            // If we found a bot to respond, then request a response
            if (respondingBotId) {
                // Send typing indicator while bots are responding
                emitSocketEvent("typing", chatId, { starting: [respondingBotId] });
                // Call LLM for bot response
                requestBotResponse({
                    chatId,
                    parent: messageData,
                    respondingBotId,
                    task: "Start", //TODO
                    participantsData: preMapUserData,
                    userData,
                });
            }
            return { __typename: "Success", success: true };
        },
        autoFill: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return llmProcessAutoFill({ ...input, userData, __process: "AutoFill" });
        },
        startTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return requestStartTask({ ...input, userData });
        },
        cancelTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return changeLlmTaskStatus(input.taskId, "Suggested", userData.id);
        },
        checkTaskStatuses: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });

            const statuses = await getLlmTaskStatuses(input.taskIds);
            return { __typename: "CheckTaskStatusesResult", statuses };
        },
    },
};
