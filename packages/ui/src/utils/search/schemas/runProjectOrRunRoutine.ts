import { endpointsUnions, FormSchema, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function runProjectOrRunRoutineSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRunProjectOrRunRoutine"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function runProjectOrRunRoutineSearchParams() {
    return toParams(runProjectOrRunRoutineSearchSchema(), { findMany: endpointsUnions.runProjectOrRunRoutines }, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc);
}
