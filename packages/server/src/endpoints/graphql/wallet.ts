import { gql } from "apollo-server-express";
import { EndpointsWallet, WalletEndpoints } from "../logic";

export const typeDef = gql`
    input FindHandlesInput {
        organizationId: ID
    }

    input WalletUpdateInput {
        id: ID!
        name: String
    }

    type Wallet {
        id: ID!
        handles: [Handle!]!
        name: String
        publicAddress: String
        stakingAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    type Handle {
        id: ID!
        handle: String!
        wallet: Wallet!
    }

    extend type Query {
        findHandles(input: FindHandlesInput!): [String!]!
    }

    extend type Mutation {
        walletUpdate(input: WalletUpdateInput!): Wallet!
    }
`;

export const resolvers: {
    Query: EndpointsWallet["Query"];
    Mutation: EndpointsWallet["Mutation"];
} = {
    ...WalletEndpoints,
};
