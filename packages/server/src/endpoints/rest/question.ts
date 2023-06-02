import { question_create, question_findMany, question_findOne, question_update } from "@local/shared";
import { QuestionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const QuestionRest = setupRoutes({
    "/question/:id": {
        get: [QuestionEndpoints.Query.question, question_findOne],
        put: [QuestionEndpoints.Mutation.questionUpdate, question_update],
    },
    "/questions": {
        get: [QuestionEndpoints.Query.questions, question_findMany],
    },
    "/question": {
        post: [QuestionEndpoints.Mutation.questionCreate, question_create],
    },
});
