import { QuizAttemptSortBy } from "@local/shared";
import { quizAttemptFindMany } from "api/generated/endpoints/quizAttempt_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizAttemptSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchQuizAttempt'),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizAttemptSearchParams = () => toParams(quizAttemptSearchSchema(), quizAttemptFindMany, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc)