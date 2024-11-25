import { Phone, PhoneCreateInput, SendVerificationTextInput, Success, ValidateVerificationTextInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { setupPhoneVerificationCode, validatePhoneVerificationCode } from "../../auth/phone";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
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
            await RequestService.get().rateLimit({ maxUser: 10, req });
            return createOneHelper({ info, input, objectType, req });
        },
        sendVerificationText: async (_, { input }, { req }) => {
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            await setupPhoneVerificationCode(input.phoneNumber, userId);
            return { __typename: "Success" as const, success: true };
        },
        validateVerificationText: async (_, { input }, { req }) => {
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            const verified = await validatePhoneVerificationCode(input.phoneNumber, userId, input.verificationCode);
            if (!verified)
                throw new CustomError("0139", "CannotVerifyPhoneCode");
            return { __typename: "Success" as const, success: true };
        },
    },
};
