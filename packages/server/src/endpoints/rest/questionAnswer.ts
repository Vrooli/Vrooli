import { questionAnswer_accept, questionAnswer_create, questionAnswer_findMany, questionAnswer_findOne, questionAnswer_update } from "../generated";
import { QuestionAnswerEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const QuestionAnswerRest = setupRoutes({
    "/questionAnswer/:id": {
        get: [QuestionAnswerEndpoints.Query.questionAnswer, questionAnswer_findOne],
        put: [QuestionAnswerEndpoints.Mutation.questionAnswerUpdate, questionAnswer_update],
    },
    "/questionAnswers": {
        get: [QuestionAnswerEndpoints.Query.questionAnswers, questionAnswer_findMany],
    },
    "/questionAnswer": {
        post: [QuestionAnswerEndpoints.Mutation.questionAnswerCreate, questionAnswer_create],
    },
    "/questionAnswer/:id/accept": {
        put: [QuestionAnswerEndpoints.Mutation.questionAnswerMarkAsAccepted, questionAnswer_accept],
    },
});
