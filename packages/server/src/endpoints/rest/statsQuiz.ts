import { statsQuiz_findMany } from "../generated";
import { StatsQuizEndpoints } from "../logic/statsQuiz";
import { setupRoutes } from "./base";

export const StatsQuizRest = setupRoutes({
    "/stats/quiz": {
        get: [StatsQuizEndpoints.Query.statsQuiz, statsQuiz_findMany],
    },
});
