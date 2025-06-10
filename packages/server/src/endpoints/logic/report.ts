import { type FindByPublicIdInput, type Report, type ReportCreateInput, type ReportSearchInput, type ReportSearchResult, type ReportUpdateInput } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsReport = {
    findOne: ApiEndpoint<FindByPublicIdInput, Report>;
    findMany: ApiEndpoint<ReportSearchInput, ReportSearchResult>;
    createOne: ApiEndpoint<ReportCreateInput, Report>;
    updateOne: ApiEndpoint<ReportUpdateInput, Report>;
}

export const report: EndpointsReport = createStandardCrudEndpoints({
    objectType: "Report",
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
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
