import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdOrHandleInput, Project, ProjectCreateInput, ProjectUpdateInput, ProjectSearchInput, ProjectSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ProjectSortBy {
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
        ScoreAsc
        ScoreDesc
        StarsAsc
        StarsDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    input ProjectCreateInput {
        id: ID!
        ownedByUserId: ID
        ownedByOrganizationId: ID
        handle: String
        isPrivate: Boolean
        parentId: ID
        permissions: String
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        versionsCreate: [ProjectVersionCreateInput!]
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    input ProjectUpdateInput {
        id: ID!
        ownedByUserId: ID
        ownedByOrganizationId: ID
        handle: String
        isPrivate: Boolean
        permissions: String
        labelsConnect: [ID!]
        labelsDisconnect: [ID!]
        labelsCreate: [LabelCreateInput!]
        versionsCreate: [ProjectVersionCreateInput!]
        versionsUpdate: [ProjectVersionUpdateInput!]
        versionsDelete: [ID!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    type Project {
        id: ID!
        created_at: Date!
        updated_at: Date!
        handle: String
        hasCompleteVersion: Boolean!
        permissions: String
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        permissionsRoot: RootPermission!
        createdBy: User
        owner: Owner
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        labelsCount: Int!
        parent: Project
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        questions: [Question!]!
        questionsCount: Int!
        quizzes: [Quiz!]!
        quizzesCount: Int!
        starredBy: [User!]!
        stats: [StatsProject!]!
        tags: [Tag!]!
        transfers: [Transfer!]!
        versions: [ProjectVersion!]!
        versionsCount: Int!
    }

    input ProjectSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        createdById: ID
        hasCompleteVersion: Boolean
        labelsId: ID
        maxScore: Int
        maxStars: Int
        minScore: Int
        minStars: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        ids: [ID!]
        searchString: String
        sortBy: ProjectSortBy
        tags: [String!]
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ProjectSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectEdge!]!
    }

    type ProjectEdge {
        cursor: String!
        node: Project!
    }

    extend type Query {
        project(input: FindByIdOrHandleInput!): Project
        projects(input: ProjectSearchInput!): ProjectSearchResult!
    }

    extend type Mutation {
        projectCreate(input: ProjectCreateInput!): Project!
        projectUpdate(input: ProjectUpdateInput!): Project!
    }
`

const objectType = 'Project';
export const resolvers: {
    ProjectSortBy: typeof ProjectSortBy;
    Query: {
        project: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<Project>>;
        projects: GQLEndpoint<ProjectSearchInput, FindManyResult<Project>>;
    },
    Mutation: {
        projectCreate: GQLEndpoint<ProjectCreateInput, CreateOneResult<Project>>;
        projectUpdate: GQLEndpoint<ProjectUpdateInput, UpdateOneResult<Project>>;
    }
} = {
    ProjectSortBy,
    Query: {
        project: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        projects: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        projectCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        projectUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}