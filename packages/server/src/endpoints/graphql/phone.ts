import { EndpointsPhone, PhoneEndpoints } from "../logic/phone";

export const typeDef = `#graphql
    input PhoneCreateInput {
        id: ID!
        phoneNumber: String!
    }
    input SendVerificationTextInput {
        phoneNumber: String!
    }
    input ValidateVerificationTextInput {
        phoneNumber: String!
        verificationCode: String!
    }

    type Phone {
        id: ID!
        phoneNumber: String!
        verified: Boolean!
    }

    extend type Mutation {
        phoneCreate(input: PhoneCreateInput!): Phone!
        sendVerificationText(input: SendVerificationTextInput!): Success!
        validateVerificationText(input: ValidateVerificationTextInput!): Success!
    }
`;

export const resolvers: {
    Mutation: EndpointsPhone["Mutation"];
} = {
    ...PhoneEndpoints,
};
