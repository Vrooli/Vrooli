import { StatsSite } from ":/consts";
import { GqlPartial } from "../types";

export const statsSite: GqlPartial<StatsSite> = {
    __typename: "StatsSite",
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        activeUsers: true,
        apiCalls: true,
        apisCreated: true,
        organizationsCreated: true,
        projectsCreated: true,
        projectsCompleted: true,
        projectCompletionTimeAverage: true,
        quizzesCreated: true,
        quizzesCompleted: true,
        routinesCreated: true,
        routinesCompleted: true,
        routineCompletionTimeAverage: true,
        routineSimplicityAverage: true,
        routineComplexityAverage: true,
        runProjectsStarted: true,
        runProjectsCompleted: true,
        runProjectCompletionTimeAverage: true,
        runProjectContextSwitchesAverage: true,
        runRoutinesStarted: true,
        runRoutinesCompleted: true,
        runRoutineCompletionTimeAverage: true,
        runRoutineContextSwitchesAverage: true,
        smartContractsCreated: true,
        smartContractsCompleted: true,
        smartContractCompletionTimeAverage: true,
        smartContractCalls: true,
        standardsCreated: true,
        standardsCompleted: true,
        standardCompletionTimeAverage: true,
        verifiedEmailsCreated: true,
        verifiedWalletsCreated: true,
    },
};
