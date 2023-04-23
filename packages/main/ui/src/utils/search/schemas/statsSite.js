import { StatsSiteSortBy } from "@local/consts";
import { statsSiteFindMany } from "../../../api/generated/endpoints/statsSite_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsSiteSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsSite"),
    containers: [],
    fields: [],
});
export const statsSiteSearchParams = () => toParams(statsSiteSearchSchema(), statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);
//# sourceMappingURL=statsSite.js.map