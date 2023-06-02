import { statsQuiz_findMany } from "@local/shared";
import { StatsQuizEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsQuizRest = setupRoutes({
    "/stats/quiz": {
        get: [StatsQuizEndpoints.Query.statsQuiz, statsQuiz_findMany],
    },
});
