import { reportFindMany, ReportSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReport"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reportSearchParams = () => toParams(reportSearchSchema(), reportFindMany, ReportSortBy, ReportSortBy.DateCreatedDesc);
