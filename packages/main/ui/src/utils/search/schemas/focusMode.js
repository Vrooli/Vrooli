import { FocusModeSortBy } from "@local/consts";
import { focusModeFindMany } from "../../../api/generated/endpoints/focusMode_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const focusModeSearchSchema = () => ({
    formLayout: searchFormLayout("SearchFocusMode"),
    containers: [],
    fields: [],
});
export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), focusModeFindMany, FocusModeSortBy, FocusModeSortBy.EventStartAsc);
//# sourceMappingURL=focusMode.js.map