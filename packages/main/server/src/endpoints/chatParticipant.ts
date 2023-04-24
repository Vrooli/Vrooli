import { FindByIdInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
    enum ChatParticipantSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    input ChatParticipantUpdateInput {
        id: ID!
        #permissions: String
    }
    type ChatParticipant {
        id: ID!
        created_at: Date!
        updated_at: Date!
        #permissions: String!
        chat: Chat!
        user: User!
    }

    input ChatParticipantSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        chatId: ID
        userId: ID
        searchString: String
        sortBy: ChatParticipantSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ChatParticipantSearchResult {
        pageInfo: PageInfo!
        edges: [ChatParticipantEdge!]!
    }

    type ChatParticipantEdge {
        cursor: String!
        node: ChatParticipant!
    }

    extend type Query {
        chatParticipant(input: FindByIdInput!): ChatParticipant
        chatParticipants(input: ChatParticipantSearchInput!): ChatParticipantSearchResult!
    }

    extend type Mutation {
        chatParticipantUpdate(input: ChatParticipantUpdateInput!): ChatParticipant!
    }
`;

const objectType = "ChatParticipant";
export const resolvers: {
    // ChatParticipantSortBy: typeof ChatParticipantSortBy;
    Query: {
        chatParticipant: GQLEndpoint<FindByIdInput, FindOneResult<any>>;
        chatParticipants: GQLEndpoint<any, FindManyResult<any>>;
    },
    Mutation: {
        chatParticipantUpdate: GQLEndpoint<any, UpdateOneResult<any>>;
    }
} = {
    // ChatParticipantSortBy,
    Query: {
        chatParticipant: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chatParticipants: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        chatParticipantUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
