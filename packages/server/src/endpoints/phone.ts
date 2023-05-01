import { Phone, PhoneCreateInput, SendVerificationTextInput, Success } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper } from "../actions";
import { setupVerificationCode } from "../auth";
import { rateLimit } from "../middleware";
import { CreateOneResult, GQLEndpoint } from "../types";

export const typeDef = gql`
    input PhoneCreateInput {
        phoneNumber: String!
    }
    input SendVerificationTextInput {
        phoneNumber: String!
    }

    type Phone {
        id: ID!
        phoneNumber: String!
        verified: Boolean!
    }

    extend type Mutation {
        phoneCreate(input: PhoneCreateInput!): Phone!
        sendVerificationText(input: SendVerificationTextInput!): Success!
    }
`;

const objectType = "Phone";
export const resolvers: {
    Mutation: {
        phoneCreate: GQLEndpoint<PhoneCreateInput, CreateOneResult<Phone>>;
        sendVerificationText: GQLEndpoint<SendVerificationTextInput, Success>;
    }
} = {
    Mutation: {
        phoneCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        sendVerificationText: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            await setupVerificationCode(input.phoneNumber, prisma, req.languages);
            return { __typename: "Success" as const, success: true };
        },
    },
};
