import { QuestionSortBy } from "@shared/consts";
import { questionFindMany } from "api/generated/endpoints/question";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const questionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchQuestion', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const questionSearchParams = (lng: string) => toParams(questionSearchSchema(lng), questionFindMany, QuestionSortBy, QuestionSortBy.ScoreDesc)