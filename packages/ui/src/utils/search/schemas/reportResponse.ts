import { ReportResponseSortBy } from "@shared/consts";
import { reportResponseFindMany } from "api/generated/endpoints/reportResponse";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchReportResponse'),
    containers: [], //TODO
    fields: [], //TODO
})

export const reportResponseSearchParams = () => toParams(reportResponseSearchSchema(), reportResponseFindMany, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);