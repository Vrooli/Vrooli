import { type Email, type EmailCreateInput, type SendVerificationEmailInput, type Success } from "@vrooli/shared";
import { PasswordAuthService } from "../../auth/email.js";
import { RequestService } from "../../auth/request.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

export type EndpointsEmail = {
    createOne: ApiEndpoint<EmailCreateInput, Email>;
    verify: ApiEndpoint<SendVerificationEmailInput, Success>;
}

export const email: EndpointsEmail = createStandardCrudEndpoints({
    objectType: "Email",
    endpoints: {
        createOne: {
            rateLimit: { maxUser: 10 },
            permissions: { hasWriteAuthPermissions: true },
        },
    },
    customEndpoints: {
        verify: async (wrapped, { req }) => {
            const input = wrapped?.input;
            const { id: userId, publicId: userPublicId } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 50, req });
            RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
            await PasswordAuthService.setupEmailVerificationCode(input.emailAddress, userId, userPublicId, req.session.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
});
