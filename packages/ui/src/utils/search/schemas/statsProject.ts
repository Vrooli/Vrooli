import { StatsProjectSortBy } from "@shared/consts";
import { statsProjectFindMany } from "api/generated/endpoints/statsProject";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsProjectSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsProject', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsProjectSearchParams = (lng: string) => toParams(statsProjectSearchSchema(lng), statsProjectFindMany, StatsProjectSortBy, StatsProjectSortBy.DateUpdatedDesc);