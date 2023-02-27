import { QuizQuestionResponseSortBy } from "@shared/consts";
import { quizQuestionResponseFindMany } from "api/generated/endpoints/quizQuestionResponse";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizQuestionResponse'),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizQuestionResponseSearchParams = () => toParams(quizQuestionResponseSearchSchema(), quizQuestionResponseFindMany, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc)