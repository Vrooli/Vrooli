import { LabelSortBy } from "@local/consts";
import { labelFindMany } from "../../../api/generated/endpoints/label_findMany";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout } from "./common";
export const labelSearchSchema = () => ({
    formLayout: searchFormLayout("SearchLabel"),
    containers: [
        languagesContainer(),
    ],
    fields: [
        ...languagesFields(),
    ],
});
export const labelSearchParams = () => toParams(labelSearchSchema(), labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc);
//# sourceMappingURL=label.js.map