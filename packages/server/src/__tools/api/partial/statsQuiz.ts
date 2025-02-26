import { StatsQuiz } from "@local/shared";
import { ApiPartial } from "../types.js";

export const statsQuiz: ApiPartial<StatsQuiz> = {
    full: {
        id: true,
        periodStart: true,
        periodEnd: true,
        periodType: true,
        timesStarted: true,
        timesPassed: true,
        timesFailed: true,
        scoreAverage: true,
        completionTimeAverage: true,
    },
};
