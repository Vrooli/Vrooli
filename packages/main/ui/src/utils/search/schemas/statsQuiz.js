import { StatsQuizSortBy } from "@local/consts";
import { statsQuizFindMany } from "../../../api/generated/endpoints/statsQuiz_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsQuizSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsQuiz"),
    containers: [],
    fields: [],
});
export const statsQuizSearchParams = () => toParams(statsQuizSearchSchema(), statsQuizFindMany, StatsQuizSortBy, StatsQuizSortBy.PeriodStartAsc);
//# sourceMappingURL=statsQuiz.js.map