import { MeetingSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsMeeting, MeetingEndpoints } from "../logic/meeting";

export const typeDef = gql`
    enum MeetingSortBy {
        AttendeesAsc
        AttendeesDesc
        InvitesAsc
        InvitesDesc
    }

    input MeetingCreateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        showOnTeamProfile: Boolean
        invitesCreate: [MeetingInviteCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        restrictedToRolesConnect: [ID!]
        scheduleCreate: ScheduleCreateInput
        teamConnect: ID!
        translationsCreate: [MeetingTranslationCreateInput!]
    }
    input MeetingUpdateInput {
        id: ID!
        openToAnyoneWithInvite: Boolean
        showOnTeamProfile: Boolean
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
        created_at: Date!
        updated_at: Date!
        openToAnyoneWithInvite: Boolean!
        showOnTeamProfile: Boolean!
        team: Team!
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
        language: String!
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
        scheduleStartTimeFrame: TimeFrame
        scheduleEndTimeFrame: TimeFrame
        showOnTeamProfile: Boolean
        labelsIds: [ID!]
        searchString: String
        sortBy: MeetingSortBy
        take: Int
        teamId: ID
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
`;

export const resolvers: {
    MeetingSortBy: typeof MeetingSortBy;
    Query: EndpointsMeeting["Query"];
    Mutation: EndpointsMeeting["Mutation"];
} = {
    MeetingSortBy,
    ...MeetingEndpoints,
};
