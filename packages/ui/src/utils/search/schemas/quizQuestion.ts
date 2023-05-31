import { quizQuestionFindMany, QuizQuestionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuizQuestion"),
    containers: [], //TODO
    fields: [], //TODO
});

export const quizQuestionSearchParams = () => toParams(quizQuestionSearchSchema(), quizQuestionFindMany, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc);
