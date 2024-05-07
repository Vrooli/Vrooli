import { endpointGetTag, endpointGetTags, TagSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const tagSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchTag"),
    containers: [], //TODO
    fields: [], //TODO
});

export const tagSearchParams = () => toParams(tagSearchSchema(), endpointGetTags, endpointGetTag, TagSortBy, TagSortBy.EmbedTopDesc);
