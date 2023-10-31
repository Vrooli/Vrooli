import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsBookmarkList = {
    Query: {
        bookmarkList: GQLEndpoint<FindByIdInput, FindOneResult<BookmarkList>>;
        bookmarkLists: GQLEndpoint<BookmarkListSearchInput, FindManyResult<BookmarkList>>;
    },
    Mutation: {
        bookmarkListCreate: GQLEndpoint<BookmarkListCreateInput, CreateOneResult<BookmarkList>>;
        bookmarkListUpdate: GQLEndpoint<BookmarkListUpdateInput, UpdateOneResult<BookmarkList>>;
    }
}

const objectType = "BookmarkList";
export const BookmarkListEndpoints: EndpointsBookmarkList = {
    Query: {
        bookmarkList: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        bookmarkLists: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        bookmarkListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        bookmarkListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, prisma, req });
        },
    },
};
