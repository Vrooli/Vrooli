import { LabelSortBy } from "@local/shared";
import { labelFindMany } from "api/generated/endpoints/label_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout } from "./common";

export const labelSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchLabel"),
    containers: [
        languagesContainer(),
    ],
    fields: [
        ...languagesFields(),
    ],
});

export const labelSearchParams = () => toParams(labelSearchSchema(), labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc);
