import { StatsUserSortBy } from "@shared/consts";
import { statsUserFindMany } from "api/generated/endpoints/statsUser_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsUserSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsUser'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsUserSearchParams = () => toParams(statsUserSearchSchema(), statsUserFindMany, StatsUserSortBy, StatsUserSortBy.PeriodStartAsc);