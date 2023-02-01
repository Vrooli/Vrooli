import { StatsSite } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsSite: GqlPartial<StatsSite> = {
    __typename: 'StatsSite',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        activeUsers: true,                           
        apiCallsPeriod: true,                         
        apis: true,                           
        organizations: true,                        
        projects: true,                           
        projectsCompleted: true,                       
        projectsCompletionTimeAverageInPeriod: true,
        quizzes: true,                 
        quizzesCompleted: true,           
        quizScoreAverageInPeriod: true,            
        routines: true,                     
        routinesCompleted: true,                 
        routinesCompletionTimeAverageInPeriod: true,  
        routinesSimplicityAverage: true,               
        routinesComplexityAverage: true,           
        runsStarted: true,          
        runsCompleted: true,          
        runsCompletionTimeAverageInPerid: true,  
        smartContractsCreated: true,     
        smartContractsCompleted: true,          
        smartContractsCompletionTimeAverageInPeriod: true,
        smartContractCalls: true,     
        standardsCreated: true,       
        standardsCompleted: true,              
        standardsCompletionTimeAverageInPeriod: true,
        verifiedEmails: true,     
        verifiedWallets: true,        
    },
}