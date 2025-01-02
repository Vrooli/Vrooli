import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, CreateOneResult, FindManyResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsBookmarkList = {
    findOne: ApiEndpoint<FindByIdInput, FindOneResult<BookmarkList>>;
    findMany: ApiEndpoint<BookmarkListSearchInput, FindManyResult<BookmarkList>>;
    createOne: ApiEndpoint<BookmarkListCreateInput, CreateOneResult<BookmarkList>>;
    updateOne: ApiEndpoint<BookmarkListUpdateInput, UpdateOneResult<BookmarkList>>;
}

const objectType = "BookmarkList";
export const bookmarkList: EndpointsBookmarkList = {
    findOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};
