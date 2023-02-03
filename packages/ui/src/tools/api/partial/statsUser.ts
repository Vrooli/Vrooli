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
        projectsCompletionTimeAverage: true,
        quizzesPassed: true,
        quizzesFailed: true,
        routines: true,
        routinesCompleted: true,
        routinesCompletionTimeAverage: true,
        runsStarted: true,
        runsCompleted: true,
        runsCompletionTimeAverage: true,
        smartContractsCreated: true,
        smartContractsCompleted: true,
        smartContractsCompletionTimeAverage: true,
        standardsCreated: true,
        standardsCompleted: true,
        standardsCompletionTimeAverage: true,
    },
}