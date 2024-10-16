import { EndpointsWallet, WalletEndpoints } from "../logic/wallet";

export const typeDef = `#graphql

    input WalletUpdateInput {
        id: ID!
        name: String
    }

    type Wallet {
        id: ID!
        name: String
        publicAddress: String
        stakingAddress: String!
        verified: Boolean!
        user: User
        team: Team
    }

    extend type Mutation {
        walletUpdate(input: WalletUpdateInput!): Wallet!
    }
`;

export const resolvers: {
    Mutation: EndpointsWallet["Mutation"];
} = {
    ...WalletEndpoints,
};
