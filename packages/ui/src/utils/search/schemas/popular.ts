import { endpointGetFeedPopular, PopularSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const popularSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchPopular"),
    containers: [], //TODO
    fields: [], //TODO
});

export const popularSearchParams = () => toParams(popularSearchSchema(), endpointGetFeedPopular, PopularSortBy, PopularSortBy.BookmarksDesc);
