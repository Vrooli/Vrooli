import { endpointsStatsQuiz, FormSchema, StatsQuizSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
