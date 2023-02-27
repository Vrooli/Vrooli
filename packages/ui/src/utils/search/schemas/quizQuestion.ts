import { QuizQuestionSortBy } from "@shared/consts";
import { quizQuestionFindMany } from "api/generated/endpoints/quizQuestion_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizQuestion'),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizQuestionSearchParams = () => toParams(quizQuestionSearchSchema(), quizQuestionFindMany, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc)