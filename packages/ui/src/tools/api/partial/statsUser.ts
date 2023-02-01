import { StatsUser } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsUser: GqlPartial<StatsUser> = {
    __typename: 'StatsUser',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        apis: true,
        organizations: true,
        projects: true,
        projectsCompleted: true,
        projectsCompletionTimeAverageInPeriod: true,
        quizzesPassed: true,
        quizzesFailed: true,
        routines: true,
        routinesCompleted: true,
        routinesCompletionTimeAverageInPeriod: true,
        runsStarted: true,
        runsCompleted: true,
        runsCompletionTimeAverageInPeriod: true,
        smartContractsCreated: true,
        smartContractsCompleted: true,
        smartContractsCompletionTimeAverageInPeriod: true,
        standardsCreated: true,
        standardsCompleted: true,
        standardsCompletionTimeAverageInPeriod: true,
    },
}