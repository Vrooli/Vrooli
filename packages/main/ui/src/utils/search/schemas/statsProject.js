import { StatsProjectSortBy } from "@local/consts";
import { statsProjectFindMany } from "../../../api/generated/endpoints/statsProject_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsProjectSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsProject"),
    containers: [],
    fields: [],
});
export const statsProjectSearchParams = () => toParams(statsProjectSearchSchema(), statsProjectFindMany, StatsProjectSortBy, StatsProjectSortBy.PeriodStartAsc);
//# sourceMappingURL=statsProject.js.map