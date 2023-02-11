import { QuizQuestionSortBy } from "@shared/consts";
import { quizQuestionFindMany } from "api/generated/endpoints/quizQuestion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizQuestions', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizQuestionSearchParams = (lng: string) => toParams(quizQuestionSearchSchema(lng), quizQuestionFindMany, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc)