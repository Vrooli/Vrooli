import { endpointsStatsQuiz, FormSchema, StatsQuizSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function statsQuizSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsQuiz"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsQuizSearchParams() {
    return toParams(statsQuizSearchSchema(), endpointsStatsQuiz, StatsQuizSortBy, StatsQuizSortBy.PeriodStartAsc);
}
