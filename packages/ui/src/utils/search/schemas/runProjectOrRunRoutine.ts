import { endpointsUnions, FormSchema, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
