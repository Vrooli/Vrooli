import { ViewSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const viewSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchView"),
    containers: [], //TODO
    fields: [], //TODO
});

export const viewSearchParams = () => toParams(viewSearchSchema(), "/views", ViewSortBy, ViewSortBy.LastViewedDesc);
