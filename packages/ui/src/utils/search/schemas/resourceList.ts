import { endpointGetResourceList, endpointGetResourceLists, ResourceListSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const resourceListSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchResourceList"),
    containers: [], //TODO
    fields: [], //TODO
});

export const resourceListSearchParams = () => toParams(resourceListSearchSchema(), endpointGetResourceLists, endpointGetResourceList, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc);
