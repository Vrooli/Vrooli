import { FindVersionInput, StandardVersion, StandardVersionCreateInput, StandardVersionSearchInput, StandardVersionSortBy, StandardVersionUpdateInput } from ":/consts";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
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
        default: String
        isComplete: Boolean
        isPrivate: Boolean
        isFile: Boolean
        props: String!
        standardType: String!
        versionLabel: String!
        versionNotes: String
        yup: String
        directoryListingsConnect: [ID!]
        resourceListCreate: ResourceListCreateInput
        rootConnect: ID!
        rootCreate: StandardCreateInput
        translationsCreate: [StandardVersionTranslationCreateInput!]
    }
    input StandardVersionUpdateInput {
        id: ID!
        isComplete: Boolean
        isPrivate: Boolean
        isFile: Boolean
        default: String
        standardType: String
        props: String
        yup: String
        versionLabel: String
        versionNotes: String
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
        isComplete: Boolean!
        isLatest: Boolean!
        isDeleted: Boolean!
        isPrivate: Boolean!
        isFile: Boolean
        default: String
        standardType: String!
        props: String!
        yup: String
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
        language: String
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
        sortBy: StandardVersionSortBy
        standardType: String
        tagsRoot: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
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

const objectType = "StandardVersion";
export const resolvers: {
    StandardVersionSortBy: typeof StandardVersionSortBy;
    Query: {
        standardVersion: GQLEndpoint<FindVersionInput, FindOneResult<StandardVersion>>;
        standardVersions: GQLEndpoint<StandardVersionSearchInput, FindManyResult<StandardVersion>>;
    },
    Mutation: {
        standardVersionCreate: GQLEndpoint<StandardVersionCreateInput, CreateOneResult<StandardVersion>>;
        standardVersionUpdate: GQLEndpoint<StandardVersionUpdateInput, UpdateOneResult<StandardVersion>>;
    }
} = {
    StandardVersionSortBy,
    Query: {
        standardVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standardVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        standardVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
