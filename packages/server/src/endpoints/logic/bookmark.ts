import { type Bookmark, type BookmarkCreateInput, type BookmarkSearchInput, type BookmarkSearchResult, type BookmarkUpdateInput, type FindByIdInput, VisibilityType } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsBookmark = {
    findOne: ApiEndpoint<FindByIdInput, Bookmark>;
    findMany: ApiEndpoint<BookmarkSearchInput, BookmarkSearchResult>;
    createOne: ApiEndpoint<BookmarkCreateInput, Bookmark>;
    updateOne: ApiEndpoint<BookmarkUpdateInput, Bookmark>;
}

const objectType = "Bookmark";
export const bookmark: EndpointsBookmark = {
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
