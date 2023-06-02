import { quiz_create, quiz_findMany, quiz_findOne, quiz_update } from "@local/shared";
import { QuizEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const QuizRest = setupRoutes({
    "/quiz/:id": {
        get: [QuizEndpoints.Query.quiz, quiz_findOne],
        put: [QuizEndpoints.Mutation.quizUpdate, quiz_update],
    },
    "/quizzes": {
        get: [QuizEndpoints.Query.quizzes, quiz_findMany],
    },
    "/quiz": {
        post: [QuizEndpoints.Mutation.quizCreate, quiz_create],
    },
});
