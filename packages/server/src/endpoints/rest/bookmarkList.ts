import { bookmarkList_create, bookmarkList_findMany, bookmarkList_findOne, bookmarkList_update } from "@local/shared";
import { BookmarkListEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const BookmarkListRest = setupRoutes({
    "/bookmarkList/:id": {
        get: [BookmarkListEndpoints.Query.bookmarkList, bookmarkList_findOne],
        put: [BookmarkListEndpoints.Mutation.bookmarkListUpdate, bookmarkList_update],
    },
    "/bookmarkLists": {
        get: [BookmarkListEndpoints.Query.bookmarkLists, bookmarkList_findMany],
    },
    "/bookmarkList": {
        post: [BookmarkListEndpoints.Mutation.bookmarkListCreate, bookmarkList_create],
    },
} as const);
