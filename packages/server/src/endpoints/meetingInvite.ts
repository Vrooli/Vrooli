import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, MeetingInviteSortBy, MeetingInviteStatus, MeetingInvite, MeetingInviteSearchInput, MeetingInviteCreateInput, MeetingInviteUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum MeetingInviteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        MemberNameAsc
        MemberNameDesc
        StatusAsc
        StatusDesc
    }

    enum MeetingInviteStatus {
        Pending
        Accepted
        Declined
    }

    input MeetingInviteCreateInput {
        id: ID!
        meetingId: ID!
        userId: ID!
        message: String
    }
    input MeetingInviteUpdateInput {
        id: ID!
        message: String
    }
    type MeetingInvite {
        id: ID!
        created_at: Date!
        updated_at: Date!
        meeting: Meeting!
        user: User!
        message: String
        status: MeetingInviteStatus!
        permissionsMeetingInvite: MeetingInvitePermission!
    }

    type MeetingInvitePermission {
        canDelete: Boolean!
        canEdit: Boolean!
    }

    input MeetingInviteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        status: MeetingInviteStatus
        meetingId: ID
        userId: ID
        organizationId: ID
        searchString: String
        sortBy: MeetingInviteSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MeetingInviteSearchResult {
        pageInfo: PageInfo!
        edges: [MeetingInviteEdge!]!
    }

    type MeetingInviteEdge {
        cursor: String!
        node: MeetingInvite!
    }

    extend type Query {
        meetingInvite(input: FindByIdInput!): MeetingInvite
        meetingInvites(input: MeetingInviteSearchInput!): MeetingInviteSearchResult!
    }

    extend type Mutation {
        meetingInviteCreate(input: MeetingInviteCreateInput!): MeetingInvite!
        meetingInviteUpdate(input: MeetingInviteUpdateInput!): MeetingInvite!
    }
`

const objectType = 'MeetingInvite';
export const resolvers: {
    MeetingInviteSortBy: typeof MeetingInviteSortBy;
    MeetingInviteStatus: typeof MeetingInviteStatus;
    Query: {
        meetingInvite: GQLEndpoint<FindByIdInput, FindOneResult<MeetingInvite>>;
        meetingInvites: GQLEndpoint<MeetingInviteSearchInput, FindManyResult<MeetingInvite>>;
    },
    Mutation: {
        meetingInviteCreate: GQLEndpoint<MeetingInviteCreateInput, CreateOneResult<MeetingInvite>>;
        meetingInviteUpdate: GQLEndpoint<MeetingInviteUpdateInput, UpdateOneResult<MeetingInvite>>;
    }
} = {
    MeetingInviteSortBy,
    MeetingInviteStatus,
    Query: {
        meetingInvite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        meetingInvites: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        meetingInviteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        meetingInviteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}