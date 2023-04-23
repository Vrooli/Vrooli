import { StatsRoutineSortBy } from "@local/consts";
import { statsRoutineFindMany } from "../../../api/generated/endpoints/statsRoutine_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const statsRoutineSearchSchema = () => ({
    formLayout: searchFormLayout("SearchStatsRoutine"),
    containers: [],
    fields: [],
});
export const statsRoutineSearchParams = () => toParams(statsRoutineSearchSchema(), statsRoutineFindMany, StatsRoutineSortBy, StatsRoutineSortBy.PeriodStartAsc);
//# sourceMappingURL=statsRoutine.js.map