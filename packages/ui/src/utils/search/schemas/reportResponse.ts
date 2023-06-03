import { ReportResponseSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReportResponse"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reportResponseSearchParams = () => toParams(reportResponseSearchSchema(), "/reportResponses", ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);
