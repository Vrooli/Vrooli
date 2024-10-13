import { EndpointsStatsSite, StatsSiteEndpoints } from "../logic/statsSite";

export const typeDef = `#graphql
    enum StatsSiteSortBy {
        PeriodStartAsc
        PeriodStartDesc
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
        codesCreated: Int!     
        codesCompleted: Int!          
        codeCompletionTimeAverage: Float!
        codeCalls: Int!                  
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
        standardsCreated: Int!       
        standardsCompleted: Int!              
        standardCompletionTimeAverage: Float!
        teamsCreated: Int!                        
        verifiedEmailsCreated: Int!     
        verifiedWalletsCreated: Int!                       
    }

    type Query {
        statsSite(input: StatsSiteSearchInput!): StatsSiteSearchResult!
    }
 `;

export const resolvers: {
    Query: EndpointsStatsSite["Query"];
} = {
    ...StatsSiteEndpoints,
};
