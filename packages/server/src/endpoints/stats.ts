import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { CustomError } from '../events/error';
import { StatsSiteSearchInput, StatsSiteSearchResult } from './types';

export const typeDef = gql`
    input StatsSiteSearchInput {
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

    type StatsSiteSearchResult {
        daily: StatisticsTimeFrame!
        weekly: StatisticsTimeFrame!
        monthly: StatisticsTimeFrame!
        yearly: StatisticsTimeFrame!
        allTime: StatisticsTimeFrame!
    }

    type StatsApi {
        id: ID!
    }

    type StatsNote {
        id: ID!
    }

    type StatsOrganization {
        id: ID!
    }

    type StatsProject {
        id: ID!
    }

    type StatsQuiz {
        id: ID!
    }

    type StatsRoutine {
        id: ID!
    }

    type StatsSmartContract {
        id: ID!
    }

    type StatsStandard {
        id: ID!
    }
    
    type StatsUser {
        id: ID!
    }

    type Query {
        statsSite(input: StatsSiteSearchInput!): StatsSiteSearchResult!
    }
 `

export const resolvers: {
    Query: {
        statsSite: GQLEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
    },
} = {
    Query: {
        statsSite: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            // Query current stats
            // Read historical stats from file
            throw new CustomError('0326', 'NotImplemented', req.languages);
        },
    },
}