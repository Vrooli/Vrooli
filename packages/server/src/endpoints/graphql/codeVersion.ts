import { CodeType, CodeVersionSortBy } from "@local/shared";
import { CodeVersionEndpoints, EndpointsCodeVersion } from "../logic/codeVersion";

export const typeDef = `#graphql
    enum CodeType {
        DataConvert
        SmartContract
    }

    enum CodeVersionSortBy {
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

    input CodeVersionCreateInput {
        id: ID!
        isComplete: Boolean
        isPrivate: Boolean!
        codeLanguage: String!
        codeType: CodeType!
        content: String!
        default: String
        versionLabel: String!
        versionNotes: String
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID!
        rootCreate: CodeCreateInput
        translationsCreate: [CodeVersionTranslationCreateInput!]
    }
    input CodeVersionUpdateInput {
        id: ID!
        isComplete: Boolean
        isPrivate: Boolean
        codeLanguage: String
        content: String
        default: String
        versionLabel: String
        versionNotes: String
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rootUpdate: CodeUpdateInput
        translationsCreate: [CodeVersionTranslationCreateInput!]
        translationsUpdate: [CodeVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type CodeVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        codeLanguage: String!
        codeType: CodeType!
        completedAt: Date
        content: String!
        default: String
        isComplete: Boolean!
        isDeleted: Boolean!
        isLatest: Boolean!
        isPrivate: Boolean!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        calledByRoutineVersionsCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        forks: [Code!]!
        forksCount: Int!
        pullRequest: PullRequest
        resourceList: ResourceList
        reports: [Report!]!
        reportsCount: Int!
        root: Code!
        translations: [CodeVersionTranslation!]!
        translationsCount: Int!
        you: VersionYou!
    }

    input CodeVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }
    input CodeVersionTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
        jsonVariable: String
    }
    type CodeVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }

    input CodeVersionSearchInput {
        after: String
        calledByRoutineVersionId: ID
        codeLanguage: String
        codeType: CodeType
        createdTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        createdByIdRoot: ID
        ownedByTeamIdRoot: ID
        ownedByUserIdRoot: ID
        ids: [ID!]
        isCompleteWithRoot: Boolean
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
        sortBy: CodeVersionSortBy
        tagsRoot: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type CodeVersionSearchResult {
        pageInfo: PageInfo!
        edges: [CodeVersionEdge!]!
    }

    type CodeVersionEdge {
        cursor: String!
        node: CodeVersion!
    }

    extend type Query {
        codeVersion(input: FindVersionInput!): CodeVersion
        codeVersions(input: CodeVersionSearchInput!): CodeVersionSearchResult!
    }

    extend type Mutation {
        codeVersionCreate(input: CodeVersionCreateInput!): CodeVersion!
        codeVersionUpdate(input: CodeVersionUpdateInput!): CodeVersion!
    }
`;

export const resolvers: {
    CodeType: typeof CodeType;
    CodeVersionSortBy: typeof CodeVersionSortBy;
    Query: EndpointsCodeVersion["Query"];
    Mutation: EndpointsCodeVersion["Mutation"];
} = {
    CodeType,
    CodeVersionSortBy,
    ...CodeVersionEndpoints,
};
