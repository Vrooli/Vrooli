import { RoutineType, RoutineVersionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRoutineVersion, RoutineVersionEndpoints } from "../logic/routineVersion";

export const typeDef = gql`
    enum RoutineType {
        Action
        Api
        Code
        Data
        Generate
        Informational
        MultiStep
        SmartContract
    }

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
        configCallData: String
        isAutomatable: Boolean
        isComplete: Boolean
        isPrivate: Boolean!
        routineType: RoutineType!
        versionLabel: String!
        versionNotes: String
        apiVersionConnect: ID
        codeVersionConnect: ID
        directoryListingsConnect: [ID!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID
        rootCreate: RoutineCreateInput
        suggestedNextByRoutineVersionConnect: [ID!]
        translationsCreate: [RoutineVersionTranslationCreateInput!]
    }
    input RoutineVersionUpdateInput {
        id: ID!
        configCallData: String
        isAutomatable: Boolean
        isComplete: Boolean
        isPrivate: Boolean
        versionLabel: String
        versionNotes: String
        apiVersionConnect: ID
        apiVersionDisconnect: Boolean
        codeVersionConnect: ID
        codeVersionDisconnect: Boolean
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        inputsUpdate: [RoutineVersionInputUpdateInput!]
        inputsDelete: [ID!]
        nodesCreate: [NodeCreateInput!]
        nodesUpdate: [NodeUpdateInput!]
        nodesDelete: [ID!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        nodeLinksUpdate: [NodeLinkUpdateInput!]
        nodeLinksDelete: [ID!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        outputsUpdate: [RoutineVersionOutputUpdateInput!]
        outputsDelete: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rootUpdate: RoutineUpdateInput
        suggestedNextByRoutineVersionConnect: [ID!]
        suggestedNextByRoutineVersionDisconnect: [ID!]
        translationsCreate: [RoutineVersionTranslationCreateInput!]
        translationsUpdate: [RoutineVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type RoutineVersion {
        id: ID!
        configCallData: String
        completedAt: Date
        complexity: Int!
        created_at: Date!
        updated_at: Date!
        isAutomatable: Boolean
        isComplete: Boolean!
        isDeleted: Boolean!
        isLatest: Boolean!
        isPrivate: Boolean!
        simplicity: Int!
        routineType: RoutineType!
        timesStarted: Int!
        timesCompleted: Int!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        apiVersion: ApiVersion
        codeVersion: CodeVersion
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
        suggestedNextByRoutineVersion: [RoutineVersion!]!
        suggestedNextByRoutineVersionCount: Int!
        translations: [RoutineVersionTranslation!]!
        translationsCount: Int!
        you: RoutineVersionYou!
    }

    type RoutineVersionYou {
        canComment: Boolean!
        canCopy: Boolean!
        canDelete: Boolean!
        canBookmark: Boolean!
        canReport: Boolean!
        canUpdate: Boolean!
        canRun: Boolean!
        canRead: Boolean!
        canReact: Boolean!
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
        language: String!
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
        createdByIdRoot: ID
        createdTimeFrame: TimeFrame
        directoryListingsId: ID
        excludeIds: [ID!]
        ids: [ID!]
        isCompleteWithRoot: Boolean
        isCompleteWithRootExcludeOwnedByTeamId: ID
        isCompleteWithRootExcludeOwnedByUserId: ID
        isInternalWithRoot: Boolean
        isInternalWithRootExcludeOwnedByTeamId: ID
        isInternalWithRootExcludeOwnedByUserId: ID
        isExternalWithRootExcludeOwnedByTeamId: ID
        isExternalWithRootExcludeOwnedByUserId: ID
        isLatest: Boolean
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        maxTimesCompleted: Int
        minTimesCompleted: Int
        maxBookmarksRoot: Int
        maxScoreRoot: Int
        maxViewsRoot: Int
        minBookmarksRoot: Int
        minScoreRoot: Int
        minViewsRoot: Int
        ownedByTeamIdRoot: ID
        ownedByUserIdRoot: ID
        reportId: ID
        rootId: ID
        searchString: String
        sortBy: RoutineVersionSortBy
        tagsRoot: [String!]
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
        routineVersion(input: FindVersionInput!): RoutineVersion
        routineVersions(input: RoutineVersionSearchInput!): RoutineVersionSearchResult!
    }

    extend type Mutation {
        routineVersionCreate(input: RoutineVersionCreateInput!): RoutineVersion!
        routineVersionUpdate(input: RoutineVersionUpdateInput!): RoutineVersion!
    }
`;

export const resolvers: {
    RoutineType: typeof RoutineType;
    RoutineVersionSortBy: typeof RoutineVersionSortBy;
    Query: EndpointsRoutineVersion["Query"];
    Mutation: EndpointsRoutineVersion["Mutation"];
} = {
    RoutineType,
    RoutineVersionSortBy,
    ...RoutineVersionEndpoints,
};
