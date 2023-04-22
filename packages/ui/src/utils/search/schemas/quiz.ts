import { QuizSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { quizFindMany } from "../../../api/generated/endpoints/quiz_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuiz"),
    containers: [], //TODO
    fields: [], //TODO
})

export const quizSearchParams = () => toParams(quizSearchSchema(), quizFindMany, QuizSortBy, QuizSortBy.BookmarksDesc)