import { StatsApiSortBy } from "@shared/consts";
import { statsApiFindMany } from "api/generated/endpoints/statsApi";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsApiSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsApi', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsApiSearchParams = (lng: string) => toParams(statsApiSearchSchema(lng), statsApiFindMany, StatsApiSortBy, StatsApiSortBy.DateUpdatedDesc);