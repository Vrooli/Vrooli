import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, LabelSortBy, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, ReportResponseSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ReportResponseSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    input ReportResponseCreateInput {
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
    input ReportResponseUpdateInput {
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
    type ReportResponse {
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
        createdBy: User
        forks: [Project!]!
        owner: Owner
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

    type ReportResponsePermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input ReportResponseTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input ReportResponseTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ReportResponseTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input ReportResponseSearchInput {
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

    type ReportResponseSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type ReportResponseEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        reportResponse(input: FindByIdInput!): ReportResponse
        reportResponses(input: ReportResponseSearchInput!): ReportResponseSearchResult!
    }

    extend type Mutation {
        reportResponseCreate(input: ReportResponseCreateInput!): ReportResponse!
        reportResponseUpdate(input: ReportResponseUpdateInput!): ReportResponse!
    }
`

const objectType = 'ReportResponse';
export const resolvers: {
    ReportResponseSortBy: typeof ReportResponseSortBy;
    Query: {
        reportResponse: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        reportResponses: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        reportResponseCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        reportResponseUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    ReportResponseSortBy,
    Query: {
        reportResponse: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        reportResponses: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        reportResponseCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        reportResponseUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}