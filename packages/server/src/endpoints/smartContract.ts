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
        ScoreAsc
        ScoreDesc
        StarsAsc
        StarsDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    input SmartContractCreateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String!
        parentConnect: ID
        userConnect: ID
        organizationConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [SmartContractVersionCreateInput!]
    }
    input SmartContractUpdateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        userConnect: ID
        organizationConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [SmartContractVersionCreateInput!]
        versionsUpdate: [SmartContractVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type SmartContract {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompletedVersion: Boolean!
        isDeleted: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        translatedName: String!
        score: Int!
        stars: Int!
        views: Int!
        createdBy: User
        forks: [SmartContract!]!
        forksCount: Int!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: SmartContract
        permissionsRoot: RootPermission!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        starredBy: [User!]!
        tags: [Tag!]!
        versions: [SmartContractVersion!]!
        versionsCount: Int
    }

    input SmartContractSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
        labelsId: ID
        maxScore: Int
        maxStars: Int
        maxViews: Int
        minScore: Int
        minStars: Int
        minViews: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        searchString: String
        sortBy: SmartContractSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
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