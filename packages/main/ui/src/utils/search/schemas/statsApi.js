import { StatsApiSortBy } from "@local/consts";
import { statsApiFindMany } from "../../../api/generated/endpoints/statsApi_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsApiSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsApi"),
    containers: [],
    fields: [],
});
export const statsApiSearchParams = () => toParams(statsApiSearchSchema(), statsApiFindMany, StatsApiSortBy, StatsApiSortBy.PeriodStartAsc);
//# sourceMappingURL=statsApi.js.map