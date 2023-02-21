import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Note, NoteSearchInput, NoteCreateInput, NoteUpdateInput, NoteSortBy } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum NoteSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IssuesAsc
        IssuesDesc
        PullRequestsAsc
        PullRequestsDesc
        QuestionsAsc
        QuestionsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    input NoteCreateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        userConnect: ID
        organizationConnect: ID
        parentConnect: ID
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [NoteVersionCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input NoteUpdateInput {
        id: ID!
        permissions: String
        userConnect: ID
        organizationConnect: ID
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [ApiVersionCreateInput!]
        versionsUpdate: [ApiVersionUpdateInput!]
        versionsDelete: [ID!]
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    type Note {
        id: ID!
        created_at: Date!
        updated_at: Date!
        isPrivate: Boolean!
        permissions: String!
        createdBy: User
        owner: Owner
        parent: NoteVersion
        tags: [Tag!]!
        versions: [NoteVersion!]!
        versionsCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        bookmarks: Int!
        views: Int!
        score: Int!
        issues: [Issue!]!
        issuesCount: Int!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        bookmarkedBy: [User!]!
        questions: [Question!]!
        questionsCount: Int!
        transfers: [Transfer!]!
        transfersCount: Int!
        you: NoteYou!
    }

    type NoteYou {
        canDelete: Boolean!
        canBookmark: Boolean!
        canTransfer: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canVote: Boolean!
        isBookmarked: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
    }

    input NoteSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        createdById: ID
        maxScore: Int
        maxBookmarks: Int
        minScore: Int
        minBookmarks: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        translationLanguagesLatestVersion: [String!]
        ids: [ID!]
        searchString: String
        sortBy: NoteSortBy
        tags: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type NoteSearchResult {
        pageInfo: PageInfo!
        edges: [NoteEdge!]!
    }

    type NoteEdge {
        cursor: String!
        node: Note!
    }

    extend type Query {
        note(input: FindByIdInput!): Note
        notes(input: NoteSearchInput!): NoteSearchResult!
    }

    extend type Mutation {
        noteCreate(input: NoteCreateInput!): Note!
        noteUpdate(input: NoteUpdateInput!): Note!
    }
`

const objectType = 'Note';
export const resolvers: {
    NoteSortBy: typeof NoteSortBy;
    Query: {
        note: GQLEndpoint<FindByIdInput, FindOneResult<Note>>;
        notes: GQLEndpoint<NoteSearchInput, FindManyResult<Note>>;
    },
    Mutation: {
        noteCreate: GQLEndpoint<NoteCreateInput, CreateOneResult<Note>>;
        noteUpdate: GQLEndpoint<NoteUpdateInput, UpdateOneResult<Note>>;
    }
} = {
    NoteSortBy,
    Query: {
        note: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        notes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        noteCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        noteUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}