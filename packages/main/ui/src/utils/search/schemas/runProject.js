import { RunProjectSortBy } from "@local/consts";
import { runProjectFindMany } from "../../../api/generated/endpoints/runProject_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const runProjectSearchSchema = () => ({
    formLayout: searchFormLayout("SearchRunProject"),
    containers: [],
    fields: [],
});
export const runProjectSearchParams = () => toParams(runProjectSearchSchema(), runProjectFindMany, RunProjectSortBy, RunProjectSortBy.DateStartedDesc);
//# sourceMappingURL=runProject.js.map