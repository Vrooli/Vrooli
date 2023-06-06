import { Email, EmailCreateInput, SendVerificationEmailInput, Success } from "@local/shared";
import { createHelper } from "../../actions";
import { setupVerificationCode } from "../../auth";
import { rateLimit } from "../../middleware";
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
            return createHelper({ info, input, objectType, prisma, req });
        },
        sendVerificationEmail: async (_, { input }, { prisma, req }) => {
            await rateLimit({ maxUser: 50, req });
            await setupVerificationCode(input.emailAddress, prisma, req.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};
