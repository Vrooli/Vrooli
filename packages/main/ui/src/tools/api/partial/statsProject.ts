import { StatsProject } from ":/consts";
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
        notes: true,
        organizations: true,
        projects: true,
        routines: true,
        smartContracts: true,
        standards: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
    },
};
