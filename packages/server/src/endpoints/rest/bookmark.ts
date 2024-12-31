import { endpointsBookmark } from "@local/shared";
import { bookmark_create, bookmark_findMany, bookmark_findOne, bookmark_update } from "../generated";
import { BookmarkEndpoints } from "../logic/bookmark";
import { setupRoutes } from "./base";

export const BookmarkRest = setupRoutes([
    [endpointsBookmark.findOne, BookmarkEndpoints.Query.bookmark, bookmark_findOne],
    [endpointsBookmark.findMany, BookmarkEndpoints.Query.bookmarks, bookmark_findMany],
    [endpointsBookmark.createOne, BookmarkEndpoints.Mutation.bookmarkCreate, bookmark_create],
    [endpointsBookmark.updateOne, BookmarkEndpoints.Mutation.bookmarkUpdate, bookmark_update],
]);
