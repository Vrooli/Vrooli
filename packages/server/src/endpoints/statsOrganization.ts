import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';

export const typeDef = gql`
    enum StatsOrganizationSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsOrganizationSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsOrganizationSortBy
        take: Int
    }
    type StatsOrganizationSearchResult {
        pageInfo: PageInfo!
        edges: [StatsOrganizationEdge!]!
    }
    type StatsOrganizationEdge {
        cursor: String!
        node: StatsOrganization!
    }

    type StatsOrganization {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        apis: Int!
        members: Int!
        notes: Int!
        projects: Int!
        routines: Int!
        smartContracts: Int!
        standards: Int!
    }

    type Query {
        statsOrganization(input: StatsOrganizationSearchInput!): StatsOrganizationSearchResult!
    }
 `

const objectType = 'StatsOrganization';
export const resolvers: {
    Query: {
        statsOrganization: GQLEndpoint<any, any>;
    },
} = {
    Query: {
        statsOrganization: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}