import { CopyInput, CopyResult, lowercaseFirstLetter } from "@local/shared";
import { copyHelper } from "../../actions/copies";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsCopy = {
    Mutation: {
        copy: ApiEndpoint<CopyInput, CopyResult>;
    }
}

export const CopyEndpoints: EndpointsCopy = {
    Mutation: {
        copy: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 500, req });
            const result = await copyHelper({ info, input, objectType: input.objectType, req });
            return { __typename: "CopyResult" as const, [lowercaseFirstLetter(input.objectType)]: result };
        },
    },
};
