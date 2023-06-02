import { bookmark_create, bookmark_findMany, bookmark_findOne, bookmark_update } from "@local/shared";
import { BookmarkEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const BookmarkRest = setupRoutes({
    "/bookmark/:id": {
        get: [BookmarkEndpoints.Query.bookmark, bookmark_findOne],
        put: [BookmarkEndpoints.Mutation.bookmarkUpdate, bookmark_update],
    },
    "/bookmarks": {
        get: [BookmarkEndpoints.Query.bookmarks, bookmark_findMany],
    },
    "/bookmark": {
        post: [BookmarkEndpoints.Mutation.bookmarkCreate, bookmark_create],
    },
});
