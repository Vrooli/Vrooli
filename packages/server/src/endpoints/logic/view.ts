import { View, ViewSearchInput, ViewSearchResult } from "@local/shared";
import { GraphQLResolveInfo } from "graphql";
import { readManyHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { Context, rateLimit } from "../../middleware";
import { FindManyResult, GQLEndpoint, IWrap } from "../../types";

export type EndpointsView = {
    Query: {
        views: GQLEndpoint<ViewSearchInput, FindManyResult<View>>;
    },
}

const objectType = "View";
export const ViewEndpoints: EndpointsView = {
    Query: {
        views: async (_parent: undefined, { input }: IWrap<ViewSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ViewSearchResult> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { byId: userData.id } });
        },
    },
};
