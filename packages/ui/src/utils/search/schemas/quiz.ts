import { QuizSortBy } from "@local/shared";
import { quizFindMany } from "api/generated/endpoints/quiz_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuiz"),
    containers: [], //TODO
    fields: [], //TODO
});

export const quizSearchParams = () => toParams(quizSearchSchema(), quizFindMany, QuizSortBy, QuizSortBy.BookmarksDesc);
