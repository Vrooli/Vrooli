import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Success, Wallet, WalletUpdateInput } from './types';
import { Context } from '../context';
import { deleteOneHelper, updateHelper, WalletModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`

    input WalletUpdateInput {
        id: ID!
        name: String
    }

    type Wallet {
        id: ID!
        name: String
        publicAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    extend type Mutation {
        walletUpdate(input: WalletUpdateInput!): Wallet!
        walletDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    Mutation: {
        emailUpdate: async (_parent: undefined, { input }: IWrap<WalletUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Wallet>> => {
            return updateHelper(req.userId, input, info, WalletModel(prisma));
        },
        emailDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, WalletModel(prisma));
        }
    }
}