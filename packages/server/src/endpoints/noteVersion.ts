import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, NoteVersion, NoteVersionSearchInput, NoteVersionCreateInput, NoteVersionUpdateInput, NoteVersionSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
        isLatest: Boolean
        isPrivate: Boolean
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        translationsCreate: [NoteVersionTranslationCreateInput!]
        directoryListingsConnect: [ID!]
    }
    input NoteVersionUpdateInput {
        id: ID!
        isLatest: Boolean
        isPrivate: Boolean
        versionIndex: Int
        versionLabel: String
        versionNotes: String
        translationsCreate: [NoteVersionTranslationCreateInput!]
        translationsUpdate: [NoteVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
        directoryListingsConnect: [ID!]
        directoryListingsDisconnect: [ID!]
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
        permissionsVersion: VersionPermission!
    }

    type NoteVersionPermission {
        canComment: Boolean!
        canCopy: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input NoteVersionTranslationCreateInput {
        id: ID!
        language: String!
        text: String!
        description: String
    }
    input NoteVersionTranslationUpdateInput {
        id: ID!
        language: String
        text: String
        description: String
    }
    type NoteVersionTranslation {
        id: ID!
        language: String!
        text: String!
        description: String
    }

    input NoteVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        createdById: ID
        ownedByUserId: ID
        ownedByOrganizationId: ID
        searchString: String
        sortBy: NoteVersionSortBy
        tags: [String!]
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
        noteVersion(input: FindByIdInput!): NoteVersion
        noteVersions(input: NoteVersionSearchInput!): NoteVersionSearchResult!
    }

    extend type Mutation {
        noteVersionCreate(input: NoteVersionCreateInput!): NoteVersion!
        noteVersionUpdate(input: NoteVersionUpdateInput!): NoteVersion!
    }
`

const objectType = 'NoteVersion';
export const resolvers: {
    NoteVersionSortBy: typeof NoteVersionSortBy;
    Query: {
        noteVersion: GQLEndpoint<FindByIdInput, FindOneResult<NoteVersion>>;
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
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        noteVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        noteVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        noteVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}