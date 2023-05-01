import { FocusModeSortBy } from "@local/shared";
import { focusModeFindMany } from "api/generated/endpoints/focusMode_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const focusModeSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchFocusMode"),
    containers: [],
    fields: [],
});

export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), focusModeFindMany, FocusModeSortBy, FocusModeSortBy.NameAsc);
