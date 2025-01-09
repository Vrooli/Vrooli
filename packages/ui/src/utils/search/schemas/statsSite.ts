import { endpointsStatsSite, FormSchema, StatsSiteSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function statsSiteSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchStatsSite"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function statsSiteSearchParams() {
    return toParams(statsSiteSearchSchema(), endpointsStatsSite, StatsSiteSortBy, StatsSiteSortBy.PeriodStartAsc);
}
