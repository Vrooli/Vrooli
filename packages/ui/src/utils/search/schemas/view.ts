import { endpointsView, FormSchema, ViewSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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

