import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, MeetingSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput } from './types';
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
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        rootId: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input MeetingUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        organizationId: ID
        userId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [ProjectTranslationCreateInput!]
        translationsUpdate: [ProjectTranslationUpdateInput!]
    }
    type Meeting {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        handle: String
        isComplete: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        forks: [Project!]!
        owner: Contributor
        parent: Project
        permissionsProject: ProjectPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        translations: [ProjectTranslation!]!
        wallets: [Wallet!]
    }

    type MeetingPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input MeetingTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input MeetingTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type MeetingTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input MeetingSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type MeetingSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type MeetingEdge {
        cursor: String!
        node: Api!
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
        meeting: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        meetings: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        meetingCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        meetingUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    MeetingSortBy: MeetingSortBy,
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