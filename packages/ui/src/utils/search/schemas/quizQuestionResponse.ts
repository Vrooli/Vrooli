import { QuizQuestionResponseSortBy } from "@shared/consts";
import { quizQuestionResponseFindMany } from "api/generated/endpoints/quizQuestionResponse";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionResponseSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizQuestionResponse', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizQuestionResponseSearchParams = (lng: string) => toParams(quizQuestionResponseSearchSchema(lng), quizQuestionResponseFindMany, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc)