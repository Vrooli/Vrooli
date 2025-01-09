import { StatsSite } from "@local/shared";
import { ApiPartial } from "../types";

export const statsSite: ApiPartial<StatsSite> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        activeUsers: true,
        apiCalls: true,
        apisCreated: true,
        codesCreated: true,
        codesCompleted: true,
        codeCompletionTimeAverage: true,
        codeCalls: true,
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
        standardsCreated: true,
        standardsCompleted: true,
        standardCompletionTimeAverage: true,
        teamsCreated: true,
        verifiedEmailsCreated: true,
        verifiedWalletsCreated: true,
    },
};
