import { StatsUserSortBy } from "@local/consts";
import { statsUserFindMany } from "../../../api/generated/endpoints/statsUser_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsUserSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsUser"),
    containers: [],
    fields: [],
});
export const statsUserSearchParams = () => toParams(statsUserSearchSchema(), statsUserFindMany, StatsUserSortBy, StatsUserSortBy.PeriodStartAsc);
//# sourceMappingURL=statsUser.js.map