import { StatsQuizSortBy } from "@shared/consts";
import { statsQuizFindMany } from "api/generated/endpoints/statsQuiz";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsQuizSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsQuiz', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsQuizSearchParams = (lng: string) => toParams(statsQuizSearchSchema(lng), statsQuizFindMany, StatsQuizSortBy, StatsQuizSortBy.DateUpdatedDesc);