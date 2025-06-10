import { type Phone, type PhoneCreateInput, type SendVerificationTextInput, type Success, type ValidateVerificationTextInput } from "@vrooli/shared";
import { setupPhoneVerificationCode, validatePhoneVerificationCode } from "../../auth/phone.js";
import { RequestService } from "../../auth/request.js";
import { CustomError } from "../../events/error.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

export type EndpointsPhone = {
    createOne: ApiEndpoint<PhoneCreateInput, Phone>;
    verify: ApiEndpoint<SendVerificationTextInput, Success>;
    validate: ApiEndpoint<ValidateVerificationTextInput, Success>;
}

export const phone: EndpointsPhone = createStandardCrudEndpoints({
    objectType: "Phone",
    endpoints: {
        createOne: {
            rateLimit: { maxUser: 10 },
            permissions: { hasWriteAuthPermissions: true },
        },
    },
    customEndpoints: {
        verify: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
            await setupPhoneVerificationCode(input.phoneNumber, userId);
            return { __typename: "Success" as const, success: true };
        },
        validate: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 25, req });
            RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
            const verified = await validatePhoneVerificationCode(input.phoneNumber, userId, input.verificationCode);
            if (!verified)
                throw new CustomError("0139", "CannotVerifyPhoneCode");
            return { __typename: "Success" as const, success: true };
        },
    },
});
