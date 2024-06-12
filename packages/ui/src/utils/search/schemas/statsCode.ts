import { endpointGetStatsCode, StatsCodeSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsCodeSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsCode"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsCodeSearchParams = () => toParams(statsCodeSearchSchema(), endpointGetStatsCode, undefined, StatsCodeSortBy, StatsCodeSortBy.PeriodStartAsc);
