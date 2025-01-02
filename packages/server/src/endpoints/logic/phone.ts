import { Phone, PhoneCreateInput, SendVerificationTextInput, Success, ValidateVerificationTextInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { setupPhoneVerificationCode, validatePhoneVerificationCode } from "../../auth/phone";
import { RequestService } from "../../auth/request";
import { CustomError } from "../../events/error";
import { ApiEndpoint, CreateOneResult } from "../../types";

export type EndpointsPhone = {
    createOne: ApiEndpoint<PhoneCreateInput, CreateOneResult<Phone>>;
    verify: ApiEndpoint<SendVerificationTextInput, Success>;
    validate: ApiEndpoint<ValidateVerificationTextInput, Success>;
}

const objectType = "Phone";
export const phone: EndpointsPhone = {
    createOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return createOneHelper({ info, input, objectType, req });
    },
    verify: async (_, { input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        await setupPhoneVerificationCode(input.phoneNumber, userId);
        return { __typename: "Success" as const, success: true };
    },
    validate: async (_, { input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        const verified = await validatePhoneVerificationCode(input.phoneNumber, userId, input.verificationCode);
        if (!verified)
            throw new CustomError("0139", "CannotVerifyPhoneCode");
        return { __typename: "Success" as const, success: true };
    },
};
