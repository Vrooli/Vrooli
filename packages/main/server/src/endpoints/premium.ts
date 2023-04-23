import { gql } from "apollo-server-express";

export const typeDef = gql`
    type Premium {
        id: ID!
        customPlan: String
        enabledAt: Date
        expiresAt: Date
        isActive: Boolean!
    }
`;

export const resolvers = {};
