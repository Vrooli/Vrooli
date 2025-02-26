import { endpointsFeed, FormSchema, PopularSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function popularSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchPopular"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function popularSearchParams() {
    return toParams(popularSearchSchema(), { findMany: endpointsFeed.popular }, PopularSortBy, PopularSortBy.BookmarksDesc);
}
