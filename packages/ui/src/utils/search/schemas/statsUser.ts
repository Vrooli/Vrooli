import { StatsUserSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsUserSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsUser"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsUserSearchParams = () => toParams(statsUserSearchSchema(), "/stats/user", StatsUserSortBy, StatsUserSortBy.PeriodStartAsc);
