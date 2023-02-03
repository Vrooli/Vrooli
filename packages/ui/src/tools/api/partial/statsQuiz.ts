import { StatsQuiz } from "@shared/consts";
import { GqlPartial } from "../types";

export const statsQuiz: GqlPartial<StatsQuiz> = {
    __typename: 'StatsQuiz',
    full: {
        id: true,
        created_at: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        timesStarted: true,
        timesPassed: true,
        timesFailed: true,
        scoreAverage: true,
        completionTimeAverage: true,
    },
}