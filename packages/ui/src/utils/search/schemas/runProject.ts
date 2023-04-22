import { RunProjectSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { runProjectFindMany } from "../../../api/generated/endpoints/runProject_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunProject"),
    containers: [], //TODO
    fields: [], //TODO
});

export const runProjectSearchParams = () => toParams(runProjectSearchSchema(), runProjectFindMany, RunProjectSortBy, RunProjectSortBy.DateStartedDesc);
