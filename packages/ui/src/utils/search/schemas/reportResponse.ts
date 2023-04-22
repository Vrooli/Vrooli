import { ReportResponseSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { reportResponseFindMany } from "../../../api/generated/endpoints/reportResponse_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportResponseSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReportResponse"),
    containers: [], //TODO
    fields: [], //TODO
})

export const reportResponseSearchParams = () => toParams(reportResponseSearchSchema(), reportResponseFindMany, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);