import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';

export const typeDef = gql`
    enum StatsSiteSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input StatsSiteSearchInput {
        after: String
        ids: [ID!]
        periodType: StatPeriodType!
        periodTimeFrame: TimeFrame
        searchString: String
        sortBy: StatsSiteSortBy
        take: Int
    }
    type StatsSiteSearchResult {
        pageInfo: PageInfo!
        edges: [StatsSiteEdge!]!
    }
    type StatsSiteEdge {
        cursor: String!
        node: StatsSite!
    }

    type StatsSite {
        id: ID!
        created_at: Date!
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        activeUsers: Int!                           
        apiCallsPeriod: Int!                         
        apis: Int!                           
        organizations: Int!                        
        projects: Int!                           
        projectsCompleted: Int!                       
        projectsCompletionTimeAverageInPeriod: Float!
        quizzes: Int!                 
        quizzesCompleted: Int!           
        quizScoreAverageInPeriod: Float!            
        routines: Int!                     
        routinesCompleted: Int!                 
        routinesCompletionTimeAverageInPeriod: Float!  
        routinesSimplicityAverage: Float!               
        routinesComplexityAverage: Float!           
        runsStarted: Int!          
        runsCompleted: Int!          
        runsCompletionTimeAverageInPerid: Float!  
        smartContractsCreated: Int!     
        smartContractsCompleted: Int!          
        smartContractsCompletionTimeAverageInPeriod: Float!
        smartContractCalls: Int!     
        standardsCreated: Int!       
        standardsCompleted: Int!              
        standardsCompletionTimeAverageInPeriod: Float!
        verifiedEmails: Int!     
        verifiedWallets: Int!                       
    }

    type Query {
        statsSite(input: StatsSiteSearchInput!): StatsSiteSearchResult!
    }
 `

const objectType = 'StatsSite';
export const resolvers: {
    Query: {
        statsSite: GQLEndpoint<any, any>;
    },
} = {
    Query: {
        statsSite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}