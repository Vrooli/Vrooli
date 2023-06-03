import { FocusModeSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const focusModeSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchFocusMode"),
    containers: [],
    fields: [],
});

export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), "/focusModes", FocusModeSortBy, FocusModeSortBy.NameAsc);
