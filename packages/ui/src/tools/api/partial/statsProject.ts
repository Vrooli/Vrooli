import { StatsProject } from "@local/shared";
import { GqlPartial } from "../types";

export const statsProject: GqlPartial<StatsProject> = {
    __typename: "StatsProject",
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
