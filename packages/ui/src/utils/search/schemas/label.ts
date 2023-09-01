import { endpointGetLabel, endpointGetLabels, LabelSortBy } from "@local/shared";
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

export const labelSearchParams = () => toParams(labelSearchSchema(), endpointGetLabels, endpointGetLabel, LabelSortBy, LabelSortBy.DateCreatedDesc);
