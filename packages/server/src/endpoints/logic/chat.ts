import { Chat, ChatCreateInput, ChatSearchInput, ChatsGrouped, ChatsGroupedSearchInput, ChatUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsChat = {
    Query: {
        chat: GQLEndpoint<FindByIdInput, FindOneResult<Chat>>;
        chats: GQLEndpoint<ChatSearchInput, FindManyResult<Chat>>;
        chatsGrouped: GQLEndpoint<ChatsGroupedSearchInput, FindManyResult<ChatsGrouped>>;
    },
    Mutation: {
        chatCreate: GQLEndpoint<ChatCreateInput, CreateOneResult<Chat>>;
        chatUpdate: GQLEndpoint<ChatUpdateInput, UpdateOneResult<Chat>>;
    }
}

const objectType = "Chat";
export const ChatEndpoints: EndpointsChat = {
    Query: {
        chat: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chats: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
        chatsGrouped: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 5000, req });
            // Find chats with only 2 participants, where you are one and the chats 
            // are grouped by the other participant
            // const groupedChats = await prisma.chat_participants.groupBy({
            //     by: ["chatId", "userId"],
            //     where: {
            //         userId: userData.id,
            //     },
            //     _count: {
            //         chatId: true,
            //     },
            //     having: {
            //         chatId: {
            //             _count: {
            //                 lte: 2,
            //             },
            //         },
            //     },
            //     orderBy: {
            //         _count: {
            //             chatId: "desc",
            //         },
            //     },
            // });
            // const groupedChats = await prisma.chat_participants.findMany({
            //     where: {
            //         chat: {
            //             participants: {
            //                 some: {
            //                     userId: userData.id,
            //                 },
            //             },
            //         },
            //     },
            //     having: {
            //         chat: {
            //             participants: {
            //                 _count: {
            //                     lte: 2,
            //                 },
            //             },
            //         },
            //     },
            //     select: {
            //         chatId: true,
            //         userId: true,
            //     },
            //     orderBy: {
            //         chatId: "desc",
            //     },
            // });
            const groupedParticipants = await prisma.chat_participants.groupBy({
                by: ["userId"],
                where: {
                    AND: [
                        { userId: { not: userData.id } },  // Exclude the current user from the groupBy results
                        {
                            chat: {
                                participantsCount: 2,  // Only consider chats with two participants
                                participants: {
                                    some: {
                                        userId: userData.id,  // Ensure the current user is one of the participants
                                    },
                                },
                            },
                        },
                    ],
                },
                _count: {
                    userId: true,  // Count the number of chats for each user
                },
                orderBy: {
                    _count: {
                        userId: "desc",  // Optional: Order by the number of chats with each user (from most to least)
                    },
                },
                skip: 0,  // for pagination
                take: 20,   // for pagination
            });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
    Mutation: {
        chatCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        chatUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
