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
        StarsAsc
        StarsDesc
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
        type: GqlModelType!
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompletedVersion: Boolean!
        isDeleted: Boolean!
        isInternal: Boolean
        isPrivate: Boolean!
        translatedName: String!
        score: Int!
        stars: Int!
        views: Int!
        createdBy: User
        forks: [Routine!]!
        forksCount: Int!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: Routine
        permissions: String!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        starredBy: [User!]!
        tags: [Tag!]!
        versions: [RoutineVersion!]!
        versionsCount: Int
        you: RoutineYou!
    }

    type RoutineYou {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canView: Boolean!
        canVote: Boolean!
        isStarred: Boolean!
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
        labelsId: ID
        maxScore: Int
        maxStars: Int
        maxViews: Int
        minScore: Int
        minStars: Int
        minViews: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
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