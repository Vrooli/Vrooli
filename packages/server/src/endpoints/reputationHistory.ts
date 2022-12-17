import { gql } from 'apollo-server-express';
import { FindManyResult, FindOneResult, GQLEndpoint } from '../types';
import { FindByIdInput, Label, LabelSearchInput, ReputationHistorySortBy } from './types';
import { rateLimit } from '../middleware';
import { readManyHelper, readOneHelper } from '../actions';

export const typeDef = gql`
    enum ReputationHistorySortBy {
        AmountAsc
        AmountDesc
        DateCreatedAsc
        DateCreatedDesc
    }

    type ReputationHistory {
        id: ID!
        created_at: Date!
        updated_at: Date!
        amound: Int!
    }

    input ReputationHistorySearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: ReputationHistorySortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ReputationHistorySearchResult {
        pageInfo: PageInfo!
        edges: [ReputationHistoryEdge!]!
    }

    type ReputationHistoryEdge {
        cursor: String!
        node: ReputationHistory!
    }

    extend type Query {
        reputationHistory(input: FindByIdInput!): ReputationHistory
        reputationHistories(input: ReputationHistorySearchInput!): ReputationHistorySearchResult!
    }
`

const objectType = 'ReputationHistory';
export const resolvers: {
    ReputationHistorySortBy: typeof ReputationHistorySortBy;
    Query: {
        reputationHistory: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        reputationHistories: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
} = {
    ReputationHistorySortBy,
    Query: {
        reputationHistory: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        reputationHistories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
}