import { ReportResponseSortBy } from "@shared/consts";
import { reportResponseFindMany } from "api/generated/endpoints/reportResponse";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportResponseSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchReportResponses', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const reportResponseSearchParams = (lng: string) => toParams(reportResponseSearchSchema(lng), reportResponseFindMany, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);