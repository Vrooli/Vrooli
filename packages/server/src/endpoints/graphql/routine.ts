import { RoutineSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRoutine, RoutineEndpoints } from "../logic/routine";

export const typeDef = gql`
    enum RoutineSortBy {
        DateCompletedAsc
        DateCompletedDesc
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
        QuizzesAsc
        QuizzesDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    input RoutineCreateInput {
        id: ID!
        isInternal: Boolean
        isPrivate: Boolean!
        permissions: String
        parentConnect: ID
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [RoutineVersionCreateInput!]
    }
    input RoutineUpdateInput {
        id: ID!
        isInternal: Boolean
        isPrivate: Boolean
        permissions: String
        ownedByTeamConnect: ID
        ownedByUserConnect: ID
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsDisconnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [RoutineVersionCreateInput!]
        versionsUpdate: [RoutineVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type Routine {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompleteVersion: Boolean!
        isDeleted: Boolean!
        isInternal: Boolean
        isPrivate: Boolean!
        translatedName: String!
        score: Int!
        bookmarks: Int!
        views: Int!
        createdBy: User
        forks: [Routine!]!
        forksCount: Int!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: RoutineVersion
        permissions: String!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        quizzes: [Quiz!]!
        quizzesCount: Int!
        bookmarkedBy: [User!]!
        stats: [StatsRoutine!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        transfersCount: Int!
        versions: [RoutineVersion!]!
        versionsCount: Int
        you: RoutineYou!
    }

    type RoutineYou {
        canComment: Boolean!
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

    input RoutineSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
        isInternal: Boolean
        issuesId: ID
        labelsIds: [ID!]
        maxScore: Int
        maxBookmarks: Int
        maxViews: Int
        minScore: Int
        minBookmarks: Int
        minViews: Int
        ownedByTeamId: ID
        ownedByUserId: ID
        parentId: ID
        pullRequestsId: ID
        searchString: String
        sortBy: RoutineSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RoutineSearchResult {
        pageInfo: PageInfo!
        edges: [RoutineEdge!]!
    }

    type RoutineEdge {
        cursor: String!
        node: Routine!
    }

    extend type Query {
        routine(input: FindByIdInput!): Routine
        routines(input: RoutineSearchInput!): RoutineSearchResult!
    }

    extend type Mutation {
        routineCreate(input: RoutineCreateInput!): Routine!
        routineUpdate(input: RoutineUpdateInput!): Routine!
    }
`;

export const resolvers: {
    RoutineSortBy: typeof RoutineSortBy;
    Query: EndpointsRoutine["Query"];
    Mutation: EndpointsRoutine["Mutation"];
} = {
    RoutineSortBy,
    ...RoutineEndpoints,
};
