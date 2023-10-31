import { Email, EmailCreateInput, SendVerificationEmailInput, Success } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { setupVerificationCode } from "../../auth/email";
import { rateLimit } from "../../middleware/rateLimit";
import { CreateOneResult, GQLEndpoint } from "../../types";

export type EndpointsEmail = {
    Mutation: {
        emailCreate: GQLEndpoint<EmailCreateInput, CreateOneResult<Email>>;
        sendVerificationEmail: GQLEndpoint<SendVerificationEmailInput, Success>;
    }
}

const objectType = "Email";
export const EmailEndpoints: EndpointsEmail = {
    Mutation: {
        emailCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ maxUser: 10, req });
            return createOneHelper({ info, input, objectType, prisma, req });
        },
        sendVerificationEmail: async (_, { input }, { prisma, req }) => {
            await rateLimit({ maxUser: 50, req });
            await setupVerificationCode(input.emailAddress, prisma, req.session.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};
