import { Bookmark, BookmarkCreateInput, BookmarkSearchInput, BookmarkUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsBookmark = {
    Query: {
        bookmark: GQLEndpoint<FindByIdInput, FindOneResult<Bookmark>>;
        bookmarks: GQLEndpoint<BookmarkSearchInput, FindManyResult<Bookmark>>;
    },
    Mutation: {
        bookmarkCreate: GQLEndpoint<BookmarkCreateInput, CreateOneResult<Bookmark>>;
        bookmarkUpdate: GQLEndpoint<BookmarkUpdateInput, UpdateOneResult<Bookmark>>;
    }
}

const objectType = "Bookmark";
export const BookmarkEndpoints: EndpointsBookmark = {
    Query: {
        bookmark: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        bookmarks: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        bookmarkCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        bookmarkUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
