import { Award, AwardSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { RequestService } from "../../auth/request";
import { ApiEndpoint, FindManyResult } from "../../types";

export type EndpointsAward = {
    findMany: ApiEndpoint<AwardSearchInput, FindManyResult<Award>>;
}

const objectType = "Award";
export const award: EndpointsAward = {
    findMany: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req });
    },
};
