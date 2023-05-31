import { StatsApiSearchInput, StatsApiSearchResult } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export const typeDef = gql`
    enum StatsApiSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsApiSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsApiSortBy
        take: Int
    }
    type StatsApiSearchResult {
        pageInfo: PageInfo!
        edges: [StatsApiEdge!]!
    }
    type StatsApiEdge {
        cursor: String!
        node: StatsApi!
    }

    type StatsApi {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        calls: Int!
        routineVersions: Int!
    }

    type Query {
        statsApi(input: StatsApiSearchInput!): StatsApiSearchResult!
    }
 `;

const objectType = "StatsApi";
export const resolvers: {
    Query: {
        statsApi: GQLEndpoint<StatsApiSearchInput, StatsApiSearchResult>;
    },
} = {
    Query: {
        statsApi: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
