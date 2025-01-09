import { endpointsReport, FormSchema, ReportSortBy } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export function reportSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchReport"),
        containers: [], //TODO
        elements: [], //TODO
    };
}

export function reportSearchParams() {
    return toParams(reportSearchSchema(), endpointsReport, ReportSortBy, ReportSortBy.DateCreatedDesc);
}
