import { QuizQuestionResponseSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { quizQuestionResponseFindMany } from "../../../api/generated/endpoints/quizQuestionResponse_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const quizQuestionResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchQuizQuestionResponse"),
    containers: [], //TODO
    fields: [], //TODO
});

export const quizQuestionResponseSearchParams = () => toParams(quizQuestionResponseSearchSchema(), quizQuestionResponseFindMany, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc);
