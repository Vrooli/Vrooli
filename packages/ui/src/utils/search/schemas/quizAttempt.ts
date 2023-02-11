import { QuizAttemptSortBy } from "@shared/consts";
import { quizAttemptFindMany } from "api/generated/endpoints/quizAttempt";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizAttemptSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizAttempts', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizAttemptSearchParams = (lng: string) => toParams(quizAttemptSearchSchema(lng), quizAttemptFindMany, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc)