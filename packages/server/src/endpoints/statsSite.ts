import { gql } from 'apollo-server-express';
import { GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper } from '../actions';
import { StatsSiteSearchInput, StatsSiteSearchResult } from '@shared/consts';

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
        searchString: String # Needed to satisfy SearchStringQueryParams, but not used
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
        periodStart: Date!
        periodEnd: Date!
        periodType: StatPeriodType!
        activeUsers: Int!                           
        apiCalls: Int!                         
        apisCreated: Int!                           
        organizationsCreated: Int!                        
        projectsCreated: Int!                           
        projectsCompleted: Int!                       
        projectCompletionTimeAverage: Float!
        quizzesCreated: Int!                 
        quizzesCompleted: Int!           
        routinesCreated: Int!                     
        routinesCompleted: Int!                 
        routineCompletionTimeAverage: Float!  
        routineSimplicityAverage: Float!               
        routineComplexityAverage: Float!           
        runProjectsStarted: Int!          
        runProjectsCompleted: Int!          
        runProjectCompletionTimeAverage: Float! 
        runProjectContextSwitchesAverage: Float! 
        runRoutinesStarted: Int!          
        runRoutinesCompleted: Int!          
        runRoutineCompletionTimeAverage: Float! 
        runRoutineContextSwitchesAverage: Float! 
        smartContractsCreated: Int!     
        smartContractsCompleted: Int!          
        smartContractCompletionTimeAverage: Float!
        smartContractCalls: Int!     
        standardsCreated: Int!       
        standardsCompleted: Int!              
        standardCompletionTimeAverage: Float!
        verifiedEmailsCreated: Int!     
        verifiedWalletsCreated: Int!                       
    }

    type Query {
        statsSite(input: StatsSiteSearchInput!): StatsSiteSearchResult!
    }
 `

const objectType = 'StatsSite';
export const resolvers: {
    Query: {
        statsSite: GQLEndpoint<StatsSiteSearchInput, StatsSiteSearchResult>;
    },
} = {
    Query: {
        statsSite: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
}