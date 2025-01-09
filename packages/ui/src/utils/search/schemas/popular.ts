import { endpointsFeed, FormSchema, PopularSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

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
