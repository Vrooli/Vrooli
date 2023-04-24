import { FindByIdInput, MemberInvite, MemberInviteCreateInput, MemberInviteSearchInput, MemberInviteSortBy, MemberInviteStatus, MemberInviteUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { CustomError } from "../events";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
    enum MemberInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        UserNameAsc
        UserNameDesc
    }

    enum MemberInviteStatus {
        Pending
        Accepted
        Declined
    }

    input MemberInviteCreateInput {
        id: ID!
        userConnect: ID!
        organizationConnect: ID!
        message: String
        willBeAdmin: Boolean
        willHavePermissions: String
    }
    input MemberInviteUpdateInput {
        id: ID!
        message: String
        willBeAdmin: Boolean
        willHavePermissions: String
    }
    type MemberInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        user: User!
        organization: Organization!
        message: String
        status: MemberInviteStatus!
        willBeAdmin: Boolean!
        willHavePermissions: String
        you: MemberInviteYou!
    }

    type MemberInviteYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input MemberInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: MemberInviteStatus
        organizationId: ID
        userId: ID
        searchString: String
        sortBy: MemberInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MemberInviteSearchResult {
        pageInfo: PageInfo!
        edges: [MemberInviteEdge!]!
    }

    type MemberInviteEdge {
        cursor: String!
        node: MemberInvite!
    }

    extend type Query {
        memberInvite(input: FindByIdInput!): MemberInvite
        memberInvites(input: MemberInviteSearchInput!): MemberInviteSearchResult!
    }

    extend type Mutation {
        memberInviteCreate(input: MemberInviteCreateInput!): MemberInvite!
        memberInviteUpdate(input: MemberInviteUpdateInput!): MemberInvite!
        memberInviteAccept(input: FindByIdInput!): MemberInvite!
        memberInviteDecline(input: FindByIdInput!): MemberInvite!
    }
`;

const objectType = "MemberInvite";
export const resolvers: {
    MemberInviteSortBy: typeof MemberInviteSortBy;
    MemberInviteStatus: typeof MemberInviteStatus;
    Query: {
        memberInvite: GQLEndpoint<FindByIdInput, FindOneResult<MemberInvite>>;
        memberInvites: GQLEndpoint<MemberInviteSearchInput, FindManyResult<MemberInvite>>;
    },
    Mutation: {
        memberInviteCreate: GQLEndpoint<MemberInviteCreateInput, CreateOneResult<MemberInvite>>;
        memberInviteUpdate: GQLEndpoint<MemberInviteUpdateInput, UpdateOneResult<MemberInvite>>;
        memberInviteAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
        memberInviteDecline: GQLEndpoint<FindByIdInput, UpdateOneResult<MemberInvite>>;
    }
} = {
    MemberInviteSortBy,
    MemberInviteStatus,
    Query: {
        memberInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        memberInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        memberInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        memberInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        memberInviteAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
        memberInviteDecline: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            throw new CustomError("0000", "NotImplemented", ["en"]);
        },
    },
};
