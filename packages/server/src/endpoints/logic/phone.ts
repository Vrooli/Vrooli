import { Phone, PhoneCreateInput, SendVerificationTextInput, Success, ValidateVerificationTextInput } from "@local/shared";
import { createOneHelper } from "../../actions/creates.js";
import { setupPhoneVerificationCode, validatePhoneVerificationCode } from "../../auth/phone.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsPhone = {
    createOne: ApiEndpoint<PhoneCreateInput, Phone>;
    verify: ApiEndpoint<SendVerificationTextInput, Success>;
    validate: ApiEndpoint<ValidateVerificationTextInput, Success>;
}

const objectType = "Phone";
export const phone: EndpointsPhone = {
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        return createOneHelper({ info, input, objectType, req });
    },
    verify: async ({ input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        await setupPhoneVerificationCode(input.phoneNumber, userId);
        return { __typename: "Success" as const, success: true };
    },
    validate: async ({ input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 25, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        const verified = await validatePhoneVerificationCode(input.phoneNumber, userId, input.verificationCode);
        if (!verified)
            throw new CustomError("0139", "CannotVerifyPhoneCode");
        return { __typename: "Success" as const, success: true };
    },
};
