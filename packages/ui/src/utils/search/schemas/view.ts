import { endpointGetViews, ViewSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
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
    return toParams(viewSearchSchema(), endpointGetViews, undefined, ViewSortBy, ViewSortBy.LastViewedDesc);
}

