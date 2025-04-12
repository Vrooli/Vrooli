import { endpointsFocusMode, FocusModeSortBy, FormSchema } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function focusModeSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchFocusMode"),
        containers: [],
        elements: [],
    } as const;
}

export function focusModeSearchParams() {
    return toParams(focusModeSearchSchema(), endpointsFocusMode, FocusModeSortBy, FocusModeSortBy.NameAsc);
}
