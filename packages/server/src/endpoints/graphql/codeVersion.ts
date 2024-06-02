import { CodeVersionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { CodeVersionEndpoints, EndpointsCodeVersion } from "../logic/codeVersion";

export const typeDef = gql`
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
        default: String
        contractType: String!
        content: String!
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
        default: String
        contractType: String
        content: String
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
        contractType: String
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
    CodeVersionSortBy: typeof CodeVersionSortBy;
    Query: EndpointsCodeVersion["Query"];
    Mutation: EndpointsCodeVersion["Mutation"];
} = {
    CodeVersionSortBy,
    ...CodeVersionEndpoints,
};
