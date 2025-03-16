import { StatsProject } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsProject: ApiPartial<StatsProject> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        directories: true,
        apis: true,
        codes: true,
        notes: true,
        projects: true,
        routines: true,
        standards: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
        teams: true,
    },
};
