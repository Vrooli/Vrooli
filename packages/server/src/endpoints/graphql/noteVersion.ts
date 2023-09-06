import { NoteVersionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsNoteVersion, NoteVersionEndpoints } from "../logic";

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
        pagesCreate: [NotePageCreateInput!]
    }
    input NoteVersionTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
        pagesCreate: [NotePageCreateInput!]
        pagesUpdate: [NotePageUpdateInput!]
        pagesDelete: [ID!]
    }
    type NoteVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
        pages: [NotePage!]!
    }

    input NotePageCreateInput {
        id: ID!
        pageIndex: Int!
        text: String!
    }
    input NotePageUpdateInput {
        id: ID!
        pageIndex: Int
        text: String
    }
    type NotePage {
        id: ID!
        pageIndex: Int!
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

export const resolvers: {
    NoteVersionSortBy: typeof NoteVersionSortBy;
    Query: EndpointsNoteVersion["Query"];
    Mutation: EndpointsNoteVersion["Mutation"];
} = {
    NoteVersionSortBy,
    ...NoteVersionEndpoints,
};
