import { LabelSortBy } from "@shared/consts";
import { labelFindMany } from "api/generated/endpoints/label";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const labelSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchLabels', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const labelSearchParams = (lng: string) => toParams(labelSearchSchema(lng), labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc);