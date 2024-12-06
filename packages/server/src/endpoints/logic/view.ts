import { View, ViewSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsView = {
    Query: {
        views: GQLEndpoint<ViewSearchInput, FindManyResult<View>>;
    },
}

const objectType = "View";
export const ViewEndpoints: EndpointsView = {
    Query: {
        views: async (_parent, { input }, { req }, info) => {
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, req, additionalQueries: { byId: userData.id } });
        },
    },
};
