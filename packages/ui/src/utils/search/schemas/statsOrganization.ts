import { endpointGetStatsOrganization, StatsOrganizationSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsOrganizationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsOrganization"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsOrganizationSearchParams = () => toParams(statsOrganizationSearchSchema(), endpointGetStatsOrganization, undefined, StatsOrganizationSortBy, StatsOrganizationSortBy.PeriodStartAsc);
