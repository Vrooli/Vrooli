import { endpointGetQuizAttempts, QuizAttemptSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizAttemptSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuizAttempt"),
    containers: [], //TODO
    fields: [], //TODO
});

export const quizAttemptSearchParams = () => toParams(quizAttemptSearchSchema(), endpointGetQuizAttempts, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc);
