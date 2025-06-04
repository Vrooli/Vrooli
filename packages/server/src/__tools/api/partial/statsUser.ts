import { type StatsUser } from "@local/shared";
import { type ApiPartial } from "../types.js";

export const statsUser: ApiPartial<StatsUser> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        resourcesCreatedByType: true,
        resourcesCompletedByType: true,
        resourceCompletionTimeAverageByType: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
        teamsCreated: true,
    },
};
