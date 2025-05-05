import { StatsSite } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsSite: ApiPartial<StatsSite> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        activeUsers: true,
        teamsCreated: true,
        verifiedEmailsCreated: true,
        verifiedWalletsCreated: true,
        resourcesCreatedByType: true,
        resourcesCompletedByType: true,
        resourceCompletionTimeAverageByType: true,
        routineSimplicityAverage: true,
        routineComplexityAverage: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
    },
};
