import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { StandardSortBy, ApiVersion, ApiVersionSearchInput, ApiVersionCreateInput, ApiVersionUpdateInput, FindByIdInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum StandardSortBy {
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

    input StandardCreateInput {
        id: ID!
        name: String!
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
        versionsCreate: [StandardVersionCreateInput!]
    }
    input StandardUpdateInput {
        id: ID!
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
        versionsCreate: [StandardVersionCreateInput!]
        versionsUpdate: [StandardVersionUpdateInput!]
        versionsDelete: [ID!]
    }
    type Standard {
        type: GqlModelType!
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        hasCompletedVersion: Boolean!
        isDeleted: Boolean!
        isInternal: Boolean!
        isPrivate: Boolean!
        permissions: String!
        translatedName: String!
        score: Int!
        stars: Int!
        views: Int!
        name: String!
        createdBy: User
        forks: [Standard!]!
        forksCount: Int!
        issues: [Issue!]!
        issuesCount: Int!
        labels: [Label!]!
        owner: Owner
        parent: Standard
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        starredBy: [User!]!
        tags: [Tag!]!
        versions: [StandardVersion!]!
        versionsCount: Int
        you: StandardYou!
    }

    type StandardYou {
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canTransfer: Boolean!
        canView: Boolean!
        canVote: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
    }

    input StandardSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
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
        sortBy: StandardSortBy
        tags: [String!]
        take: Int
        translationLanguagesLatestVersion: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type StandardSearchResult {
        pageInfo: PageInfo!
        edges: [StandardEdge!]!
    }

    type StandardEdge {
        cursor: String!
        node: Standard!
    }

    extend type Query {
        standard(input: FindByIdInput!): Standard
        standards(input: StandardSearchInput!): StandardSearchResult!
    }

    extend type Mutation {
        standardCreate(input: StandardCreateInput!): Standard!
        standardUpdate(input: StandardUpdateInput!): Standard!
    }
`

const objectType = 'Standard';
export const resolvers: {
    StandardSortBy: typeof StandardSortBy;
    Query: {
        standard: GQLEndpoint<FindByIdInput, FindOneResult<ApiVersion>>;
        standards: GQLEndpoint<ApiVersionSearchInput, FindManyResult<ApiVersion>>;
    },
    Mutation: {
        standardCreate: GQLEndpoint<ApiVersionCreateInput, CreateOneResult<ApiVersion>>;
        standardUpdate: GQLEndpoint<ApiVersionUpdateInput, UpdateOneResult<ApiVersion>>;
    }
} = {
    StandardSortBy,
    Query: {
        standard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        standards: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        standardCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        standardUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}