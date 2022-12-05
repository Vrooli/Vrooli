import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ApiVersionSortBy, ApiVersion, ApiVersionSearchInput, ApiVersionCreateInput, FindByIdInput, ApiVersionUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ApiVersionSortBy {
        CommentsAsc
        CommentsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
        VotesAsc
        VotesDesc
    }

    input ApiVersionCreateInput {
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
    input ApiVersionUpdateInput {
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
    type ApiVersion {
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

    type ApiVersionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input ApiVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input ApiVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ApiVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input ApiVersionSearchInput {
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

    type ApiVersionSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type ApiVersionEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        apiVersion(input: FindByIdInput!): ApiVersion
        apiVersions(input: ApiVersionSearchInput!): ApiVersionSearchResult!
    }

    extend type Mutation {
        apiVersionCreate(input: ApiVersionCreateInput!): ApiVersion!
        apiVersionUpdate(input: ApiVersionUpdateInput!): ApiVersion!
    }
`

const objectType = 'ApiVersion';
export const resolvers: {
    ApiVersionSortBy: typeof ApiVersionSortBy;
    Query: {
        apiVersion: GQLEndpoint<FindByIdInput, FindOneResult<ApiVersion>>;
        apiVersions: GQLEndpoint<ApiVersionSearchInput, FindManyResult<ApiVersion>>;
    },
    Mutation: {
        apiVersionCreate: GQLEndpoint<ApiVersionCreateInput, CreateOneResult<ApiVersion>>;
        apiVersionUpdate: GQLEndpoint<ApiVersionUpdateInput, UpdateOneResult<ApiVersion>>;
    }
} = {
    ApiVersionSortBy,
    Query: {
        apiVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        apiVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        apiVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        apiVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}