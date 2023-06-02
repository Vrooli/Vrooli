import { quizQuestionResponse_findMany, quizQuestionResponse_findOne } from "@local/shared";
import { QuizQuestionResponseEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const QuizQuestionResponseRest = setupRoutes({
    "/quizQuestionResponse/:id": {
        get: [QuizQuestionResponseEndpoints.Query.quizQuestionResponse, quizQuestionResponse_findOne],
    },
    "/quizQuestionResponses": {
        get: [QuizQuestionResponseEndpoints.Query.quizQuestionResponses, quizQuestionResponse_findMany],
    },
});
