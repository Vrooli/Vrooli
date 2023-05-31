import { statsApiFindMany, StatsApiSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsApiSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsApi"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsApiSearchParams = () => toParams(statsApiSearchSchema(), statsApiFindMany, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);
