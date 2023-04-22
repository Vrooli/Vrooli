import { ReportSortBy } from "@shared/consts";
import { FormSchema } from "../../../forms/types";
import { reportFindMany } from "../../../api/generated/endpoints/report_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const reportSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchReport"),
    containers: [], //TODO
    fields: [], //TODO
})

export const reportSearchParams = () => toParams(reportSearchSchema(), reportFindMany, ReportSortBy, ReportSortBy.DateCreatedDesc);