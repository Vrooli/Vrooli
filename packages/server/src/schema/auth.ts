// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { gql } from 'apollo-server-express';
import { CODE, COOKIE, logInSchema, passwordSchema, requestPasswordChangeSchema, ROLES, signUpSchema } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateNonce, verifySignedMessage } from '../auth/walletAuth';
import { generateGuestToken, generateUserToken } from '../auth/auth.js';
import { IWrap } from '../types';
import { CompleteValidateWalletInput, DeleteOneInput, InitValidateWalletInput, LogInInput, RequestPasswordChangeInput, ResetPasswordInput, SignUpInput, User } from './types';
import { GraphQLResolveInfo } from 'graphql';
import { Context } from '../context';
import { UserModel } from '../models';
import pkg from '@prisma/client';
const { AccountStatus } = pkg;

const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

export const typeDef = gql`
    enum AccountStatus {
        DELETED
        UNLOCKED
        SOFT_LOCKED
        HARD_LOCKED
    }

    input InitValidateWalletInput {
        publicAddress: String!
        nonceDescription: String
    }

    input CompleteValidateWalletInput {
        publicAddress: String!
        signedMessage: String!
    }

    input LogInInput {
        email: String
        password: String
        verificationCode: String
    }

    input SignUpInput {
        username: String!
        pronouns: String
        email: String!
        theme: String!
        marketingEmails: Boolean!
        password: String!
    }

    input RequestPasswordChangeInput {
        email: String!
    }

    input ResetPasswordInput {
        id: ID!
        code: String!
        newPassword: String!
    }

    type Wallet {
        id: ID!
        publicAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    extend type Mutation {
        completeValidateWallet(input: CompleteValidateWalletInput!): Boolean!
        initValidateWallet(input: InitValidateWalletInput!): String!
        enterAsGuest: Boolean!
        logIn(input: LogInInput): User!
        logOut: Boolean
        removeWallet(input: DeleteOneInput!): Boolean!
        requestPasswordChange(input: RequestPasswordChangeInput!): Boolean
        resetPassword(input: ResetPasswordInput!): User!
        signUp(input: SignUpInput!): User!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Mutation: {
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
            await generateUserToken(res, walletData.userId);
            return true;
        },
        enterAsGuest: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Set up session token
            await generateGuestToken(res);
            return true;
        },
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
        logIn: async (_parent: undefined, { input }: IWrap<LogInInput | null>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // If email and password are not provided, attempt to log in using session cookie
            if (!Boolean(input?.email) && !Boolean(input?.password)) {
                // If session is expired
                if (!Array.isArray(req.roles) || req.roles.length === 0) {
                    res.clearCookie(COOKIE.Session);
                    throw new CustomError(CODE.SessionExpired);
                }
                const userData = await UserModel(prisma).findById({ id: req.userId ?? '' }, info);
                if (userData) return userData;
                // If user data failed to fetch, clear session and return error
                res.clearCookie(COOKIE.Session);
                throw new CustomError(CODE.ErrorUnknown);
            }
            // Validate arguments with schema
            const validateError = await validateArgs(logInSchema, input);
            if (validateError) throw validateError;
            // Get user
            let user = await UserModel(prisma).findByEmail(input?.email as string);
            // Check for password in database, if doesn't exist, send a password reset link
            // TODO fix this. Can't have password in type, but need password returned
            // if (!Boolean(user.password)) {
            //     await UserModel(prisma).setupPasswordReset(user);
            //     throw new CustomError(CODE.MustResetPassword);
            // }
            // Validate verification code, if supplied
            if (Boolean(input?.verificationCode)) {
                user = await UserModel(prisma).validateVerificationCode(user, input?.verificationCode as string);
            }
            user = await UserModel(prisma).logIn(input?.password as string, user, info);
            if (Boolean(user)) {
                // Set session token
                await generateUserToken(res, user.id);
                return user;
            } else {
                throw new CustomError(CODE.BadCredentials);
            }
        },
        logOut: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            res.clearCookie(COOKIE.Session);
            return true;
        },
        removeWallet: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { req }: any, _info: GraphQLResolveInfo): Promise<boolean> => {
            // TODO Must deleting your own
            // TODO must keep at least one wallet per user
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        requestPasswordChange: async (_parent: undefined, { input }: IWrap<RequestPasswordChangeInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Validate input format
            const validateError = await validateArgs(requestPasswordChangeSchema, input);
            if (validateError) throw validateError;
            // Find user in database
            const user = await UserModel(prisma).findByEmail(input.email);
            // Generate and send password reset code
            return await UserModel(prisma).setupPasswordReset(user);
        },
        resetPassword: async (_parent: undefined, { input }: IWrap<ResetPasswordInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, input.newPassword);
            if (validateError) throw validateError;
            // Find user in database
            let user = await prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    status: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } }
                }
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // If code is invalid
            if (!UserModel(prisma).validateCode(input.code, user.resetPasswordCode ?? '', user.lastResetPasswordReqestAttempt as Date)) {
                // Generate and send new code
                await UserModel(prisma).setupPasswordReset(user);
                // Return error
                throw new CustomError(CODE.InvalidResetCode);
            }
            // Remove request data from user, and set new password
            await prisma.user.update({
                where: { id: user.id },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: UserModel(prisma).hashPassword(input.newPassword)
                }
            })
            // Return user data
            return await UserModel(prisma).findById({ id: user.id }, info) as User;
        },
        signUp: async (_parent: undefined, { input }: IWrap<SignUpInput>, { prisma, res }: Context, info: GraphQLResolveInfo): Promise<User> => {
            // Validate input format
            const validateError = await validateArgs(signUpSchema, input);
            if (validateError) throw validateError;
            // Find user role to give to new user
            const actorRole = await prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) throw new CustomError(CODE.ErrorUnknown);
            // Create user object
            const user = await UserModel(prisma).upsertUser({
                username: input.username,
                password: UserModel(prisma).hashPassword(input.password),
                theme: input.theme,
                status: AccountStatus.UNLOCKED,
                emails: [{ emailAddress: input.email }],
                roles: [actorRole]
            }, info);
            // Set up session token
            await generateUserToken(res, user.id);
            // Send verification email
            await UserModel(prisma).setupVerificationCode(user);
            // Return user data
            return await UserModel(prisma).findById({ id: user.id }, info) as User;
        },
    }
}