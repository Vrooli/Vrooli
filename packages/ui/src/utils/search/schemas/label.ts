import { LabelSortBy } from "@shared/consts";
import { labelFindMany } from "api/generated/endpoints/label";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout } from "./common";

export const labelSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchLabel', lng),
    containers: [
        languagesContainer(lng),
    ],
    fields: [
        ...languagesFields(lng),
    ]
})

export const labelSearchParams = (lng: string) => toParams(labelSearchSchema(lng), labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc);