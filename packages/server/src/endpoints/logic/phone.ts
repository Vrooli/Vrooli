import { Phone, PhoneCreateInput, SendVerificationTextInput, Success } from "@local/shared";
import { createHelper } from "../../actions";
import { setupVerificationCode } from "../../auth";
import { rateLimit } from "../../middleware";
import { CreateOneResult, GQLEndpoint } from "../../types";

export type EndpointsPhone = {
    Mutation: {
        phoneCreate: GQLEndpoint<PhoneCreateInput, CreateOneResult<Phone>>;
        sendVerificationText: GQLEndpoint<SendVerificationTextInput, Success>;
    }
}

const objectType = "Phone";
export const PhoneEndpoints: EndpointsPhone = {
    Mutation: {
        phoneCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        sendVerificationText: async (_, { input }, { prisma, req }) => {
            await rateLimit({ maxUser: 50, req });
            await setupVerificationCode(input.phoneNumber, prisma, req.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};