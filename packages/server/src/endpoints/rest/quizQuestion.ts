import { quizQuestion_findMany, quizQuestion_findOne } from "@local/shared";
import { QuizQuestionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const QuizQuestionRest = setupRoutes({
    "/quizQuestion/:id": {
        get: [QuizQuestionEndpoints.Query.quizQuestion, quizQuestion_findOne],
    },
    "/quizQuestions": {
        get: [QuizQuestionEndpoints.Query.quizQuestions, quizQuestion_findMany],
    },
});
