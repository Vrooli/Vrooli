import { endpointGetTag, endpointGetTags, TagSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const tagSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchTag"),
    containers: [], //TODO
    elements: [], //TODO
});

export const tagSearchParams = () => toParams(tagSearchSchema(), endpointGetTags, endpointGetTag, TagSortBy, TagSortBy.EmbedTopDesc);
