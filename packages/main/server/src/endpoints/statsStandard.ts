import { StatsStandardSearchInput, StatsStandardSearchResult } from "@local/shared;";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { rateLimit } from "../middleware";
import { GQLEndpoint } from "../types";

export const typeDef = gql`
    enum StatsStandardSortBy {
        PeriodStartAsc
        PeriodStartDesc
    }

    input StatsStandardSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsStandardSortBy
        take: Int
    }
    type StatsStandardSearchResult {
        pageInfo: PageInfo!
        edges: [StatsStandardEdge!]!
    }
    type StatsStandardEdge {
        cursor: String!
        node: StatsStandard!
    }

    type StatsStandard {
        id: ID!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        linksToInputs: Int!
        linksToOutputs: Int!
    }

    type Query {
        statsStandard(input: StatsStandardSearchInput!): StatsStandardSearchResult!
    }
 `;

const objectType = "StatsStandard";
export const resolvers: {
    Query: {
        statsStandard: GQLEndpoint<StatsStandardSearchInput, StatsStandardSearchResult>;
    },
} = {
    Query: {
        statsStandard: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
