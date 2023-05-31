import { gql } from "apollo-server-express";
import { EndpointsPhone, PhoneEndpoints } from "../logic";

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

export const resolvers: {
    Mutation: EndpointsPhone["Mutation"];
} = {
    ...PhoneEndpoints,
};
