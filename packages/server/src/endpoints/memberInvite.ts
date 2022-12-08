import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, MeetingInviteSortBy, MeetingInviteStatus, MeetingInvite, MeetingInviteSearchInput, MeetingInviteCreateInput, MeetingInviteUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum MemberInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        MemberNameAsc
        MemberNameDesc
        StatusAsc
        StatusDesc
    }

    enum MemberInviteStatus {
        Pending
        Accepted
        Declined
    }

    input MemberInviteCreateInput {
        id: ID!
        memberId: ID!
        organizationId: ID!
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
        member: Member!
        organization: Organization!
        message: String
        status: MemberInviteStatus!
        willBeAdmin: Boolean!
        willHavePermissions: String
        permissionsMemberInvite: MemberInvitePermission!
    }

    type MemberInvitePermission {
        canDelete: Boolean!
        canEdit: Boolean!
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
    }
`

const objectType = 'MemberInvite';
export const resolvers: {
    MemberInviteSortBy: typeof MeetingInviteSortBy;
    MemberInviteStatus: typeof MeetingInviteStatus;
    Query: {
        memberInvite: GQLEndpoint<FindByIdInput, FindOneResult<MeetingInvite>>;
        memberInvites: GQLEndpoint<MeetingInviteSearchInput, FindManyResult<MeetingInvite>>;
    },
    Mutation: {
        memberInviteCreate: GQLEndpoint<MeetingInviteCreateInput, CreateOneResult<MeetingInvite>>;
        memberInviteUpdate: GQLEndpoint<MeetingInviteUpdateInput, UpdateOneResult<MeetingInvite>>;
    }
} = {
    MemberInviteSortBy: MeetingInviteSortBy,
    MemberInviteStatus: MeetingInviteStatus,
    Query: {
        memberInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        memberInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        memberInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        memberInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}