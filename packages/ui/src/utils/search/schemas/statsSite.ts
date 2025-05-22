import { endpointsStatsSite, type FormSchema, StatsSiteSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
