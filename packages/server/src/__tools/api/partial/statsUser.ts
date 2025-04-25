import { StatsUser } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsUser: ApiPartial<StatsUser> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        apisCreated: true,
        codesCreated: true,
        codesCompleted: true,
        codeCompletionTimeAverage: true,
        projectsCreated: true,
        projectsCompleted: true,
        projectCompletionTimeAverage: true,
        routinesCreated: true,
        routinesCompleted: true,
        routineCompletionTimeAverage: true,
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
    },
};
