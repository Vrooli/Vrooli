import { CopyInput, CopyResult, lowercaseFirstLetter } from "@local/shared";
import { copyHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export type EndpointsCopy = {
    Mutation: {
        copy: GQLEndpoint<CopyInput, CopyResult>;
    }
}

export const CopyEndpoints: EndpointsCopy = {
    Mutation: {
        copy: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            const result = await copyHelper({ info, input, objectType: input.objectType, prisma, req });
            return { __typename: "CopyResult" as const, [lowercaseFirstLetter(input.objectType)]: result };
        },
    },
};
