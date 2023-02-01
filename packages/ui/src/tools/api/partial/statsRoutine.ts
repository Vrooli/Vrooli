import { StatsRoutine } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsRoutine: GqlPartial<StatsRoutine> = {
    __typename: 'StatsRoutine',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        runsStarted: true,
        runsCompleted: true,
        runCompletionTimeAverageInPeriod: true,
    },
}