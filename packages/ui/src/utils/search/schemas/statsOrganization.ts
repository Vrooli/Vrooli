import { StatsOrganizationSortBy } from "@shared/consts";
import { statsOrganizationFindMany } from "api/generated/endpoints/statsOrganization";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsOrganizationSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchStatsOrganization', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const statsOrganizationSearchParams = (lng: string) => toParams(statsOrganizationSearchSchema(lng), statsOrganizationFindMany, StatsOrganizationSortBy, StatsOrganizationSortBy.DateUpdatedDesc);