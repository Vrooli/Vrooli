import { StatsApiSortBy } from "@local/shared";
import { statsApiFindMany } from "api/generated/endpoints/statsApi_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsApiSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsApi'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsApiSearchParams = () => toParams(statsApiSearchSchema(), statsApiFindMany, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);