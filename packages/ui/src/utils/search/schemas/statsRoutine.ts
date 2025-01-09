import { endpointsStatsRoutine, FormSchema, StatsRoutineSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
