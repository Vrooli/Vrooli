import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { SmartContractVersionSortBy, SmartContractVersionSearchInput, SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput, FindVersionInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum SmartContractVersionSortBy {
        CalledByRoutinesAsc
        CalledByRoutinesDesc
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
        isComplete: Boolean
        isLatest: Boolean
        isPrivate: Boolean
        default: String
        contractType: String!
        content: String!
        rootConnect: ID!
        rootCreate: SmartContractCreateInput
        versionLabel: String!
        versionNotes: String
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        translationsCreate: [SmartContractVersionTranslationCreateInput!]
    }
    input SmartContractVersionUpdateInput {
        id: ID!
        isComplete: Boolean
        isLatest: Boolean
        isPrivate: Boolean
        default: String
        contractType: String
        content: String
        versionLabel: String
        versionNotes: String
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rootUpdate: SmartContractUpdateInput
        translationsCreate: [SmartContractVersionTranslationCreateInput!]
        translationsUpdate: [SmartContractVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type SmartContractVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        completedAt: Date
        isComplete: Boolean!
        isDeleted: Boolean!
        isLatest: Boolean!
        isPrivate: Boolean!
        default: String
        contractType: String!
        content: String!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        forks: [SmartContract!]!
        forksCount: Int!
        pullRequest: PullRequest
        resourceList: ResourceList
        reports: [Report!]!
        reportsCount: Int!
        root: SmartContract!
        translations: [SmartContractVersionTranslation!]!
        translationsCount: Int!
        you: VersionYou!
    }

    input SmartContractVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }
    input SmartContractVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
        jsonVariable: String
    }
    type SmartContractVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }

    input SmartContractVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        languages: [String!]
        reportId: ID
        rootId: ID
        searchString: String
        sortBy: SmartContractVersionSortBy
        contractType: String
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type SmartContractVersionSearchResult {
        pageInfo: PageInfo!
        edges: [SmartContractVersionEdge!]!
    }

    type SmartContractVersionEdge {
        cursor: String!
        node: SmartContractVersion!
    }

    extend type Query {
        smartContractVersion(input: FindVersionInput!): SmartContractVersion
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
        smartContractVersion: GQLEndpoint<FindVersionInput, FindOneResult<SmartContractVersion>>;
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