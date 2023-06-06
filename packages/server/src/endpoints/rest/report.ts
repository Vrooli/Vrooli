import { report_create, report_findMany, report_findOne, report_update } from "../generated";
import { ReportEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ReportRest = setupRoutes({
    "/report/:id": {
        get: [ReportEndpoints.Query.report, report_findOne],
        put: [ReportEndpoints.Mutation.reportUpdate, report_update],
    },
    "/reports": {
        get: [ReportEndpoints.Query.reports, report_findMany],
    },
    "/report": {
        post: [ReportEndpoints.Mutation.reportCreate, report_create],
    },
});
