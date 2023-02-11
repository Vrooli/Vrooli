import { QuestionAnswerSortBy } from "@shared/consts";
import { questionAnswerFindMany } from "api/generated/endpoints/questionAnswer";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionAnswerSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuestionAnswers', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const questionAnswerSearchParams = (lng: string) => toParams(questionAnswerSearchSchema(lng), questionAnswerFindMany, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc)