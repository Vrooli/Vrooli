import { endpointGetFeedPopular, PopularSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const popularSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchPopular"),
    containers: [], //TODO
    elements: [], //TODO
});

export const popularSearchParams = () => toParams(popularSearchSchema(), endpointGetFeedPopular, undefined, PopularSortBy, PopularSortBy.BookmarksDesc);
