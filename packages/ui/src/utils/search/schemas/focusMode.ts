import { endpointGetFocusMode, endpointGetFocusModes, FocusModeSortBy, FormSchema } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const focusModeSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchFocusMode"),
    containers: [],
    elements: [],
});

export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), endpointGetFocusModes, endpointGetFocusMode, FocusModeSortBy, FocusModeSortBy.NameAsc);
