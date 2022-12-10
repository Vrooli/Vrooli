import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, MeetingSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, Meeting, MeetingSearchInput, MeetingCreateInput, MeetingUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum MeetingSortBy {
        EventStartAsc
        EventStartDesc
        EventEndAsc
        EventEndDesc
        RecurrStartAsc
        RecurrStartDesc
        RecurrEndAsc
        RecurrEndDesc
        AttendeesAsc
        AttendeesDesc
        InvitesAsc
        InvitesDesc
    }

    input MeetingCreateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        showOnOrganizationProfile: Boolean
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        organizationId: ID!
        restrictedToRolesConnect: [ID!]
        invitedUsersConnect: [ID!]
        labelsConnect: [ID!]
        translationsCreate: [MeetingTranslationCreateInput!]
    }
    input MeetingUpdateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        showOnOrganizationProfile: Boolean
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean
        recurrStart: Date
        recurrEnd: Date
        restrictedToRolesConnect: [ID!]
        restrictedToRolesDisconnect: [ID!]
        invitedUsersConnect: [ID!]
        invitedUsersDisconnect: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        translationsCreate: [MeetingTranslationCreateInput!]
        translationsUpdate: [MeetingTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Meeting {
        id: ID!
        openToAnyoneWithInvite: Boolean!
        showOnOrganizationProfile: Boolean!
        timeZone: String
        eventStart: Date
        eventEnd: Date
        recurring: Boolean!
        recurrStart: Date
        recurrEnd: Date
        organization: Organization!
        permissionsMeeting: MeetingPermission!
        restrictedToRoles: [Role!]!
        attendees: [User!]!
        attendeesCount: Int!
        invites: [MeetingInvite!]!
        invitesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        translations: [MeetingTranslation!]!
        translationsCount: Int!
    }

    type MeetingPermission {
        canDelete: Boolean!
        canEdit: Boolean!
        canInvite: Boolean!
    }

    input MeetingTranslationCreateInput {
        id: ID!
        language: String!
        name: String
        description: String
        link: String
    }
    input MeetingTranslationUpdateInput {
        id: ID!
        language: String
        name: String
        description: String
        link: String
    }
    type MeetingTranslation {
        id: ID!
        language: String!
        name: String
        description: String
        link: String
    }

    input MeetingSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        openToAnyoneWithInvite: Boolean
        showOnOrganizationProfile: Boolean
        labels: [ID!]
        languages: [String!]
        organizationId: ID
        searchString: String
        sortBy: MeetingSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type MeetingSearchResult {
        pageInfo: PageInfo!
        edges: [MeetingEdge!]!
    }

    type MeetingEdge {
        cursor: String!
        node: Meeting!
    }

    extend type Query {
        meeting(input: FindByIdInput!): Meeting
        meetings(input: MeetingSearchInput!): MeetingSearchResult!
    }

    extend type Mutation {
        meetingCreate(input: MeetingCreateInput!): Meeting!
        meetingUpdate(input: MeetingUpdateInput!): Meeting!
    }
`

const objectType = 'Meeting';
export const resolvers: {
    MeetingSortBy: typeof MeetingSortBy;
    Query: {
        meeting: GQLEndpoint<FindByIdInput, FindOneResult<Meeting>>;
        meetings: GQLEndpoint<MeetingSearchInput, FindManyResult<Meeting>>;
    },
    Mutation: {
        meetingCreate: GQLEndpoint<MeetingCreateInput, CreateOneResult<Meeting>>;
        meetingUpdate: GQLEndpoint<MeetingUpdateInput, UpdateOneResult<Meeting>>;
    }
} = {
    MeetingSortBy,
    Query: {
        meeting: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        meetings: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        meetingCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        meetingUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}