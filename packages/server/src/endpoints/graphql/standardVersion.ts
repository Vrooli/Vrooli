import { StandardType, StandardVersionSortBy } from "@local/shared";
import { EndpointsStandardVersion, StandardVersionEndpoints } from "../logic/standardVersion";

export const typeDef = `#graphql
    enum StandardType {
        DataStructure
        Prompt
    }

    enum StandardVersionSortBy {
        CommentsAsc
        CommentsDesc
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
        ReportsAsc
        ReportsDesc
    }

    input StandardVersionCreateInput {
        id: ID!
        codeLanguage: String!
        default: String
        isComplete: Boolean
        isPrivate: Boolean!
        isFile: Boolean
        props: String!
        variant: StandardType!
        versionLabel: String!
        versionNotes: String
        yup: String
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID
        rootCreate: StandardCreateInput
        translationsCreate: [StandardVersionTranslationCreateInput!]
    }
    input StandardVersionUpdateInput {
        id: ID!
        codeLanguage: String
        isComplete: Boolean
        isPrivate: Boolean
        isFile: Boolean
        default: String
        props: String
        variant: StandardType
        versionLabel: String
        versionNotes: String
        yup: String
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rootUpdate: StandardUpdateInput
        translationsCreate: [StandardVersionTranslationCreateInput!]
        translationsUpdate: [StandardVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type StandardVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        completedAt: Date
        codeLanguage: String!
        isComplete: Boolean!
        isLatest: Boolean!
        isDeleted: Boolean!
        isPrivate: Boolean!
        isFile: Boolean
        default: String
        props: String!
        yup: String
        variant: StandardType!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        forks: [Standard!]!
        forksCount: Int!
        pullRequest: PullRequest
        resourceList: ResourceList
        reports: [Report!]!
        reportsCount: Int!
        root: Standard!
        translations: [StandardVersionTranslation!]!
        translationsCount: Int!
        you: VersionYou!
    }

    input StandardVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }
    input StandardVersionTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
        jsonVariable: String
    }
    type StandardVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }

    input StandardVersionSearchInput {
        after: String
        codeLanguage: String
        createdTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        createdByIdRoot: ID
        ownedByTeamIdRoot: ID
        ownedByUserIdRoot: ID
        ids: [ID!]
        isCompleteWithRoot: Boolean
        isInternalWithRoot: Boolean
        isLatest: Boolean
        translationLanguages: [String!]
        maxBookmarksRoot: Int
        maxScoreRoot: Int
        maxViewsRoot: Int
        minBookmarksRoot: Int
        minScoreRoot: Int
        minViewsRoot: Int
        reportId: ID
        rootId: ID
        searchString: String
        sortBy: StandardVersionSortBy
        tagsRoot: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        variant: StandardType
        visibility: VisibilityType
    }

    type StandardVersionSearchResult {
        pageInfo: PageInfo!
        edges: [StandardVersionEdge!]!
    }

    type StandardVersionEdge {
        cursor: String!
        node: StandardVersion!
    }

    extend type Query {
        standardVersion(input: FindVersionInput!): StandardVersion
        standardVersions(input: StandardVersionSearchInput!): StandardVersionSearchResult!
    }

    extend type Mutation {
        standardVersionCreate(input: StandardVersionCreateInput!): StandardVersion!
        standardVersionUpdate(input: StandardVersionUpdateInput!): StandardVersion!
    }
`;

export const resolvers: {
    StandardType: typeof StandardType;
    StandardVersionSortBy: typeof StandardVersionSortBy;
    Query: EndpointsStandardVersion["Query"];
    Mutation: EndpointsStandardVersion["Mutation"];
} = {
    StandardType,
    StandardVersionSortBy,
    ...StandardVersionEndpoints,
};
