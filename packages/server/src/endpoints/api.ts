import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ApiSortBy, FindByIdInput, Api, ApiSearchInput, ApiCreateInput, ApiUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ApiSortBy {
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

    input ApiCreateInput {
        id: ID!
        permissions: String
        userConnect: ID
        organizationConnect: ID
        parentConnect: ID
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        versionsCreate: [ApiVersionCreateInput!]
        labelsConnect: [ID!]
        labelsCreate: [LabelCreateInput!]
    }
    input ApiUpdateInput {
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
    type Api {
        id: ID!
        created_at: Date!
        updated_at: Date!
        permissions: String!
        createdBy: User
        owner: Owner
        parent: Api
        tags: [Tag!]!
        versions: [ApiVersion!]!
        versionsCount: Int!
        labels: [Label!]!
        stars: Int!
        views: Int!
        score: Int!
        issues: [Issue!]!
        pullRequests: [PullRequest!]!
        pullRequestsCount: Int!
        stats: [StatsApi!]!
        questions: [Question!]!
        questionsCount: Int!
        transfers: [Transfer!]!
        transfersCount: Int!
        starredBy: [User!]!
        you: ApiYou!
    }

    type ApiYou {
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

    input ApiSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        createdById: ID
        maxScore: Int
        maxStars: Int
        minScore: Int
        minStars: Int
        ownedByUserId: ID
        ownedByOrganizationId: ID
        parentId: ID
        languages: [String!]
        ids: [ID!]
        searchString: String
        sortBy: ApiSortBy
        tags: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ApiSearchResult {
        pageInfo: PageInfo!
        edges: [ApiEdge!]!
    }

    type ApiEdge {
        cursor: String!
        node: Api!
    }

    extend type Query {
        api(input: FindByIdInput!): Api
        apis(input: ApiSearchInput!): ApiSearchResult!
    }

    extend type Mutation {
        apiCreate(input: ApiCreateInput!): Api!
        apiUpdate(input: ApiUpdateInput!): Api!
    }
`

const objectType = 'Api';
export const resolvers: {
    ApiSortBy: typeof ApiSortBy;
    Query: {
        api: GQLEndpoint<FindByIdInput, FindOneResult<Api>>;
        apis: GQLEndpoint<ApiSearchInput, FindManyResult<Api>>;
    },
    Mutation: {
        apiCreate: GQLEndpoint<ApiCreateInput, CreateOneResult<Api>>;
        apiUpdate: GQLEndpoint<ApiUpdateInput, UpdateOneResult<Api>>;
    }
} = {
    ApiSortBy,
    Query: {
        api: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        apis: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        apiCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        apiUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}