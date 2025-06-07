import { type BookmarkList, type BookmarkListCreateInput, type BookmarkListSearchInput, type BookmarkListSearchResult, type BookmarkListUpdateInput, type FindByIdInput, VisibilityType } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsBookmarkList = {
    findOne: ApiEndpoint<FindByIdInput, BookmarkList>;
    findMany: ApiEndpoint<BookmarkListSearchInput, BookmarkListSearchResult>;
    createOne: ApiEndpoint<BookmarkListCreateInput, BookmarkList>;
    updateOne: ApiEndpoint<BookmarkListUpdateInput, BookmarkList>;
}

const objectType = "BookmarkList";
export const bookmarkList: EndpointsBookmarkList = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
};
