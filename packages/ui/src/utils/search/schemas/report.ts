import { ReportSortBy, endpointsReport, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

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
