import { ViewSearchInput, ViewSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads.js";
import { RequestService } from "../../auth/request.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsView = {
    findMany: ApiEndpoint<ViewSearchInput, ViewSearchResult>;
}

const objectType = "View";
export const view: EndpointsView = {
    findMany: async ({ input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 2000, req });
        return readManyHelper({ info, input, objectType, req, additionalQueries: { byId: userData.id } });
    },
};
