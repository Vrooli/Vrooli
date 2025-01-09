import { endpointsLabel, FormSchema, LabelSortBy } from "@local/shared";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout } from "./common";

export function labelSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchLabel"),
        containers: [
            languagesContainer(),
        ],
        elements: [
            ...languagesFields(),
        ],
    };
}

export function labelSearchParams() {
    return toParams(labelSearchSchema(), endpointsLabel, LabelSortBy, LabelSortBy.DateCreatedDesc);
}
