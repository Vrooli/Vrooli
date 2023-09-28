import { RoutineVersionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRoutineVersion, RoutineVersionEndpoints } from "../logic";

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
        isPrivate: Boolean!
        versionLabel: String!
        versionNotes: String
        smartContractCallData: String
        apiVersionConnect: ID
        directoryListingsConnect: [ID!]
        inputsCreate: [RoutineVersionInputCreateInput!]
        nodesCreate: [NodeCreateInput!]
        nodeLinksCreate: [NodeLinkCreateInput!]
        outputsCreate: [RoutineVersionOutputCreateInput!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID
        rootCreate: RoutineCreateInput
        smartContractVersionConnect: ID
        suggestedNextByRoutineVersionConnect: [ID!]
        translationsCreate: [RoutineVersionTranslationCreateInput!]
    }
    input RoutineVersionUpdateInput {
        id: ID!
        apiCallData: String
        isAutomatable: Boolean
        isComplete: Boolean
        isPrivate: Boolean
        versionLabel: String
        versionNotes: String
        smartContractCallData: String
        apiVersionConnect: ID
        apiVersionDisconnect: Boolean
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
        smartContractVersionConnect: ID
        smartContractVersionDisconnect: Boolean
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
        apiVersion: ApiVersion
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
        smartContractVersion: SmartContractVersion
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
        isCompleteWithRootExcludeOwnedByOrganizationId: ID
        isCompleteWithRootExcludeOwnedByUserId: ID
        isInternalWithRoot: Boolean
        isInternalWithRootExcludeOwnedByOrganizationId: ID
        isInternalWithRootExcludeOwnedByUserId: ID
        isExternalWithRootExcludeOwnedByOrganizationId: ID
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
        ownedByUserIdRoot: ID
        ownedByOrganizationIdRoot: ID
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
    RoutineVersionSortBy: typeof RoutineVersionSortBy;
    Query: EndpointsRoutineVersion["Query"];
    Mutation: EndpointsRoutineVersion["Mutation"];
} = {
    RoutineVersionSortBy,
    ...RoutineVersionEndpoints,
};
