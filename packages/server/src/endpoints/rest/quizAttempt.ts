import { quizAttempt_create, quizAttempt_findMany, quizAttempt_findOne, quizAttempt_update } from "../generated";
import { QuizAttemptEndpoints } from "../logic/quizAttempt";
import { setupRoutes } from "./base";

export const QuizAttemptRest = setupRoutes({
    "/quizAttempt/:id": {
        get: [QuizAttemptEndpoints.Query.quizAttempt, quizAttempt_findOne],
        put: [QuizAttemptEndpoints.Mutation.quizAttemptUpdate, quizAttempt_update],
    },
    "/quizAttempts": {
        get: [QuizAttemptEndpoints.Query.quizAttempts, quizAttempt_findMany],
    },
    "/quizAttempt": {
        post: [QuizAttemptEndpoints.Mutation.quizAttemptCreate, quizAttempt_create],
    },
});
