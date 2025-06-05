import { ViewSortBy, endpointsView, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function viewSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchView"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function viewSearchParams() {
    return toParams(viewSearchSchema(), endpointsView, ViewSortBy, ViewSortBy.LastViewedDesc);
}

