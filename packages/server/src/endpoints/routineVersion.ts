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
        projectConnect: ID
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        resourceListsCreate: [ResourceListCreateInput!]
        suggestedNextByRoutineVersionConnect: [ID!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [RoutineTranslationCreateInput!]
    }
    input RoutineVersionUpdateInput {
        isAutomatable: Boolean
        isComplete: Boolean
        isInternal: Boolean
        isPrivate: Boolean
        versionConnect: ID # If versionId passed, then we're updating an existing version
        versionLabel: String # If version label passed, then we're creating a new version
        userConnect: ID
        nodesDelete: [ID!]
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodeLinksDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        inputsUpdate: [RoutineVersionInputUpdateInput!]
        inputsDelete: [ID!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        outputsUpdate: [RoutineVersionOutputUpdateInput!]
        outputsDelete: [ID!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        suggestedNextByRoutineVersionConnect: [ID!]
        suggestedNextByRoutineVersionDisconnect: [ID!]
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
        createdBy: User
        forks: [Routine!]!
        inputs: [RoutineVersionInput!]!
        nodeLists: [NodeRoutineList!]!
        nodes: [Node!]!
        nodesCount: Int
        nodeLinks: [NodeLink!]!
        outputs: [RoutineVersionOutput!]!
        owner: Owner
        parent: Routine
        permissionsRoutine: RoutinePermission!
        project: Project
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]!
        runs: [RunRoutine!]!
        suggestedNextByRoutineVersion: [RoutineVersion!]!
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
        parentId: ID
        reportId: ID
        rootId: ID
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
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