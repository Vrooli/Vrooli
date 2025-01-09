import { AwardSearchInput, AwardSearchResult } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsAward = {
    findMany: ApiEndpoint<AwardSearchInput, AwardSearchResult>;
}

const objectType = "Award";
export const award: EndpointsAward = {
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
