import { CopyInput, CopyResult, lowercaseFirstLetter } from "@local/shared";
import { copyHelper } from "../../actions/copies";
import { rateLimit } from "../../middleware/rateLimit";
import { GQLEndpoint } from "../../types";

export type EndpointsCopy = {
    Mutation: {
        copy: GQLEndpoint<CopyInput, CopyResult>;
    }
}

export const CopyEndpoints: EndpointsCopy = {
    Mutation: {
        copy: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 500, req });
            const result = await copyHelper({ info, input, objectType: input.objectType, prisma, req });
            return { __typename: "CopyResult" as const, [lowercaseFirstLetter(input.objectType)]: result };
        },
    },
};
