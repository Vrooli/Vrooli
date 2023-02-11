import { StatsStandardSortBy } from "@shared/consts";
import { statsStandardFindMany } from "api/generated/endpoints/statsStandard";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsStandardSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsStandard', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsStandardSearchParams = (lng: string) => toParams(statsStandardSearchSchema(lng), statsStandardFindMany, StatsStandardSortBy, StatsStandardSortBy.DateUpdatedDesc);