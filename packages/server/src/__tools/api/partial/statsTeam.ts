import { type StatsTeam } from "@local/shared";
import { type ApiPartial } from "../types.js";

export const statsTeam: ApiPartial<StatsTeam> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        resources: true,
        members: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
    },
};
