import { endpointGetReports, ReportSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReport"),
    containers: [], //TODO
    fields: [], //TODO
});

export const reportSearchParams = () => toParams(reportSearchSchema(), endpointGetReports, ReportSortBy, ReportSortBy.DateCreatedDesc);
