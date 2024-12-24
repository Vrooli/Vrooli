import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsBookmarkList = {
    Query: {
        bookmarkList: ApiEndpoint<FindByIdInput, FindOneResult<BookmarkList>>;
        bookmarkLists: ApiEndpoint<BookmarkListSearchInput, FindManyResult<BookmarkList>>;
    },
    Mutation: {
        bookmarkListCreate: ApiEndpoint<BookmarkListCreateInput, CreateOneResult<BookmarkList>>;
        bookmarkListUpdate: ApiEndpoint<BookmarkListUpdateInput, UpdateOneResult<BookmarkList>>;
    }
}

const objectType = "BookmarkList";
export const BookmarkListEndpoints: EndpointsBookmarkList = {
    Query: {
        bookmarkList: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        bookmarkLists: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        bookmarkListCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        bookmarkListUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
