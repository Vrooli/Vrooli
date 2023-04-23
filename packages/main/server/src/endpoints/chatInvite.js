import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { CustomError } from "../events";
import { rateLimit } from "../middleware";
export const typeDef = gql `
    enum ChatInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    enum ChatInviteStatus {
        Pending
        Accepted
        Declined
    }

    input ChatInviteCreateInput {
        id: ID!
        userConnect: ID!
        chatConnect: ID!
        message: String
        #willHavePermissions: String
    }
    input ChatInviteUpdateInput {
        id: ID!
        message: String
        #willHavePermissions: String
    }
    type ChatInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        user: User!
        chat: Chat!
        message: String
        status: ChatInviteStatus!
        #willHavePermissions: String
        you: ChatInviteYou!
    }

    type ChatInviteYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input ChatInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: ChatInviteStatus
        chatId: ID
        userId: ID
        searchString: String
        sortBy: ChatInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ChatInviteSearchResult {
        pageInfo: PageInfo!
        edges: [ChatInviteEdge!]!
    }

    type ChatInviteEdge {
        cursor: String!
        node: ChatInvite!
    }

    extend type Query {
        chatInvite(input: FindByIdInput!): ChatInvite
        chatInvites(input: ChatInviteSearchInput!): ChatInviteSearchResult!
    }

    extend type Mutation {
        chatInviteCreate(input: ChatInviteCreateInput!): ChatInvite!
        chatInviteUpdate(input: ChatInviteUpdateInput!): ChatInvite!
        chatInviteAccept(input: FindByIdInput!): ChatInvite!
        chatInviteDecline(input: FindByIdInput!): ChatInvite!
    }
`;
const objectType = "ChatInvite";
export const resolvers = {
    Query: {
        chatInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        chatInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        chatInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        chatInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        chatInviteAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        chatInviteDecline: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
//# sourceMappingURL=chatInvite.js.map