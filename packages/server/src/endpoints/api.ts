import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ApiSortBy, FindByIdInput, Api, ApiSearchInput, ApiCreateInput, ApiUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ApiSortBy {
        CalledByRoutinesAsc
        CalledByRoutinesDesc
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
        StarsAsc
        StarsDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
        VotesAsc
        VotesDesc
    }

    input ApiCreateInput {
        id: ID!
    }
    input ApiUpdateInput {
        id: ID!
    }
    type Api {
        id: ID!
    }

    type ApiPermission {
        canCopy: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canView: Boolean!
        canVote: Boolean!
    }

    input ApiSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: ApiSortBy
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