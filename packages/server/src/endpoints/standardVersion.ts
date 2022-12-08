import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { StandardVersionSortBy, StandardVersion, StandardVersionSearchInput, StandardVersionCreateInput, StandardVersionUpdateInput, FindByIdInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum StandardVersionSortBy {
        CommentsAsc
        CommentsDesc
        DirectoryListingsAsc
        DirectoryListingsDesc
        ForksAsc
        ForksDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ReportsAsc
        ReportsDesc
    }

    input StandardVersionCreateInput {
        id: ID!
        default: String
        isInternal: Boolean
        isPrivate: Boolean
        name: String
        type: String!
        props: String!
        yup: String
        versionLabel: String
        createdByUserId: ID
        createdByOrganizationId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [StandardTranslationCreateInput!]
    }
    input StandardVersionUpdateInput {
        id: ID!
        makeAnonymous: Boolean
        isPrivate: Boolean
        userId: ID
        default: String
        type: String!
        props: String!
        yup: String
        organizationId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [StandardTranslationCreateInput!]
        translationsUpdate: [StandardTranslationUpdateInput!]
        versionId: ID # If versionId passed, then we're updating an existing version. NOTE: This will throw an error if you try to update a completed version
        versionLabel: String # If version label passed, then we're creating a new version
    }
    type StandardVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        default: String
        name: String!
        isDeleted: Boolean!
        isInternal: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        type: String!
        props: String!
        yup: String
        versionLabel: String!
        rootId: ID!
        # versions: [Version!]!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        permissionsStandard: StandardPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        routineInputs: [Routine!]!
        routineOutputs: [Routine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [StandardTranslation!]!
    }

    type StandardVersionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input StandardVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        jsonVariable: String
    }
    input StandardVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        jsonVariable: String
    }
    type StandardVersionTranslation {
        id: ID!
        language: String!
        description: String
        jsonVariable: String
    }

    input StandardVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        projectId: ID
        reportId: ID
        routineId: ID
        searchString: String
        sortBy: StandardVersionSortBy
        tags: [String!]
        take: Int
        type: String
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type StandardVersionSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }

    type StandardVersionEdge {
        cursor: String!
        node: StandardVersion!
    }

    extend type Query {
        standardVersion(input: FindByIdInput!): StandardVersion
        standardVersions(input: StandardVersionSearchInput!): StandardVersionSearchResult!
    }

    extend type Mutation {
        standardVersionCreate(input: StandardVersionCreateInput!): StandardVersion!
        standardVersionUpdate(input: StandardVersionUpdateInput!): StandardVersion!
    }
`

const objectType = 'StandardVersion';
export const resolvers: {
    StandardVersionSortBy: typeof StandardVersionSortBy;
    Query: {
        standardVersion: GQLEndpoint<FindByIdInput, FindOneResult<StandardVersion>>;
        standardVersions: GQLEndpoint<StandardVersionSearchInput, FindManyResult<StandardVersion>>;
    },
    Mutation: {
        standardVersionCreate: GQLEndpoint<StandardVersionCreateInput, CreateOneResult<StandardVersion>>;
        standardVersionUpdate: GQLEndpoint<StandardVersionUpdateInput, UpdateOneResult<StandardVersion>>;
    }
} = {
    StandardVersionSortBy,
    Query: {
        standardVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standardVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        standardVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}