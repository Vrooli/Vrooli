import { endpointGetQuiz, endpointGetQuizzes, QuizSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchQuiz"),
    containers: [], //TODO
    elements: [], //TODO
});

export const quizSearchParams = () => toParams(quizSearchSchema(), endpointGetQuizzes, endpointGetQuiz, QuizSortBy, QuizSortBy.BookmarksDesc);
