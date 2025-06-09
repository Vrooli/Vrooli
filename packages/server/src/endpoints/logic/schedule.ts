import { type FindByPublicIdInput, type Schedule, type ScheduleCreateInput, type ScheduleSearchInput, type ScheduleSearchResult, type ScheduleUpdateInput, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsSchedule = {
    findOne: ApiEndpoint<FindByPublicIdInput, Schedule>;
    findMany: ApiEndpoint<ScheduleSearchInput, ScheduleSearchResult>;
    createOne: ApiEndpoint<ScheduleCreateInput, Schedule>;
    updateOne: ApiEndpoint<ScheduleUpdateInput, Schedule>;
}

export const schedule: EndpointsSchedule = createStandardCrudEndpoints({
    objectType: "Schedule",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.Own,
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
