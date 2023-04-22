import { ViewSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { viewFindMany } from "../../../api/generated/endpoints/view_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const viewSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchView"),
    containers: [], //TODO
    fields: [], //TODO
});

export const viewSearchParams = () => toParams(viewSearchSchema(), viewFindMany, ViewSortBy, ViewSortBy.LastViewedDesc);
