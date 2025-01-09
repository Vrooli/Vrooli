import { StatsRoutine } from "@local/shared";
import { ApiPartial } from "../types";

export const statsRoutine: ApiPartial<StatsRoutine> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverage: true,
        runContextSwitchesAverage: true,
    },
};
