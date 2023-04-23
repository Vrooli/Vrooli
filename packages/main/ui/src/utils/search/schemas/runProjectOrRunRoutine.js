import { RunProjectOrRunRoutineSortBy } from "@local/consts";
import { runProjectOrRunRoutineFindMany } from "../../../api/generated/endpoints/runProjectOrRunRoutine_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const runProjectOrRunRoutineSearchSchema = () => ({
    formLayout: searchFormLayout("SearchRunProjectOrRunRoutine"),
    containers: [],
    fields: [],
});
export const runProjectOrRunRoutineSearchParams = () => toParams(runProjectOrRunRoutineSearchSchema(), runProjectOrRunRoutineFindMany, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);
//# sourceMappingURL=runProjectOrRunRoutine.js.map