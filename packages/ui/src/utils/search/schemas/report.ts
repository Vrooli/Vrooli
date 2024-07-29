import { endpointGetReport, endpointGetReports, FormSchema, ReportSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchReport"),
    containers: [], //TODO
    elements: [], //TODO
});

export const reportSearchParams = () => toParams(reportSearchSchema(), endpointGetReports, endpointGetReport, ReportSortBy, ReportSortBy.DateCreatedDesc);
