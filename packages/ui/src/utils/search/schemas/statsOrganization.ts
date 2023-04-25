import { StatsOrganizationSortBy } from "@local/shared";
import { statsOrganizationFindMany } from "api/generated/endpoints/statsOrganization_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsOrganizationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsOrganization'),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsOrganizationSearchParams = () => toParams(statsOrganizationSearchSchema(), statsOrganizationFindMany, StatsOrganizationSortBy, StatsOrganizationSortBy.PeriodStartAsc);