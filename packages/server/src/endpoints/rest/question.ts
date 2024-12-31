import { endpointsQuestion } from "@local/shared";
import { question_create, question_findMany, question_findOne, question_update } from "../generated";
import { QuestionEndpoints } from "../logic/question";
import { setupRoutes } from "./base";

export const QuestionRest = setupRoutes([
    [endpointsQuestion.findOne, QuestionEndpoints.Query.question, question_findOne],
    [endpointsQuestion.updateOne, QuestionEndpoints.Mutation.questionUpdate, question_update],
    [endpointsQuestion.findMany, QuestionEndpoints.Query.questions, question_findMany],
    [endpointsQuestion.createOne, QuestionEndpoints.Mutation.questionCreate, question_create],
]);
