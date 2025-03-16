import { endpointsStatsRoutine, FormSchema, StatsRoutineSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function statsRoutineSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsRoutine"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsRoutineSearchParams() {
    return toParams(statsRoutineSearchSchema(), endpointsStatsRoutine, StatsRoutineSortBy, StatsRoutineSortBy.PeriodStartAsc);
}
