import { FindByIdInput, Meeting, MeetingCreateInput, MeetingSearchInput, MeetingSortBy, MeetingUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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
        organizationConnect: ID!
        restrictedToRolesConnect: [ID!]
        invitesCreate: [MeetingInviteCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        scheduleCreate: ScheduleCreateInput
        translationsCreate: [MeetingTranslationCreateInput!]
    }
    input MeetingUpdateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        showOnOrganizationProfile: Boolean
        restrictedToRolesConnect: [ID!]
        restrictedToRolesDisconnect: [ID!]
        invitesCreate: [MeetingInviteCreateInput!]
        invitesUpdate: [MeetingInviteUpdateInput!]
        invitesDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
        translationsCreate: [MeetingTranslationCreateInput!]
        translationsUpdate: [MeetingTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Meeting {
        id: ID!
        openToAnyoneWithInvite: Boolean!
        showOnOrganizationProfile: Boolean!
        organization: Organization!
        restrictedToRoles: [Role!]!
        attendees: [User!]!
        attendeesCount: Int!
        invites: [MeetingInvite!]!
        invitesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        schedule: Schedule
        translations: [MeetingTranslation!]!
        translationsCount: Int!
        you: MeetingYou!
    }

    type MeetingYou {
        canDelete: Boolean!
        canInvite: Boolean!
        canUpdate: Boolean!
    }

    input MeetingTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        link: String
        name: String
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
        maxEventStart: Date
        maxEventEnd: Date
        maxRecurrStart: Date
        maxRecurrEnd: Date
        minEventStart: Date
        minEventEnd: Date
        minRecurrStart: Date
        minRecurrEnd: Date
        labelsIds: [ID!]
        organizationId: ID
        searchString: String
        sortBy: MeetingSortBy
        take: Int
        translationLanguages: [String!]
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