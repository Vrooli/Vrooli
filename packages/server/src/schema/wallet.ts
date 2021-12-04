import { gql } from 'apollo-server-express';
import { CODE, COOKIE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { generateNonce, verifySignedMessage } from '../auth/walletAuth';
import { generateToken } from '../auth/auth.js';

const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

export const typeDef = gql`
    type Wallet {
        id: ID!
        publicAddress: String!
        verified: Boolean!
        customer: Customer
    }

    extend type Mutation {
        addWallet(publicAddress: String!): String!
        verifyWallet(
            publicAddress: String!
            signedMessage: String! 
            nonceWrapperText: String! 
        ): Boolean!
        removeWallets(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Mutation: {
        // Start handshake for establishing trust between backend and user wallet
        // Returns human-readable message, which includes nonce
        addWallet: async (_parent: undefined, args: any, context: any, info: any) => {
            let userData;
            // If not signed in, create new user row
            if (!context.req.customerId) {
                userData = await context.prisma.customer.create();
            }
            // Otherwise, find user data using id in session token 
            else {
                userData = await context.prisma.customer.findUnique({ where: { id: context.req.customerId } });
            }
            if (!userData) return new CustomError(CODE.ErrorUnknown);

            // Find existing wallet data in database
            let walletData = await context.prisma.wallet.findUnique({
                where: { publicAddress: args.publicAddress },
                select: {
                    id: true,
                    verified: true,
                    customerId: true,
                }
            });
            // If wallet data didn't exist, create
            if (!walletData) {
                walletData = await context.prisma.wallet.create({
                    data: { publicAddress: args.publicAddress },
                    select: {
                        id: true,
                        verified: true,
                        customerId: true,
                    }
                })
            }

            // If wallet is either: (1) unverified; or (2) already verified with user, update wallet with nonce and user id
            if (walletData.customerId === userData.id) {
                const nonce = generateNonce();
                await context.prisma.wallet.update({
                    where: { id: walletData.id },
                    data: { 
                        nonce: nonce, 
                        nonceCreationTime: new Date().toISOString(),
                        customerId: userData.id
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
        verifyWallet: async (_parent: undefined, args: any, context: any, info: any) => {
            // Find wallet with public address
            const walletData = await context.prisma.wallet.findUnique({ 
                where: { publicAddress: args.publicAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    customerId: true,
                }
            });

            // Verify wallet data
            if (!walletData) return new CustomError(CODE.InvalidArgs);
            if (!walletData.customerId) return new CustomError(CODE.ErrorUnknown);
            if (!walletData.nonce || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) return new CustomError(CODE.NonceExpired)

            // Concatenate wrapperText with nonce (since user signs human-readable message
            // that looks something like "Please verify this nonce: fufhduafhdasf", rather than the nonce itself).
            // Could store this entire message as the nonce, but then localization would be more difficult
            const nonceWithMessage = `${args.nonceWrapperText}${walletData.nonce}`;
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(args.publicAddress, nonceWithMessage, args.signedMessage);
            if (!walletVerified) return false;

            // Update wallet and remove nonce data
            await context.prisma.wallet.update({
                where: { id: walletData.id },
                data: { 
                    verified: true,
                    nonce: null, 
                    nonceCreationTime: null,
                }
            })
            // Add session token to return payload
            await generateToken(context.res, walletData.customerId);
            return true;
        },
        removeWallets: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must deleting your own
            // TODO must keep at least one wallet per customer
            const specified = await context.prisma.email.findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const customerIds = [...new Set(specified.map((s: any) => s.customerId))];
            if (customerIds.length > 1 || context.req.customerId !== customerIds[0]) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented); //TODO
        }
    }
}