import { FindVersionInput, NoteVersion, NoteVersionCreateInput, NoteVersionSearchInput, NoteVersionSortBy, NoteVersionUpdateInput } from "@local/consts";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
    enum NoteVersionSortBy {
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

    input NoteVersionCreateInput {
        id: ID!
        isPrivate: Boolean
        versionLabel: String!
        versionNotes: String
        directoryListingsConnect: [ID!]
        rootConnect: ID
        rootCreate: NoteCreateInput
        translationsCreate: [NoteVersionTranslationCreateInput!]
    }
    input NoteVersionUpdateInput {
        id: ID!
        isPrivate: Boolean
        versionLabel: String
        versionNotes: String
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
        rootUpdate: NoteUpdateInput
        translationsCreate: [NoteVersionTranslationCreateInput!]
        translationsUpdate: [NoteVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type NoteVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        isLatest: Boolean!
        isPrivate: Boolean!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        translations: [NoteVersionTranslation!]!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        pullRequest: PullRequest
        reports: [Report!]!
        reportsCount: Int!
        root: Note!
        forks: [Note!]!
        forksCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        you: VersionYou!
    }

    input NoteVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
        text: String!
    }
    input NoteVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
        text: String
    }
    type NoteVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        text: String!
    }

    input NoteVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isLatest: Boolean
        translationLanguages: [String!]
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
        sortBy: NoteVersionSortBy
        tagsRoot: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type NoteVersionSearchResult {
        pageInfo: PageInfo!
        edges: [NoteVersionEdge!]!
    }

    type NoteVersionEdge {
        cursor: String!
        node: NoteVersion!
    }

    extend type Query {
        noteVersion(input: FindVersionInput!): NoteVersion
        noteVersions(input: NoteVersionSearchInput!): NoteVersionSearchResult!
    }

    extend type Mutation {
        noteVersionCreate(input: NoteVersionCreateInput!): NoteVersion!
        noteVersionUpdate(input: NoteVersionUpdateInput!): NoteVersion!
    }
`;

const objectType = "NoteVersion";
export const resolvers: {
    NoteVersionSortBy: typeof NoteVersionSortBy;
    Query: {
        noteVersion: GQLEndpoint<FindVersionInput, FindOneResult<NoteVersion>>;
        noteVersions: GQLEndpoint<NoteVersionSearchInput, FindManyResult<NoteVersion>>;
    },
    Mutation: {
        noteVersionCreate: GQLEndpoint<NoteVersionCreateInput, CreateOneResult<NoteVersion>>;
        noteVersionUpdate: GQLEndpoint<NoteVersionUpdateInput, UpdateOneResult<NoteVersion>>;
    }
} = {
    NoteVersionSortBy,
    Query: {
        noteVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        noteVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        noteVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        noteVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
