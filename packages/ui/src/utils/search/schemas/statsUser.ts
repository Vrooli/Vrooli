import { StatsUserSortBy } from "@shared/consts";
import { statsUserFindMany } from "api/generated/endpoints/statsUser";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsUserSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsUser', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsUserSearchParams = (lng: string) => toParams(statsUserSearchSchema(lng), statsUserFindMany, StatsUserSortBy, StatsUserSortBy.DateUpdatedDesc);