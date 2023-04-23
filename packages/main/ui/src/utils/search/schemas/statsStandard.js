import { StatsStandardSortBy } from "@local/consts";
import { statsStandardFindMany } from "../../../api/generated/endpoints/statsStandard_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsStandardSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsStandard"),
    containers: [],
    fields: [],
});
export const statsStandardSearchParams = () => toParams(statsStandardSearchSchema(), statsStandardFindMany, StatsStandardSortBy, StatsStandardSortBy.PeriodStartAsc);
//# sourceMappingURL=statsStandard.js.map