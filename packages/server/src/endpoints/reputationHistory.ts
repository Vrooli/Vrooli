import { FindByIdInput, ReputationHistory, ReputationHistorySearchInput, ReputationHistorySortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper, readOneHelper } from "../actions";
import { rateLimit } from "../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../types";

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
        amount: Int!
        event: String!
        objectId1: ID
        objectId2: ID
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
`;

const objectType = "ReputationHistory";
export const resolvers: {
    ReputationHistorySortBy: typeof ReputationHistorySortBy;
    Query: {
        reputationHistory: GQLEndpoint<FindByIdInput, FindOneResult<ReputationHistory>>;
        reputationHistories: GQLEndpoint<ReputationHistorySearchInput, FindManyResult<ReputationHistory>>;
    },
} = {
    ReputationHistorySortBy,
    Query: {
        reputationHistory: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reputationHistories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
