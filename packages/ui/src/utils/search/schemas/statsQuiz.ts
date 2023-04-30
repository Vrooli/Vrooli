import { StatsQuizSortBy } from "@local/shared";
import { statsQuizFindMany } from "api/generated/endpoints/statsQuiz_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const statsQuizSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchStatsQuiz"),
    containers: [], //TODO
    fields: [], //TODO
});

export const statsQuizSearchParams = () => toParams(statsQuizSearchSchema(), statsQuizFindMany, StatsQuizSortBy, StatsQuizSortBy.PeriodStartAsc);
