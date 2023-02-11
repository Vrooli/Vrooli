import { ReportSortBy } from "@shared/consts";
import { reportFindMany } from "api/generated/endpoints/report";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchReports', lng),
    containers: [], //TODO
    fields: [], //TODO
})

export const reportSearchParams = (lng: string) => toParams(reportSearchSchema(lng), reportFindMany, ReportSortBy, ReportSortBy.DateCreatedDesc);