import { gql } from "apollo-server-express";
import { EmailEndpoints, EndpointsEmail } from "../logic/email";

export const typeDef = gql`
    input EmailCreateInput {
        id: ID!
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

export const resolvers: {
    Mutation: EndpointsEmail["Mutation"];
} = {
    ...EmailEndpoints,
};
