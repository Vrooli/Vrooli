import { StatsStandardSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsStandardSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsStandard"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsStandardSearchParams = () => toParams(statsStandardSearchSchema(), "/stats/standard", StatsStandardSortBy, StatsStandardSortBy.PeriodStartAsc);
