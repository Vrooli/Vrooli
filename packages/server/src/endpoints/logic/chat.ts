import { Chat, ChatCreateInput, ChatSearchInput, ChatsGrouped, ChatsGroupedSearchInput, ChatUpdateInput, FindByIdInput, isObject, User } from "@local/shared";
import { createHelper, getDesiredTake, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { addSupplementalFields, combineQueries, modelToGql, selectHelper, toPartialGqlInfo } from "../../builders";
import { PartialGraphQLInfo } from "../../builders/types";
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
            // Find requested user data
            const partialInfo = toPartialGqlInfo(info, {
                __typename: "ChatsGrouped",
                user: "User",
            }, req.session.languages, true);
            const userSelect = isObject(partialInfo.user) ? selectHelper(partialInfo.user) : {};
            // Query for search results
            let searchResults: ChatsGrouped[] = [];
            try {
                // Perform groupBy query
                const participants = await prisma.chat_participants.groupBy({
                    by: ["userId"],
                    where,
                    // orderBy, //TODO switch orderBy to chat participant
                    orderBy: {
                        userId: "asc",
                    },
                    take: desiredTake + 1,  // Take one more than desiredTake to determine if there is a next page
                    skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
                    _count: {
                        userId: true,  // Count the number of chats for each user
                    },
                    //... paginated search-related args
                });
                // Now query user data for each userId
                const userIds = participants.map((p) => p.userId);
                const users = await prisma.user.findMany({
                    where: { id: { in: userIds } },
                    ...userSelect,
                });
                // Combine the two queries
                searchResults = participants.map((p) => {
                    const chatsCount = p._count?.userId ?? 0;
                    const user: User | undefined = users.find((u) => u.id === p.userId) as User | undefined;
                    if (!user) return null;
                    return {
                        __typename: "ChatsGrouped" as const,
                        id: user.id,
                        chatsCount,
                        user,
                    };
                }).filter((n) => n !== null) as ChatsGrouped[];
            } catch (error) {
                logger.error("chatsGrouped: Failed to find searchResults", { trace: "0392", error, objectType, userSelect, where, orderBy });
                throw new CustomError("0392", "InternalError", req.session.languages, { objectType });
            }
            let hasNextPage = false;
            let endCursor: string | null = null;
            // Check if there's an extra item (indicating more results)
            if (searchResults.length > desiredTake) {
                hasNextPage = true;
                searchResults.pop(); // remove the extra item
            }
            if (searchResults.length > 0) {
                endCursor = searchResults[searchResults.length - 1].id;
            }
            //TODO validate that the user has permission to read all of the results, including relationships
            // Add supplemental fields, if requested
            searchResults = searchResults.map(n => modelToGql(n, partialInfo as PartialGraphQLInfo));
            searchResults = await addSupplementalFields(prisma, userData, searchResults, partialInfo) as ChatsGrouped[];
            // Return formatted for GraphQL
            return {
                __typename: "ChatsGroupedSearchResult" as const,
                pageInfo: {
                    __typename: "PageInfo" as const,
                    hasNextPage,
                    endCursor,
                },
                edges: searchResults.map((result) => ({
                    __typename: "ChatsGroupedEdge" as const,
                    cursor: result.id,
                    node: result,
                })),
            };
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
