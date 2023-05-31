import { Email, EmailCreateInput, SendVerificationEmailInput, Success } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper } from "../../actions";
import { setupVerificationCode } from "../../auth";
import { rateLimit } from "../../middleware";
import { CreateOneResult, GQLEndpoint } from "../../types";

export const typeDef = gql`
    input EmailCreateInput {
        emailAddress: String!
    }
    input SendVerificationEmailInput {
        emailAddress: String!
    }

    type Email {
        id: ID!
        emailAddress: String!
        verified: Boolean!
    }

    extend type Mutation {
        emailCreate(input: EmailCreateInput!): Email!
        sendVerificationEmail(input: SendVerificationEmailInput!): Success!
    }
`;

const objectType = "Email";
export const resolvers: {
    Mutation: {
        emailCreate: GQLEndpoint<EmailCreateInput, CreateOneResult<Email>>;
        sendVerificationEmail: GQLEndpoint<SendVerificationEmailInput, Success>;
    }
} = {
    Mutation: {
        emailCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        sendVerificationEmail: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            await setupVerificationCode(input.emailAddress, prisma, req.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};
