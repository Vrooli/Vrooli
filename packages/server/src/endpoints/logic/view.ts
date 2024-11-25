import { View, ViewSearchInput, ViewSearchResult } from "@local/shared";
import { GraphQLResolveInfo } from "graphql";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { Context } from "../../middleware";
import { FindManyResult, GQLEndpoint, IWrap } from "../../types";

export type EndpointsView = {
    Query: {
        views: GQLEndpoint<ViewSearchInput, FindManyResult<View>>;
    },
}

const objectType = "View";
export const ViewEndpoints: EndpointsView = {
    Query: {
        views: async (_parent: undefined, { input }: IWrap<ViewSearchInput>, { req }: Context, info: GraphQLResolveInfo): Promise<ViewSearchResult> => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, req, additionalQueries: { byId: userData.id } });
        },
    },
};
