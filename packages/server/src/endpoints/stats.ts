import { gql } from 'apollo-server-express';
import { StatisticsPageInput, StatisticsPageResult } from './types';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { CustomError } from '../events/error';

export const typeDef = gql`

    input StatisticsPageInput {
        searchString: String!
        take: Int
    }

    type StatisticsTimeFrame {
        organizations: [Int!]!
        projects: [Int!]!
        routines: [Int!]!
        standards: [Int!]!
        users: [Int!]!
    }

    type StatisticsPageResult {
        daily: StatisticsTimeFrame!
        weekly: StatisticsTimeFrame!
        monthly: StatisticsTimeFrame!
        yearly: StatisticsTimeFrame!
        allTime: StatisticsTimeFrame!
    }
 
    type Query {
        statisticsPage(input: StatisticsPageInput!): StatisticsPageResult!
    }
 `

export const resolvers: {
    Query: {
        statisticsPage: GQLEndpoint<StatisticsPageInput, StatisticsPageResult>;
    },
} = {
    Query: {
        statisticsPage: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            // Query current stats
            // Read historical stats from file
            throw new CustomError('0326', 'NotImplemented', req.languages);
        },
    },
}