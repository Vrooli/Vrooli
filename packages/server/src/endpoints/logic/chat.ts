import { Chat, ChatCreateInput, ChatSearchInput, ChatsGrouped, ChatsGroupedSearchInput, ChatUpdateInput, FindByIdInput } from "@local/shared";
import { createHelper, getDesiredTake, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { combineQueries, selectHelper, toPartialGqlInfo } from "../../builders";
import { CustomError, logger } from "../../events";
import { getSearchStringQuery } from "../../getters";
import { rateLimit } from "../../middleware";
import { ChatModel } from "../../models/base";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";
import { SearchMap, SortMap } from "../../utils";

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
            // Partially convert info type
            const partialInfo = toPartialGqlInfo(info, ChatModel.format.gqlRelMap, req.session.languages, true);
            const searchQuery = input.searchString ? getSearchStringQuery({ objectType: ChatModel.__typename, searchString: input.searchString }) : undefined;
            // Loop through search fields and add each to the search query, 
            // if the field is specified in the input
            const customQueries: { [x: string]: unknown }[] = [];
            for (const field of Object.keys(ChatModel.search.searchFields)) {
                if (input[field as string] !== undefined) {
                    customQueries.push(SearchMap[field as string](input[field], userData, ChatModel.__typename));
                }
            }
            if (ChatModel.search?.customQueryData) {
                customQueries.push(ChatModel.search.customQueryData(input, userData));
            }
            // Combine queries
            const where = combineQueries([searchQuery, ...customQueries, {
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
            }]);
            // Determine sortBy, orderBy, and take
            const sortBy = input.sortBy ?? ChatModel.search.defaultSort;
            const orderBy = sortBy in SortMap ? SortMap[sortBy] : undefined;
            const desiredTake = getDesiredTake(input.take, req.session.languages, objectType);
            // Find requested search array
            const select = selectHelper(partialInfo);
            // Search results have at least an id
            let searchResults: Record<string, any>[] = [];
            try {
                searchResults = await prisma.chat_participants.groupBy({
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
                    //... paginated search-related args
                }) as any;
            } catch (error) {
                logger.error("chatsGrouped: Failed to find searchResults", { trace: "0392", error, objectType, ...select, where, orderBy });
                throw new CustomError("0392", "InternalError", req.session.languages, { objectType });
            }
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
