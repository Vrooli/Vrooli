import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, PullRequestSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum PullRequestSortBy {
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input PullRequestCreateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        rootId: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input PullRequestUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        organizationId: ID
        userId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [ProjectTranslationCreateInput!]
        translationsUpdate: [ProjectTranslationUpdateInput!]
    }
    type PullRequest {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        handle: String
        isComplete: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        createdBy: User
        forks: [Project!]!
        owner: Owner
        parent: Project
        permissionsProject: ProjectPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        translations: [ProjectTranslation!]!
        wallets: [Wallet!]
    }

    type PullRequestPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input PullRequestTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input PullRequestTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type PullRequestTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input PullRequestSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type PullRequestSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type PullRequestEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        pullRequest(input: FindByIdInput!): PullRequest
        pullRequests(input: PullRequestSearchInput!): PullRequestSearchResult!
    }

    extend type Mutation {
        pullRequestCreate(input: PullRequestCreateInput!): PullRequest!
        pullRequestUpdate(input: PullRequestUpdateInput!): PullRequest!
        pullRequestAccept(input: FindByIdInput!): PullRequest!
        pullRequestReject(input: FindByIdInput!): PullRequest!
    }
`

const objectType = 'PullRequest';
export const resolvers: {
    PullRequestSortBy: typeof PullRequestSortBy;
    Query: {
        pullRequest: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        pullRequests: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        pullRequestCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        pullRequestUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
        pullRequestAccept: GQLEndpoint<FindByIdInput, UpdateOneResult<Label>>;
        pullRequestReject: GQLEndpoint<FindByIdInput, UpdateOneResult<Label>>;
    }
} = {
    PullRequestSortBy,
    Query: {
        pullRequest: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        pullRequests: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        pullRequestCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        pullRequestUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        pullRequestAccept: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        pullRequestReject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        }
    }
}