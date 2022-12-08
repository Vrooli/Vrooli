import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { SmartContractVersionSortBy, FindByIdInput, SmartContractVersionSearchInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum SmartContractVersionSortBy {
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

    input SmartContractVersionCreateInput {
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
    input SmartContractVersionUpdateInput {
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
    type SmartContractVersion {
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

    type SmartContractVersionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input SmartContractVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input SmartContractVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type SmartContractVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input SmartContractVersionSearchInput {
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

    type SmartContractVersionSearchResult {
        pageInfo: PageInfo!
        edges: [SmartContractEdge!]!
    }

    type SmartContractVersionEdge {
        cursor: String!
        node: SmartContract!
    }

    extend type Query {
        smartContractVersion(input: FindByIdInput!): SmartContractVersion
        smartContractVersions(input: SmartContractVersionSearchInput!): SmartContractVersionSearchResult!
    }

    extend type Mutation {
        smartContractVersionCreate(input: SmartContractVersionCreateInput!): SmartContractVersion!
        smartContractVersionUpdate(input: SmartContractVersionUpdateInput!): SmartContractVersion!
    }
`

const objectType = 'SmartContractVersion';
export const resolvers: {
    SmartContractVersionSortBy: typeof SmartContractVersionSortBy;
    Query: {
        smartContractVersion: GQLEndpoint<FindByIdInput, FindOneResult<SmartContractVersion>>;
        smartContractVersions: GQLEndpoint<SmartContractVersionSearchInput, FindManyResult<SmartContractVersion>>;
    },
    Mutation: {
        smartContractVersionCreate: GQLEndpoint<SmartContractVersionCreateInput, CreateOneResult<SmartContractVersion>>;
        smartContractVersionUpdate: GQLEndpoint<SmartContractVersionUpdateInput, UpdateOneResult<SmartContractVersion>>;
    }
} = {
    SmartContractVersionSortBy,
    Query: {
        smartContractVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        smartContractVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        smartContractVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        smartContractVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}