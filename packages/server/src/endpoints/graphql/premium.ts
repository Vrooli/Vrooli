import { gql } from "apollo-server-express";

export const typeDef = gql`
    type Premium {
        id: ID!
        credits: Int!
        customPlan: String
        enabledAt: Date
        expiresAt: Date
        isActive: Boolean!
    }
`;

export const resolvers = {};
