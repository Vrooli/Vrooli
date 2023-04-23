import { RunRoutineInputSortBy } from "@local/consts";
import { runRoutineInputFindMany } from "../../../api/generated/endpoints/runRoutineInput_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const runRoutineInputSearchSchema = () => ({
    formLayout: searchFormLayout("SearchRunRoutineInput"),
    containers: [],
    fields: [],
});
export const runRoutineInputSearchParams = () => toParams(runRoutineInputSearchSchema(), runRoutineInputFindMany, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc);
//# sourceMappingURL=runRoutineInput.js.map