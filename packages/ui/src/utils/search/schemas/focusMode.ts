import { endpointGetFocusMode, endpointGetFocusModes, FocusModeSortBy, FormSchema } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function focusModeSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchFocusMode"),
        containers: [],
        elements: [],
    } as const;
}

export function focusModeSearchParams() {
    return toParams(focusModeSearchSchema(), endpointGetFocusModes, endpointGetFocusMode, FocusModeSortBy, FocusModeSortBy.NameAsc);
}
