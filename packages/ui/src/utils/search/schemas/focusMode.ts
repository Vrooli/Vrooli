import { endpointGetFocusMode, endpointGetFocusModes, FocusModeSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const focusModeSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchFocusMode"),
    containers: [],
    elements: [],
});

export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), endpointGetFocusModes, endpointGetFocusMode, FocusModeSortBy, FocusModeSortBy.NameAsc);
