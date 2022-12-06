import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { RoutineVersionSortBy, FindByIdInput, RoutineVersionSearchInput, RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput } from './types';
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
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isPrivate: Boolean
        versionLabel: String
        parentId: ID
        projectId: ID
        createdByUserId: ID
        createdByOrganizationId: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [InputItemCreateInput!]
        outputsCreate: [OutputItemCreateInput!]
        resourceListsCreate: [ResourceListCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [RoutineTranslationCreateInput!]
    }
    input RoutineVersionUpdateInput {
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isPrivate: Boolean
        versionId: ID # If versionId passed, then we're updating an existing version
        versionLabel: String # If version label passed, then we're creating a new version
        userId: ID
        organizationId: ID
        projectId: ID
        nodesDelete: [ID!]
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodeLinksDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        inputsCreate: [InputItemCreateInput!]
        inputsUpdate: [InputItemUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [OutputItemCreateInput!]
        outputsUpdate: [OutputItemUpdateInput!]
        outputsDelete: [ID!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [RoutineTranslationCreateInput!]
        translationsUpdate: [RoutineTranslationUpdateInput!]
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
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        simplicity: Int!
        stars: Int!
        views: Int!
        versionLabel: String!
        rootId: ID!
        # versions: [Version!]!
        comments: [Comment!]!
        commentsCount: Int!
        creator: Contributor
        forks: [Routine!]!
        inputs: [InputItem!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        nodesCount: Int
        nodeLinks: [NodeLink!]!
        outputs: [OutputItem!]!
        owner: Contributor
        parent: Routine
        permissionsRoutine: RoutinePermission!
        project: Project
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        runs: [RunRoutine!]!
        starredBy: [User!]!
        tags: [Tag!]!
        translations: [RoutineTranslation!]!
    }

    type RoutineVersionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canFork: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canRun: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input RoutineVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        instructions: String!
        title: String!
    }
    input RoutineVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        instructions: String
        title: String
    }
    type RoutineVersionTranslation {
        id: ID!
        language: String!
        description: String
        instructions: String!
        title: String!
    }

    input InputItemCreateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardVersionConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    input InputItemUpdateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [InputItemTranslationCreateInput!]
        translationsUpdate: [InputItemTranslationUpdateInput!]
    }
    type InputItem {
        id: ID!
        isRequired: Boolean
        name: String
        routineVersion: RoutineVersion!
        standardVersion: StandardVersion
        translations: [InputItemTranslation!]!
    }

    input InputItemTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input InputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type InputItemTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }

    input OutputItemCreateInput {
        id: ID!
        name: String
        standardVersionConnect: ID
        standardCreate: StandardCreateInput
        translationsCreate: [OutputItemTranslationCreateInput!]
    }
    input OutputItemUpdateInput {
        id: ID!
        name: String
        standardVersionConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [OutputItemTranslationCreateInput!]
        translationsUpdate: [OutputItemTranslationUpdateInput!]
    }
    type OutputItem {
        id: ID!
        name: String
        routineVersion: RoutineVersion!
        standardVersion: StandardVersion
        translations: [OutputItemTranslation!]!
    }

    input OutputItemTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input OutputItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type OutputItemTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }

    input RoutineVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        isInternal: Boolean
        isInternalExceptions: [SearchException!]
        languages: [String!]
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        maxTimesCompleted: Int
        minScore: Int
        minStars: Int
        minTimesCompleted: Int
        minViews: Int
        organizationId: ID
        projectId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type RoutineVersionSearchResult {
        pageInfo: PageInfo!
        edges: [RoutineEdge!]!
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