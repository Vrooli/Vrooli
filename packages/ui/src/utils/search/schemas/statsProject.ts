import { StatsProjectSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsProjectSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsProject"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsProjectSearchParams = () => toParams(statsProjectSearchSchema(), "/stats/project", StatsProjectSortBy, StatsProjectSortBy.PeriodStartAsc);
