import { type FindByIdInput, type ReportResponse, type ReportResponseCreateInput, type ReportResponseSearchInput, type ReportResponseSearchResult, type ReportResponseUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReportResponse = {
    findOne: ApiEndpoint<FindByIdInput, ReportResponse>;
    findMany: ApiEndpoint<ReportResponseSearchInput, ReportResponseSearchResult>;
    createOne: ApiEndpoint<ReportResponseCreateInput, ReportResponse>;
    updateOne: ApiEndpoint<ReportResponseUpdateInput, ReportResponse>;
}

export const reportResponse: EndpointsReportResponse = createStandardCrudEndpoints({
    objectType: "ReportResponse",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        createOne: {
            rateLimit: RateLimitPresets.STRICT,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
