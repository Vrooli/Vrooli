import { StatsProjectSortBy } from "@shared/consts";
import { statsProjectFindMany } from "api/generated/endpoints/statsProject_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsProjectSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsProject'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsProjectSearchParams = () => toParams(statsProjectSearchSchema(), statsProjectFindMany, StatsProjectSortBy, StatsProjectSortBy.PeriodStartAsc);