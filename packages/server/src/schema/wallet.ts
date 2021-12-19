import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { generateNonce, verifySignedMessage } from '../auth/walletAuth';
import { generateToken } from '../auth/auth.js';
import { IWrap } from '../types';
import { CompleteValidateWalletInput, DeleteOneInput, InitValidateWalletInput } from './types';
import { GraphQLResolveInfo } from 'graphql';

const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

export const typeDef = gql`
    type Wallet {
        id: ID!
        publicAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    input InitValidateWalletInput {
        publicAddress: String!
        nonceDescription: String
    }

    input CompleteValidateWalletInput {
        publicAddress: String!
        signedMessage: String!
    }

    extend type Mutation {
        initValidateWallet(input: InitValidateWalletInput!): String!
        completeValidateWallet(input: CompleteValidateWalletInput!): Boolean!
        removeWallet(input: DeleteOneInput!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        // Start handshake for establishing trust between backend and user wallet
        // Returns nonce
        initValidateWallet: async (_parent: undefined, { input }: IWrap<InitValidateWalletInput>, { prisma, req }: any, _info: GraphQLResolveInfo): Promise<string> => {
            let userData;
            // If not signed in, create new user row
            if (!req.userId) userData = await prisma.user.create({ data: {} });
            // Otherwise, find user data using id in session token 
            else userData = await prisma.user.findUnique({ where: { id: req.userId } });
            if (!userData) throw new CustomError(CODE.ErrorUnknown);

            // Find existing wallet data in database
            let walletData = await prisma.wallet.findUnique({
                where: { publicAddress: input.publicAddress },
                select: {
                    id: true,
                    verified: true,
                    userId: true,
                }
            });
            // If wallet data didn't exist, create
            if (!walletData) {
                walletData = await prisma.wallet.create({
                    data: { publicAddress: input.publicAddress },
                    select: {
                        id: true,
                        verified: true,
                        userId: true,
                    }
                })
            }

            // If wallet is either: (1) unverified; or (2) already verified with user, update wallet with nonce and user id
            if (!walletData.verified || walletData.userId === userData.id) {
                const nonce = await generateNonce(input.nonceDescription as string | undefined);
                await prisma.wallet.update({
                    where: { id: walletData.id },
                    data: { 
                        nonce: nonce, 
                        nonceCreationTime: new Date().toISOString(),
                        userId: userData.id
                    }
                })
                return nonce;
            }
            // If wallet is verified by another account
            else {
                throw new CustomError(CODE.NotYourWallet);
            }
        },
        // Verify that signed message from user wallet has been signed by the correct public address
        completeValidateWallet: async (_parent: undefined, { input }: IWrap<CompleteValidateWalletInput>, { prisma, res }: any, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Find wallet with public address
            const walletData = await prisma.wallet.findUnique({ 
                where: { publicAddress: input.publicAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    userId: true,
                }
            });

            // Verify wallet data
            if (!walletData) throw new CustomError(CODE.InvalidArgs);
            if (!walletData.userId) throw new CustomError(CODE.ErrorUnknown);
            if (!walletData.nonce || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) throw new CustomError(CODE.NonceExpired)

            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.publicAddress, walletData.nonce, input.signedMessage);
            if (!walletVerified) return false;

            // Update wallet and remove nonce data
            await prisma.wallet.update({
                where: { id: walletData.id },
                data: { 
                    verified: true,
                    lastVerifiedTime: new Date().toISOString(),
                    nonce: null, 
                    nonceCreationTime: null,
                }
            })
            // Add session token to return payload
            await generateToken(res, walletData.userId);
            return true;
        },
        removeWallet: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { req }: any, _info: GraphQLResolveInfo): Promise<boolean> => {
            // TODO Must deleting your own
            // TODO must keep at least one wallet per user
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}