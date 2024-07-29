import { endpointGetQuizAttempt, endpointGetQuizAttempts, FormSchema, QuizAttemptSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizAttemptSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuizAttempt"),
    containers: [], //TODO
    elements: [], //TODO
});

export const quizAttemptSearchParams = () => toParams(quizAttemptSearchSchema(), endpointGetQuizAttempts, endpointGetQuizAttempt, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc);
