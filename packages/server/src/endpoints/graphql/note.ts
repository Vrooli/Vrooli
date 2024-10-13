import { NoteSortBy } from "@local/shared";
import { EndpointsNote, NoteEndpoints } from "../logic/note";

export const typeDef = `#graphql
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
        isPrivate: Boolean!
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        parentConnect: ID
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [NoteVersionCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input NoteUpdateInput {
        id: ID!
        isPrivate: Boolean
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [NoteVersionCreateInput!]
        versionsUpdate: [NoteVersionUpdateInput!]
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
        canReact: Boolean!
        isBookmarked: Boolean!
        reaction: String
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
        ownedByTeamId: ID
        ownedByUserId: ID
        parentId: ID
        translationLanguagesLatestVersion: [String!]
        ids: [ID!]
        searchString: String
        sortBy: NoteSortBy
        take: Int
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
`;

export const resolvers: {
    NoteSortBy: typeof NoteSortBy;
    Query: EndpointsNote["Query"];
    Mutation: EndpointsNote["Mutation"];
} = {
    NoteSortBy,
    ...NoteEndpoints,
};
