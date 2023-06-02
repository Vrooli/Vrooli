import { reportResponse_create, reportResponse_findMany, reportResponse_findOne, reportResponse_update } from "@local/shared";
import { ReportResponseEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ReportResponseRest = setupRoutes({
    "/reportResponse/:id": {
        get: [ReportResponseEndpoints.Query.reportResponse, reportResponse_findOne],
        put: [ReportResponseEndpoints.Mutation.reportResponseUpdate, reportResponse_update],
    },
    "/reportResponses": {
        get: [ReportResponseEndpoints.Query.reportResponses, reportResponse_findMany],
    },
    "/reportResponse": {
        post: [ReportResponseEndpoints.Mutation.reportResponseCreate, reportResponse_create],
    },
});
