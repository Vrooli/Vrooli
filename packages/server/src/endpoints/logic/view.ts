import { View, ViewSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsView = {
    findMany: ApiEndpoint<ViewSearchInput, FindManyResult<View>>;
}

const objectType = "View";
export const view: EndpointsView = {
    findMany: async (_parent, { input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, additionalQueries: { byId: userData.id } });
    },
};
