import { ApiVersionSortBy } from "@local/consts";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
export const typeDef = gql `
    enum ApiVersionSortBy {
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

    input ApiVersionCreateInput {
        id: ID!
        callLink: String!
        documentationLink: String
        isPrivate: Boolean
        isComplete: Boolean
        versionLabel: String!
        versionNotes: String
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID
        rootCreate: ApiCreateInput
        translationsCreate: [ApiVersionTranslationCreateInput!]
    }
    input ApiVersionUpdateInput {
        id: ID!
        callLink: String
        documentationLink: String
        isPrivate: Boolean
        isComplete: Boolean
        versionLabel: String
        versionNotes: String
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        resourceListUpdate: ResourceListUpdateInput
        rootUpdate: ApiUpdateInput
        translationsCreate: [ApiVersionTranslationCreateInput!]
        translationsUpdate: [ApiVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type ApiVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        callLink: String!
        documentationLink: String
        isLatest: Boolean!
        isPrivate: Boolean!
        isComplete: Boolean!
        resourceList: ResourceList
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        translations: [ApiVersionTranslation!]!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        pullRequest: PullRequest
        reports: [Report!]!
        reportsCount: Int!
        root: Api!
        forks: [Api!]!
        forksCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        you: VersionYou!
    }

    input ApiVersionTranslationCreateInput {
        id: ID!
        language: String!
        details: String
        name: String!
        summary: String
    }
    input ApiVersionTranslationUpdateInput {
        id: ID!
        language: String
        details: String
        name: String
        summary: String
    }
    type ApiVersionTranslation {
        id: ID!
        language: String!
        details: String
        name: String!
        summary: String
    }

    input ApiVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isCompleteWithRoot: Boolean
        isLatest: Boolean
        maxBookmarksRoot: Int
        maxScoreRoot: Int
        maxViewsRoot: Int
        minBookmarksRoot: Int
        minScoreRoot: Int
        minViewsRoot: Int
        createdByIdRoot: ID
        ownedByUserIdRoot: ID
        ownedByOrganizationIdRoot: ID
        searchString: String
        sortBy: ApiVersionSortBy
        tagsRoot: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ApiVersionSearchResult {
        pageInfo: PageInfo!
        edges: [ApiVersionEdge!]!
    }

    type ApiVersionEdge {
        cursor: String!
        node: ApiVersion!
    }

    extend type Query {
        apiVersion(input: FindVersionInput!): ApiVersion
        apiVersions(input: ApiVersionSearchInput!): ApiVersionSearchResult!
    }

    extend type Mutation {
        apiVersionCreate(input: ApiVersionCreateInput!): ApiVersion!
        apiVersionUpdate(input: ApiVersionUpdateInput!): ApiVersion!
    }
`;
const objectType = "ApiVersion";
export const resolvers = {
    ApiVersionSortBy,
    Query: {
        apiVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        apiVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        apiVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        apiVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
//# sourceMappingURL=apiVersion.js.map