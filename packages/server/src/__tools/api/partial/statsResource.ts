import { type StatsResource } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const statsResource: ApiPartial<StatsResource> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        references: true,
        referencedBy: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
    },
};
