import { QuizSortBy } from "@shared/consts";
import { quizFindMany } from "api/generated/endpoints/quiz";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizzes', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizSearchParams = (lng: string) => toParams(quizSearchSchema(lng), quizFindMany, QuizSortBy, QuizSortBy.StarsDesc)