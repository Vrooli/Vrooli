import { StatsApiSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { statsApiFindMany } from "../../../api/generated/endpoints/statsApi_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsApiSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsApi"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsApiSearchParams = () => toParams(statsApiSearchSchema(), statsApiFindMany, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);
