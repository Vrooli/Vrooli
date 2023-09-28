import { SmartContractVersionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsSmartContractVersion, SmartContractVersionEndpoints } from "../logic";

export const typeDef = gql`
    enum SmartContractVersionSortBy {
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

    input SmartContractVersionCreateInput {
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
        rootCreate: SmartContractCreateInput
        translationsCreate: [SmartContractVersionTranslationCreateInput!]
    }
    input SmartContractVersionUpdateInput {
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
        rootUpdate: SmartContractUpdateInput
        translationsCreate: [SmartContractVersionTranslationCreateInput!]
        translationsUpdate: [SmartContractVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type SmartContractVersion {
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
        forks: [SmartContract!]!
        forksCount: Int!
        pullRequest: PullRequest
        resourceList: ResourceList
        reports: [Report!]!
        reportsCount: Int!
        root: SmartContract!
        translations: [SmartContractVersionTranslation!]!
        translationsCount: Int!
        you: VersionYou!
    }

    input SmartContractVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }
    input SmartContractVersionTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
        jsonVariable: String
    }
    type SmartContractVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        jsonVariable: String
    }

    input SmartContractVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        createdByIdRoot: ID
        ownedByUserIdRoot: ID
        ownedByOrganizationIdRoot: ID
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
        sortBy: SmartContractVersionSortBy
        contractType: String
        tagsRoot: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type SmartContractVersionSearchResult {
        pageInfo: PageInfo!
        edges: [SmartContractVersionEdge!]!
    }

    type SmartContractVersionEdge {
        cursor: String!
        node: SmartContractVersion!
    }

    extend type Query {
        smartContractVersion(input: FindVersionInput!): SmartContractVersion
        smartContractVersions(input: SmartContractVersionSearchInput!): SmartContractVersionSearchResult!
    }

    extend type Mutation {
        smartContractVersionCreate(input: SmartContractVersionCreateInput!): SmartContractVersion!
        smartContractVersionUpdate(input: SmartContractVersionUpdateInput!): SmartContractVersion!
    }
`;

export const resolvers: {
    SmartContractVersionSortBy: typeof SmartContractVersionSortBy;
    Query: EndpointsSmartContractVersion["Query"];
    Mutation: EndpointsSmartContractVersion["Mutation"];
} = {
    SmartContractVersionSortBy,
    ...SmartContractVersionEndpoints,
};
