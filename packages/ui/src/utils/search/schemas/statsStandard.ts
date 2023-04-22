import { StatsStandardSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { statsStandardFindMany } from "../../../api/generated/endpoints/statsStandard_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsStandardSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsStandard"),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsStandardSearchParams = () => toParams(statsStandardSearchSchema(), statsStandardFindMany, StatsStandardSortBy, StatsStandardSortBy.PeriodStartAsc);