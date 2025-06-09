import { type Bookmark, type BookmarkCreateInput, type BookmarkSearchInput, type BookmarkSearchResult, type BookmarkUpdateInput, type FindByIdInput, VisibilityType } from "@vrooli/shared";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsBookmark = {
    findOne: ApiEndpoint<FindByIdInput, Bookmark>;
    findMany: ApiEndpoint<BookmarkSearchInput, BookmarkSearchResult>;
    createOne: ApiEndpoint<BookmarkCreateInput, Bookmark>;
    updateOne: ApiEndpoint<BookmarkUpdateInput, Bookmark>;
}

export const bookmark: EndpointsBookmark = createStandardCrudEndpoints({
    objectType: "Bookmark",
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
            rateLimit: RateLimitPresets.MEDIUM,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
        updateOne: {
            rateLimit: RateLimitPresets.LOW,
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
});
