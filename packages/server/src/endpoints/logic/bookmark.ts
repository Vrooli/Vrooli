import { Bookmark, BookmarkCreateInput, BookmarkSearchInput, BookmarkUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsBookmark = {
    Query: {
        bookmark: ApiEndpoint<FindByIdInput, FindOneResult<Bookmark>>;
        bookmarks: ApiEndpoint<BookmarkSearchInput, FindManyResult<Bookmark>>;
    },
    Mutation: {
        bookmarkCreate: ApiEndpoint<BookmarkCreateInput, CreateOneResult<Bookmark>>;
        bookmarkUpdate: ApiEndpoint<BookmarkUpdateInput, UpdateOneResult<Bookmark>>;
    }
}

const objectType = "Bookmark";
export const BookmarkEndpoints: EndpointsBookmark = {
    Query: {
        bookmark: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        bookmarks: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        bookmarkCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        bookmarkUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
    },
};
