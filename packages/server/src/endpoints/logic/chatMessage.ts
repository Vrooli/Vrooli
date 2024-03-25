import { AutoFillInput, AutoFillResult, ChatMessage, ChatMessageCreateInput, ChatMessageSearchInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageUpdateInput, FindByIdInput, RegenerateResponseInput, StartTaskInput, Success, uuidValidate } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { ModelMap } from "../../models/base";
import { PreMapMessageData, PreMapUserData } from "../../models/base/chatMessage";
import { ChatMessageModelLogic } from "../../models/base/types";
import { llmProcessAutoFill, llmProcessStartTask } from "../../tasks/llm/process";
import { requestBotResponse } from "../../tasks/llm/queue";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";
import { bestTranslation } from "../../utils/bestTranslation";
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
            const { canDelete } = await getSingleTypePermissions("ChatMessage", [input.messageId], userData);
            // Use delete permissions to determine if we can regenerate a response, 
            // even though we keep the old message
            if (!canDelete) {
                throw new CustomError("0424", "Unauthorized", userData.languages, { input });
            }
            const existingMessageInfo = await prismaInstance.chat_message.findUnique({
                where: { id: input.messageId },
                select: {
                    chat: {
                        select: {
                            id: true,
                            participants: {
                                select: {
                                    id: true,
                                    user: {
                                        select: {
                                            id: true,
                                            botSettings: true,
                                            isBot: true,
                                            name: true,
                                        },
                                    },
                                },
                            },
                        },
                    },
                    parent: {
                        select: {
                            id: true,
                            parent: {
                                select: {
                                    id: true,
                                },
                            },
                            translations: {
                                select: {
                                    id: true,
                                    language: true,
                                    text: true,
                                },
                            },
                            user: {
                                select: {
                                    id: true,
                                    isBot: true,
                                },
                            },
                        },
                    },
                    user: {
                        select: {
                            id: true,
                            isBot: true,
                        },
                    },
                },
            });
            if (
                !existingMessageInfo ||
                !existingMessageInfo.chat ||
                !existingMessageInfo.chat.participants ||
                !existingMessageInfo.user
            ) {
                throw new CustomError("0426", "InternalError", userData.languages, { existingMessageInfo });
            }
            const chatId = existingMessageInfo.chat.id;
            const botId = existingMessageInfo.user.isBot ? existingMessageInfo.user.id : null;
            const translation = existingMessageInfo.parent ?
                bestTranslation(existingMessageInfo.parent.translations, userData.languages) :
                undefined;
            if (!chatId || !botId) {
                throw new CustomError("0427", "InternalError", userData.languages, { chatId, botId, input });
            }
            const parent = existingMessageInfo.parent;
            const previousMessage: PreMapMessageData | null = parent && translation ? {
                chatId,
                content: translation.text,
                id: parent.id,
                isNew: false,
                language: translation.language,
                parentId: parent.parent?.id,
                translations: parent.translations,
                userId: parent.user?.id ?? "",
            } : null;
            // Map of participants by Id
            const participants: Record<string, PreMapUserData> = {};
            for (const participant of existingMessageInfo.chat.participants) {
                if (participant.user) {
                    participants[participant.user.id] = {
                        botSettings: participant.user.botSettings ?? JSON.stringify({}),
                        id: participant.user.id,
                        isBot: participant.user.isBot,
                        name: participant.user.name,
                    };
                }
            }
            // Call LLM for bot response
            requestBotResponse({
                chatId,
                parent: previousMessage,
                respondingBotId: botId,
                task: "Start", //TODO
                participantsData: participants,
                userData,
            });
            return { __typename: "Success", success: true };
        },
        autoFill: async (_, { input }, { req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return llmProcessAutoFill({ ...input, userData, __process: "AutoFill" });
        },
        startTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return llmProcessStartTask({ ...input, userData, __process: "StartTask" });
        },
    },
};
