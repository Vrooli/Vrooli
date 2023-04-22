import { LabelSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { labelFindMany } from "../../../api/generated/endpoints/label_findMany";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout } from "./common";

export const labelSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchLabel"),
    containers: [
        languagesContainer(),
    ],
    fields: [
        ...languagesFields(),
    ]
})

export const labelSearchParams = () => toParams(labelSearchSchema(), labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc);