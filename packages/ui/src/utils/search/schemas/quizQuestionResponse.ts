import { endpointGetQuizQuestionResponses, QuizQuestionResponseSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuizQuestionResponse"),
    containers: [], //TODO
    fields: [], //TODO
});

export const quizQuestionResponseSearchParams = () => toParams(quizQuestionResponseSearchSchema(), endpointGetQuizQuestionResponses, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc);
