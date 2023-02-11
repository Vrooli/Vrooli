import { RunProjectSortBy } from "@shared/consts";
import { runProjectFindMany } from "api/generated/endpoints/runProject";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runProjectSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunProjects', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const runProjectSearchParams = (lng: string) => toParams(runProjectSearchSchema(lng), runProjectFindMany, RunProjectSortBy, RunProjectSortBy.DateStartedDesc);