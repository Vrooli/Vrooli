import { endpointsReportResponse, FormSchema, ReportResponseSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function reportResponseSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchReportResponse"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function reportResponseSearchParams() {
    return toParams(reportResponseSearchSchema(), endpointsReportResponse, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);
}
