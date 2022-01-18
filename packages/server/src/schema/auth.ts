// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { gql } from 'apollo-server-express';
import { CODE, COOKIE, emailLogInSchema, emailSignUpSchema, passwordSchema, emailRequestPasswordChangeSchema, ROLES, AccountStatus } from '@local/shared';
import { CustomError, validateArgs } from '../error';
import { generateNonce, verifySignedMessage } from '../auth/walletAuth';
import { generateSessionToken } from '../auth/auth.js';
import { IWrap, RecursivePartial } from '../types';
import { WalletCompleteInput, DeleteOneInput, EmailLogInInput, EmailSignUpInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, WalletInitInput, Session, Success } from './types';
import { GraphQLResolveInfo } from 'graphql';
import { Context } from '../context';
import { UserModel } from '../models';

const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

export const typeDef = gql`
    enum AccountStatus {
        Deleted
        Unlocked
        SoftLocked
        HardLocked
    }

    input EmailLogInInput {
        email: String
        password: String
        verificationCode: String
    }

    input EmailSignUpInput {
        username: String!
        email: String!
        theme: String!
        marketingEmails: Boolean!
        password: String!
    }

    input EmailRequestPasswordChangeInput {
        email: String!
    }

    input EmailResetPasswordInput {
        id: ID!
        code: String!
        newPassword: String!
    }

    input WalletInitInput {
        publicAddress: String!
        nonceDescription: String
    }

    input WalletCompleteInput {
        publicAddress: String!
        signedMessage: String!
    }

    type Session {
        id: ID
        roles: [String!]!
        theme: String!
    }

    type Wallet {
        id: ID!
        publicAddress: String!
        verified: Boolean!
        user: User
        organization: Organization
    }

    extend type Mutation {
        emailLogIn(input: EmailLogInInput!): Session!
        emailSignUp(input: EmailSignUpInput!): Session!
        emailRequestPasswordChange(input: EmailRequestPasswordChangeInput!): Success!
        emailResetPassword(input: EmailResetPasswordInput!): Session!
        guestLogIn: Session!
        logOut: Success!
        validateSession: Session!
        walletInit(input: WalletInitInput!): String!
        walletComplete(input: WalletCompleteInput!): Session!
        walletRemove(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Mutation: {
        emailLogIn: async (_parent: undefined, { input }: IWrap<EmailLogInInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<Session> => {
            // Validate arguments with schema
            const validateError = await validateArgs(emailLogInSchema, input);
            if (validateError) throw validateError;
            // Get user
            let { user, email } = await UserModel(prisma).findByEmail(input?.email as string);
            // Check for password in database, if doesn't exist, send a password reset link
            if (!Boolean(user.password)) {
                await UserModel(prisma).setupPasswordReset(user);
                throw new CustomError(CODE.MustResetPassword);
            }
            // Validate verification code, if supplied
            if (Boolean(input?.verificationCode)) {
                await UserModel(prisma).validateVerificationCode(email.emailAddress, user.id, input?.verificationCode as string);
            }
            const session = await UserModel(prisma).logIn(input?.password as string, user);
            if (session) {
                // Set session token
                await generateSessionToken(res, session);
                return session;
            } else {
                throw new CustomError(CODE.BadCredentials);
            }
        },
        emailSignUp: async (_parent: undefined, { input }: IWrap<EmailSignUpInput>, { prisma, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Validate input format
            const validateError = await validateArgs(emailSignUpSchema, input);
            if (validateError) throw validateError;
            // Find user role to give to new user
            const actorRole = await prisma.role.findUnique({ where: { title: ROLES.Actor } });
            if (!actorRole) throw new CustomError(CODE.ErrorUnknown);
            // Create user object
            const user = await UserModel(prisma).create({
                username: input.username,
                password: UserModel(prisma).hashPassword(input.password),
                theme: input.theme,
                status: AccountStatus.Unlocked,
                emails: [{ emailAddress: input.email }],
                roles: [{ role: actorRole }]
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // Create session from user object
            const session = UserModel(prisma).toSession(user);
            // Set up session token
            await generateSessionToken(res, session);
            // Send verification email
            await UserModel(prisma).setupVerificationCode(input.email);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_parent: undefined, { input }: IWrap<EmailRequestPasswordChangeInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Validate input format
            const validateError = await validateArgs(emailRequestPasswordChangeSchema, input);
            if (validateError) throw validateError;
            // Find user in database
            const user = await UserModel(prisma).findByEmail(input.email);
            // Generate and send password reset code
            const success = await UserModel(prisma).setupPasswordReset(user);
            return { success };
        },
        emailResetPassword: async (_parent: undefined, { input }: IWrap<EmailResetPasswordInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Validate input format
            const validateError = await validateArgs(passwordSchema, input.newPassword);
            if (validateError) throw validateError;
            // Find user in database
            let user = await UserModel(prisma).findById(
                { id: input.id },
                {
                    id: true,
                    status: true,
                    theme: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { emailAddress: true },
                    roles: { title: true } 
                }
            );
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // // If code is invalid TODO fix this
            // if (!UserModel(prisma).validateCode(input.code, user.resetPasswordCode ?? '', user.lastResetPasswordReqestAttempt as Date)) {
            //     // Generate and send new code
            //     await UserModel(prisma).setupPasswordReset(user);
            //     // Return error
            //     throw new CustomError(CODE.InvalidResetCode);
            // }
            // Remove request data from user, and set new password
            await prisma.user.update({
                where: { id: user.id as unknown as string },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: UserModel(prisma).hashPassword(input.newPassword)
                }
            })
            // Return session
            return UserModel().toSession(user);
        },
        guestLogIn: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Create session
            const session: RecursivePartial<Session> = {
                roles: [ROLES.Guest],
                theme: 'light',
            }
            // Set up session token
            await generateSessionToken(res, session);
            return session;
        },
        logOut: async (_parent: undefined, _args: undefined, { res }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            res.clearCookie(COOKIE.Session);
            return { success: true};
        },
        validateSession: async (_parent: undefined, _args: undefined, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // If session is expired
            if (!Array.isArray(req.roles) || req.roles.length === 0) {
                res.clearCookie(COOKIE.Session);
                throw new CustomError(CODE.SessionExpired);
            }
            console.log('VALIDATE SESSIon', req.roles)
            // If guest, return default session
            if (req.roles.includes(ROLES.Guest)) {
                return {
                    roles: [ROLES.Guest],
                    theme: 'light',
                }
            }
            console.log('verifying by id', req.userId);
            // Otherwise, check if session can be verified from userId
            const userData = await UserModel(prisma).findById(
                { id: req.userId ?? '' },
                { id: true, status: true, theme: true, roles: { title: true } }
            );
            console.log('userData', userData);
            if (userData) return UserModel(prisma).toSession(userData);
            // If user data failed to fetch, clear session and return error
            res.clearCookie(COOKIE.Session);
            throw new CustomError(CODE.ErrorUnknown);
        },
        // Start handshake for establishing trust between backend and user wallet
        // Returns nonce
        walletInit: async (_parent: undefined, { input }: IWrap<WalletInitInput>, { prisma, req }: any, _info: GraphQLResolveInfo): Promise<string> => {
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
        walletComplete: async (_parent: undefined, { input }: IWrap<WalletCompleteInput>, { prisma, res }: any, _info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Find wallet with public address
            const walletData = await prisma.wallet.findUnique({
                where: { publicAddress: input.publicAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    userId: true,
                    user: {
                        select: {
                            theme: true
                        }
                    }
                }
            });

            // Verify wallet data
            if (!walletData) throw new CustomError(CODE.InvalidArgs);
            if (!walletData.userId) throw new CustomError(CODE.ErrorUnknown);
            if (!walletData.nonce || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) throw new CustomError(CODE.NonceExpired)

            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.publicAddress, walletData.nonce, input.signedMessage);
            if (!walletVerified) throw new CustomError(CODE.Unauthorized);

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
            // Create session token
            const session = {
                id: walletData.userId,
                roles: [ROLES.Actor],
                theme: walletData.user?.theme ?? 'light',
            }
            // Add session token to return payload
            await generateSessionToken(res, session);
            console.log('walletcomplete', session)
            return session;
        },
        walletRemove: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { req }: any, _info: GraphQLResolveInfo): Promise<Success> => {
            // TODO Must deleting your own
            // TODO must keep at least one wallet per user
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
    }
}