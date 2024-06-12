import { Phone, PhoneCreateInput, SendVerificationTextInput, Success, ValidateVerificationTextInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { setupPhoneVerificationCode, validatePhoneVerificationCode } from "../../auth/phone";
import { assertRequestFrom } from "../../auth/request";
import { CustomError } from "../../events/error";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, GQLEndpoint } from "../../types";

export type EndpointsPhone = {
    Mutation: {
        phoneCreate: GQLEndpoint<PhoneCreateInput, CreateOneResult<Phone>>;
        sendVerificationText: GQLEndpoint<SendVerificationTextInput, Success>;
        validateVerificationText: GQLEndpoint<ValidateVerificationTextInput, Success>;
    }
}

const objectType = "Phone";
export const PhoneEndpoints: EndpointsPhone = {
    Mutation: {
        phoneCreate: async (_, { input }, { req }, info) => {
            await rateLimit({ maxUser: 10, req });
            return createOneHelper({ info, input, objectType, req });
        },
        sendVerificationText: async (_, { input }, { req }) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 25, req });
            await setupPhoneVerificationCode(input.phoneNumber, userId, req.session.languages);
            return { __typename: "Success" as const, success: true };
        },
        validateVerificationText: async (_, { input }, { req }) => {
            const { id: userId } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 25, req });
            const verified = await validatePhoneVerificationCode(input.phoneNumber, userId, input.verificationCode, req.session.languages);
            if (!verified)
                throw new CustomError("0139", "CannotVerifyPhoneCode", req.session.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};
