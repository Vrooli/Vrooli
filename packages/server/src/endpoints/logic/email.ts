import { Email, EmailCreateInput, SendVerificationEmailInput, Success } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { PasswordAuthService } from "../../auth/email";
import { RequestService } from "../../auth/request";
import { ApiEndpoint } from "../../types";

export type EndpointsEmail = {
    createOne: ApiEndpoint<EmailCreateInput, Email>;
    verify: ApiEndpoint<SendVerificationEmailInput, Success>;
}

const objectType = "Email";
export const email: EndpointsEmail = {
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return createOneHelper({ info, input, objectType, req });
    },
    verify: async ({ input }, { req }) => {
        const { id: userId } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 50, req });
        await PasswordAuthService.setupEmailVerificationCode(input.emailAddress, userId, req.session.languages);
        return { __typename: "Success" as const, success: true };
    },
};
