import { endpointGetStatsQuiz, FormSchema, StatsQuizSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsQuizSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStatsQuiz"),
    containers: [], //TODO
    elements: [], //TODO
});

export const statsQuizSearchParams = () => toParams(statsQuizSearchSchema(), endpointGetStatsQuiz, undefined, StatsQuizSortBy, StatsQuizSortBy.PeriodStartAsc);
