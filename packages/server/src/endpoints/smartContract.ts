import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { SmartContractSortBy, FindByIdInput, SmartContract, SmartContractSearchInput, SmartContractCreateInput, SmartContractUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum SmartContractSortBy {
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IssuesAsc
        IssuesDesc
        PullRequestsAsc
        PullRequestsDesc
        QuestionsAsc
        QuestionsDesc
        StarsAsc
        StarsDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
        VotesAsc
        VotesDesc
    }

    input SmartContractCreateInput {
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
    input SmartContractUpdateInput {
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
    type SmartContract {
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

    type SmartContractPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input SmartContractTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input SmartContractTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type SmartContractTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input SmartContractSearchInput {
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

    type SmartContractSearchResult {
        pageInfo: PageInfo!
        edges: [SmartContractEdge!]!
    }

    type SmartContractEdge {
        cursor: String!
        node: SmartContract!
    }

    extend type Query {
        smartContract(input: FindByIdInput!): SmartContract
        smartContracts(input: SmartContractSearchInput!): SmartContractSearchResult!
    }

    extend type Mutation {
        smartContractCreate(input: SmartContractCreateInput!): SmartContract!
        smartContractUpdate(input: SmartContractUpdateInput!): Api!
    }
`

const objectType = 'SmartContract';
export const resolvers: {
    SmartContractSortBy: typeof SmartContractSortBy;
    Query: {
        smartContract: GQLEndpoint<FindByIdInput, FindOneResult<SmartContract>>;
        smartContracts: GQLEndpoint<SmartContractSearchInput, FindManyResult<SmartContract>>;
    },
    Mutation: {
        smartContractCreate: GQLEndpoint<SmartContractCreateInput, CreateOneResult<SmartContract>>;
        smartContractUpdate: GQLEndpoint<SmartContractUpdateInput, UpdateOneResult<SmartContract>>;
    }
} = {
    SmartContractSortBy,
    Query: {
        smartContract: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        smartContracts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        smartContractCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        smartContractUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}