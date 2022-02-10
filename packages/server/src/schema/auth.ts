// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { gql } from 'apollo-server-express';
import { CODE, COOKIE, emailLogInSchema, emailSignUpSchema, passwordSchema, emailRequestPasswordChangeSchema, ROLES, AccountStatus } from '@local/shared';
import { CustomError } from '../error';
import { generateNonce, randomString, verifySignedMessage } from '../auth/walletAuth';
import { generateSessionToken } from '../auth/auth.js';
import { IWrap, RecursivePartial } from '../types';
import { WalletCompleteInput, DeleteOneInput, EmailLogInInput, EmailSignUpInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, WalletInitInput, Session, Success, WalletComplete } from './types';
import { GraphQLResolveInfo } from 'graphql';
import { Context } from '../context';
import { UserModel, userSessioner } from '../models';
import { hasProfanity } from '../utils/censor';

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
        confirmPassword: String!
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
        signedPayload: String!
    }

    type WalletComplete {
        session: Session
        firstLogIn: Boolean!
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
        walletComplete(input: WalletCompleteInput!): WalletComplete!
        walletRemove(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Mutation: {
        emailLogIn: async (_parent: undefined, { input }: IWrap<EmailLogInInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<Session> => {
            // Validate arguments with schema
            emailLogInSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (!email) throw new CustomError(CODE.BadCredentials);
            // Find user
            let user = await prisma.user.findUnique({ where: { id: email.userId ?? '' } });
            if (!user) throw new CustomError(CODE.InternalError, 'User not found');
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
            emailSignUpSchema.validateSync(input, { abortEarly: false });
            // Find user role to give to new user
            const roles = await prisma.role.findMany({ select: { id: true, title: true } });
            const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;
            if (!actorRoleId) throw new CustomError(CODE.ErrorUnknown);
            // Check for censored words
            if (hasProfanity(input.username)) throw new CustomError(CODE.BannedWord);
            // Check if email exists
            const existingEmail = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (existingEmail) throw new CustomError(CODE.EmailInUse);
            // Check if username exists
            let existingUser = await prisma.user.findUnique({ where: { username: input.username } });
            if (existingUser) throw new CustomError(CODE.UsernameInUse);
            // Create user object
            const user = await prisma.user.create({
                data: {
                    username: input.username,
                    password: UserModel(prisma).hashPassword(input.password),
                    theme: input.theme,
                    status: AccountStatus.Unlocked,
                    emails: {
                        create: [
                            { emailAddress: input.email },
                        ]
                    },
                    roles: {
                        create: [{ role: { connect: { id: actorRoleId } } }]
                    }
                }
            });
            if (!user) throw new CustomError(CODE.ErrorUnknown);
            // Create session from user object
            const session = UserModel(prisma).toSession(user);
            console.log('session', session);
            // Set up session token
            await generateSessionToken(res, session);
            // Send verification email
            await UserModel(prisma).setupVerificationCode(input.email);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_parent: undefined, { input }: IWrap<EmailRequestPasswordChangeInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Validate input format
            console.log('emailRequestPasswordChange', input);
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            console.log('emai', email);
            if (!email) throw new CustomError(CODE.EmailNotFound);
            // Find user
            let user = await prisma.user.findUnique({ where: { id: email.userId ?? '' } });
            if (!user) throw new CustomError(CODE.NoUser);
            // Generate and send password reset code
            const success = await UserModel(prisma).setupPasswordReset(user);
            return { success };
        },
        emailResetPassword: async (_parent: undefined, { input }: IWrap<EmailResetPasswordInput>, { prisma }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Validate input format
            passwordSchema.validateSync(input.newPassword, { abortEarly: false });
            // Find user
            let user = await prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    status: true,
                    theme: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                    emails: { select: { emailAddress: true } },
                    roles: { select: { role: { select: { title: true } } } }
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
                where: { id: user.id as unknown as string },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: UserModel(prisma).hashPassword(input.newPassword)
                }
            })
            // Return session
            return userSessioner().toSession(user);
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
            return { success: true };
        },
        validateSession: async (_parent: undefined, _args: undefined, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // If session is expired
            if (!req.userId || !Array.isArray(req.roles) || req.roles.length === 0) {
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
            const userData = await prisma.user.findUnique({
                where: { id: req.userId },
                select: {
                    id: true,
                    status: true,
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
            console.log('userData', userData);
            if (userData) return UserModel(prisma).toSession(userData);
            // If user data failed to fetch, clear session and return error
            res.clearCookie(COOKIE.Session);
            throw new CustomError(CODE.ErrorUnknown);
        },
        /**
         * Starts handshake for establishing trust between backend and user wallet
         * @returns Nonce that wallet must sign and send to walletComplete endpoint
         */
        walletInit: async (_parent: undefined, { input }: IWrap<WalletInitInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<string> => {
            // Generate nonce for handshake
            const nonce = await generateNonce(input.nonceDescription as string | undefined);
            // Find existing wallet data in database
            let walletData = await prisma.wallet.findUnique({
                where: { publicAddress: input.publicAddress },
                select: {
                    id: true,
                    verified: true,
                    userId: true,
                }
            });
            // If wallet exists, update with new nonce
            if (walletData) {
                console.log('walletData exists', walletData);
                await prisma.wallet.update({
                    where: { id: walletData.id },
                    data: {
                        nonce: nonce,
                        nonceCreationTime: new Date().toISOString(),
                    }
                })
            }
            // If wallet data doesn't exist, create
            if (!walletData) {
                console.log('creating wallet', input.publicAddress);
                walletData = await prisma.wallet.create({
                    data: {
                        publicAddress: input.publicAddress,
                        nonce: nonce,
                        nonceCreationTime: new Date().toISOString(),
                    },
                    select: {
                        id: true,
                        verified: true,
                        userId: true,
                    }
                })
            }
            return nonce;
        },
        // Verify that signed message from user wallet has been signed by the correct public address
        walletComplete: async (_parent: undefined, { input }: IWrap<WalletCompleteInput>, { prisma, res }: Context, _info: GraphQLResolveInfo): Promise<WalletComplete> => {
            // Find wallet with public address
            const walletData = await prisma.wallet.findUnique({
                where: { publicAddress: input.publicAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    userId: true,
                    user: {
                        select: { id: true, theme: true }
                    }
                }
            });
            // If wallet doesn't exist, return error
            if (!walletData) throw new CustomError(CODE.InvalidArgs);
            // If nonce expired, return error
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) throw new CustomError(CODE.NonceExpired)
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.publicAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified) throw new CustomError(CODE.Unauthorized);
            let userData = walletData.user;
            // If user doesn't exist, create new user
            let firstLogIn = false;
            if (!userData?.id) {
                firstLogIn = true;
                const roles = await prisma.role.findMany({ select: { id: true, title: true } });
                const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;
                userData = await prisma.user.create({
                    data: {
                        username: `user${randomString(8)}`,
                        roles: {
                            create: [{ role: { connect: { id: actorRoleId } } }]
                        }
                    },
                    select: { id: true, theme: true }
                });
            }
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
                id: userData.id,
                roles: [ROLES.Actor],
                theme: userData.theme ?? 'light',
            }
            // Add session token to return payload
            await generateSessionToken(res, session);
            return { session, firstLogIn } as WalletComplete;
        },
        walletRemove: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // TODO Must deleting your own
            // TODO must keep at least one wallet per user
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
    }
}