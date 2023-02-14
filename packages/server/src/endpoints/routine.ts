import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { Routine, RoutineCreateInput, RoutineUpdateInput, RoutineSearchInput, RoutineSortBy, FindByIdInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
        isPrivate: Boolean
        permissions: String
        parentConnect: ID
        userConnect: ID
        organizationConnect: ID
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        tagsConnect: [ID!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [RoutineVersionCreateInput!]
    }
    input RoutineUpdateInput {
        isInternal: Boolean
        isPrivate: Boolean
        permissions: String
        userConnect: ID
        organizationConnect: ID
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
        canStar: Boolean!
        canTransfer: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
        canVote: Boolean!
        isBookmarked: Boolean!
        isUpvoted: Boolean
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
        ownedByUserId: ID
        ownedByOrganizationId: ID
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
`

const objectType = 'Routine';
export const resolvers: {
    RoutineSortBy: typeof RoutineSortBy;
    Query: {
        routine: GQLEndpoint<FindByIdInput, FindOneResult<Routine>>;
        routines: GQLEndpoint<RoutineSearchInput, FindManyResult<Routine>>;
    },
    Mutation: {
        routineCreate: GQLEndpoint<RoutineCreateInput, CreateOneResult<Routine>>;
        routineUpdate: GQLEndpoint<RoutineUpdateInput, UpdateOneResult<Routine>>;
    }
} = {
    RoutineSortBy,
    Query: {
        routine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        routines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        routineCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        routineUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}