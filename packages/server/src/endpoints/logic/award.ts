import { Award, AwardSearchInput } from "@local/shared";
import { readManyHelper } from "../../actions/reads";
import { rateLimit } from "../../middleware/rateLimit";
import { FindManyResult, GQLEndpoint } from "../../types";

export type EndpointsAward = {
    Query: {
        awards: GQLEndpoint<AwardSearchInput, FindManyResult<Award>>;
    },
}

const objectType = "Award";
export const AwardEndpoints: EndpointsAward = {
    Query: {
        awards: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};
