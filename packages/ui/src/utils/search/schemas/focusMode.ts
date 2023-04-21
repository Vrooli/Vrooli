import { FocusModeSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { focusModeFindMany } from "../../api/generated/endpoints/focusMode_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const focusModeSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchFocusMode'),
    containers: [], //TODO
    fields: [], //TODO
})

export const focusModeSearchParams = () => toParams(focusModeSearchSchema(), focusModeFindMany, FocusModeSortBy, FocusModeSortBy.EventStartAsc)