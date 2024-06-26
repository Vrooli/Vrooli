import { quizQuestion_findMany, quizQuestion_findOne } from "../generated";
import { QuizQuestionEndpoints } from "../logic/quizQuestion";
import { setupRoutes } from "./base";

export const QuizQuestionRest = setupRoutes({
    "/quizQuestion/:id": {
        get: [QuizQuestionEndpoints.Query.quizQuestion, quizQuestion_findOne],
    },
    "/quizQuestions": {
        get: [QuizQuestionEndpoints.Query.quizQuestions, quizQuestion_findMany],
    },
});
