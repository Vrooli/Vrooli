import { endpointGetQuizQuestionResponse, endpointGetQuizQuestionResponses, QuizQuestionResponseSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionResponseSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuizQuestionResponse"),
    containers: [], //TODO
    elements: [], //TODO
});

export const quizQuestionResponseSearchParams = () => toParams(quizQuestionResponseSearchSchema(), endpointGetQuizQuestionResponses, endpointGetQuizQuestionResponse, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc);
