import { ReportSortBy } from "@local/consts";
import { reportFindMany } from "../../../api/generated/endpoints/report_findMany";
import { toParams } from "./base";
import { searchFormLayout } from "./common";
export const reportSearchSchema = () => ({
    formLayout: searchFormLayout("SearchReport"),
    containers: [],
    fields: [],
});
export const reportSearchParams = () => toParams(reportSearchSchema(), reportFindMany, ReportSortBy, ReportSortBy.DateCreatedDesc);
//# sourceMappingURL=report.js.map