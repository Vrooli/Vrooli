import { ReportResponseSortBy } from "@local/consts";
import { reportResponseFindMany } from "../../../api/generated/endpoints/reportResponse_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const reportResponseSearchSchema = () => ({
    formLayout: searchFormLayout("SearchReportResponse"),
    containers: [],
    fields: [],
});
export const reportResponseSearchParams = () => toParams(reportResponseSearchSchema(), reportResponseFindMany, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc);
//# sourceMappingURL=reportResponse.js.map