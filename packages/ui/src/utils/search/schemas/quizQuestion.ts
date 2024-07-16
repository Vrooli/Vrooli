import { endpointGetQuizQuestion, endpointGetQuizQuestions, QuizQuestionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuizQuestion"),
    containers: [], //TODO
    elements: [], //TODO
});

export const quizQuestionSearchParams = () => toParams(quizQuestionSearchSchema(), endpointGetQuizQuestions, endpointGetQuizQuestion, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc);
