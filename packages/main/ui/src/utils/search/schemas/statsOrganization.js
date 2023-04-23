import { StatsOrganizationSortBy } from "@local/consts";
import { statsOrganizationFindMany } from "../../../api/generated/endpoints/statsOrganization_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsOrganizationSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsOrganization"),
    containers: [],
    fields: [],
});
export const statsOrganizationSearchParams = () => toParams(statsOrganizationSearchSchema(), statsOrganizationFindMany, StatsOrganizationSortBy, StatsOrganizationSortBy.PeriodStartAsc);
//# sourceMappingURL=statsOrganization.js.map