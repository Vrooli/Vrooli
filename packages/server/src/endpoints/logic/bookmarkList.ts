import { type BookmarkList, type BookmarkListCreateInput, type BookmarkListSearchInput, type BookmarkListSearchResult, type BookmarkListUpdateInput, type FindByIdInput, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsBookmarkList = {
    findOne: ApiEndpoint<FindByIdInput, BookmarkList>;
    findMany: ApiEndpoint<BookmarkListSearchInput, BookmarkListSearchResult>;
    createOne: ApiEndpoint<BookmarkListCreateInput, BookmarkList>;
    updateOne: ApiEndpoint<BookmarkListUpdateInput, BookmarkList>;
}

export const bookmarkList: EndpointsBookmarkList = createStandardCrudEndpoints({
    objectType: "BookmarkList",
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
        findMany: {
            rateLimit: RateLimitPresets.VERY_HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            visibility: VisibilityType.Own,
        },
        createOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
