import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { RoutineVersionSortBy, FindByIdInput, RoutineVersionSearchInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum RoutineVersionSortBy {
        CommentsAsc
        CommentsDesc
        ComplexityAsc
        ComplexityDesc
        DirectoryListingsAsc
        DirectoryListingsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        RunRoutinesAsc
        RunRoutinesDesc
        SimplicityAsc
        SimplicityDesc
    }

    input RoutineVersionCreateInput {
        id: ID!
        apiCallData: String
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isLatest: Boolean
        isPrivate: Boolean
        rootConnect: ID!
        rootCreate: RoutineCreateInput
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        smartContractCallData: String
        apiVersionConnect: ID
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        smartContractVersionConnect: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        suggestedNextByRoutineVersionConnect: [ID!]
        translationsCreate: [RoutineVersionTranslationCreateInput!]
    }
    input RoutineVersionUpdateInput {
        id: ID!
        apiCallData: String
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isLatest: Boolean
        isPrivate: Boolean
        versionIndex: Int!
        versionLabel: String
        versionNotes: String
        smartContractCallData: String
        apiVersionConnect: ID
        apiVersionDisconnect: ID
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        smartContractVersionConnect: ID
        smartContractVersionDisconnect: ID
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodesDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        nodeLinksDelete: [ID!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        inputsUpdate: [RoutineVersionInputUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        outputsUpdate: [RoutineVersionOutputUpdateInput!]
        outputsDelete: [ID!]
        suggestedNextByRoutineVersionConnect: [ID!]
        suggestedNextByRoutineVersionDisconnect: [ID!]
        translationsCreate: [RoutineVersionTranslationCreateInput!]
        translationsUpdate: [RoutineVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type RoutineVersion {
        id: ID!
        completedAt: Date
        complexity: Int!
        created_at: Date!
        updated_at: Date!
        isAutomatable: Boolean
        isComplete: Boolean!
        isDeleted: Boolean!
        isInternal: Boolean
        isLatest: Boolean!
        isPrivate: Boolean!
        simplicity: Int!
        timesStarted: Int!
        timesCompleted: Int!
        smartContractCallData: String
        apiCallData: String
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        api: Api
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        forks: [Routine!]!
        forksCount: Int!
        inputs: [RoutineVersionInput!]!
        inputsCount: Int!
        nodes: [Node!]!
        nodesCount: Int!
        nodeLinks: [NodeLink!]!
        nodeLinksCount: Int!
        outputs: [RoutineVersionOutput!]!
        outputsCount: Int!
        pullRequest: PullRequest
        resourceList: ResourceList
        reports: [Report!]!
        reportsCount: Int!
        root: Routine!
        smartContract: SmartContract
        suggestedNextByRoutineVersion: [RoutineVersion!]!
        suggestedNextByRoutineVersionCount: Int!
        translations: [RoutineVersionTranslation!]!
        translationsCount: Int!
        you: RoutineVersionYou!
    }

    type RoutineVersionYou {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canFork: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canRun: Boolean!
        canView: Boolean!
        canVote: Boolean!
        runs: [RunRoutine!]!
    }

    input RoutineVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        instructions: String!
        name: String!
    }
    input RoutineVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        instructions: String
        name: String
    }
    type RoutineVersionTranslation {
        id: ID!
        language: String!
        description: String
        instructions: String!
        name: String!
    }

    input RoutineVersionSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        directoryListingsId: ID
        excludeIds: [ID!]
        ids: [ID!]
        isCompleteWithRoot: Boolean
        isCompleteWithRootExcludeOwnedByOrganizationId: ID
        isCompleteWithRootExcludeOwnedByUserId: ID
        isInternalWithRoot: Boolean
        isInternalWithRootExcludeOwnedByOrganizationId: ID
        isInternalWithRootExcludeOwnedByUserId: ID
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        maxTimesCompleted: Int
        minScoreRoot: Int
        minStarsRoot: Int
        minTimesCompleted: Int
        minViewsRoot: Int
        ownedByOrganizationId: ID
        ownedByUserId: ID
        reportId: ID
        rootId: ID
        searchString: String
        sortBy: RoutineVersionSortBy
        tags: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RoutineVersionSearchResult {
        pageInfo: PageInfo!
        edges: [RoutineVersionEdge!]!
    }

    type RoutineVersionEdge {
        cursor: String!
        node: RoutineVersion!
    }

    extend type Query {
        routineVersion(input: FindByIdInput!): RoutineVersion
        routineVersions(input: RoutineVersionSearchInput!): RoutineVersionSearchResult!
    }

    extend type Mutation {
        routineVersionCreate(input: RoutineVersionCreateInput!): RoutineVersion!
        routineVersionUpdate(input: RoutineVersionUpdateInput!): RoutineVersion!
    }
`

const objectType = 'RoutineVersion';
export const resolvers: {
    RoutineVersionSortBy: typeof RoutineVersionSortBy;
    Query: {
        routineVersion: GQLEndpoint<FindByIdInput, FindOneResult<RoutineVersion>>;
        routineVersions: GQLEndpoint<RoutineVersionSearchInput, FindManyResult<RoutineVersion>>;
    },
    Mutation: {
        routineVersionCreate: GQLEndpoint<RoutineVersionCreateInput, CreateOneResult<RoutineVersion>>;
        routineVersionUpdate: GQLEndpoint<RoutineVersionUpdateInput, UpdateOneResult<RoutineVersion>>;
    }
} = {
    RoutineVersionSortBy,
    Query: {
        routineVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        routineVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        routineVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        routineVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}