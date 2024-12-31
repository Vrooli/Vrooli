import { endpointsQuiz } from "@local/shared";
import { quiz_create, quiz_findMany, quiz_findOne, quiz_update } from "../generated";
import { QuizEndpoints } from "../logic/quiz";
import { setupRoutes } from "./base";

export const QuizRest = setupRoutes([
    [endpointsQuiz.findOne, QuizEndpoints.Query.quiz, quiz_findOne],
    [endpointsQuiz.findMany, QuizEndpoints.Query.quizzes, quiz_findMany],
    [endpointsQuiz.createOne, QuizEndpoints.Mutation.quizCreate, quiz_create],
    [endpointsQuiz.updateOne, QuizEndpoints.Mutation.quizUpdate, quiz_update],
]);
