import { StatsRoutine } from "@local/shared";
import { GqlPartial } from "../types";

export const statsRoutine: GqlPartial<StatsRoutine> = {
    __typename: 'StatsRoutine',
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
}