import { endpointsBookmarkList } from "@local/shared";
import { bookmarkList_create, bookmarkList_findMany, bookmarkList_findOne, bookmarkList_update } from "../generated";
import { BookmarkListEndpoints } from "../logic/bookmarkList";
import { setupRoutes } from "./base";

export const BookmarkListRest = setupRoutes([
    [endpointsBookmarkList.findOne, BookmarkListEndpoints.Query.bookmarkList, bookmarkList_findOne],
    [endpointsBookmarkList.findMany, BookmarkListEndpoints.Query.bookmarkLists, bookmarkList_findMany],
    [endpointsBookmarkList.createOne, BookmarkListEndpoints.Mutation.bookmarkListCreate, bookmarkList_create],
    [endpointsBookmarkList.updateOne, BookmarkListEndpoints.Mutation.bookmarkListUpdate, bookmarkList_update],
]);
