import { endpointGetReportResponse, endpointGetReportResponses, FormSchema, ReportResponseSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportResponseSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReportResponse"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reportResponseSearchParams = () => toParams(reportResponseSearchSchema(), endpointGetReportResponses, endpointGetReportResponse, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);
