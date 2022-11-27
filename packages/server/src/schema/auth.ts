// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { gql } from 'apollo-server-express';
import { emailLogInSchema, emailSignUpSchema, passwordSchema, emailRequestPasswordChangeSchema } from '@shared/validation';
import { COOKIE } from "@shared/consts";
import { CustomError } from '../events/error';
import { generateNonce, randomString, serializedAddressToBech32, verifySignedMessage } from '../auth/wallet';
import { assertRequestFrom, generateSessionJwt, updateSessionTimeZone } from '../auth/request.js';
import { IWrap, RecursivePartial } from '../types';
import { WalletCompleteInput, EmailLogInInput, EmailSignUpInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, WalletInitInput, Session, Success, WalletComplete, ApiKeyStatus, LogOutInput, SwitchCurrentAccountInput } from './types';
import { GraphQLResolveInfo } from 'graphql';
import { Context, rateLimit } from '../middleware';
import { hasProfanity } from '../utils/censor';
import pkg from '@prisma/client';
import { Trigger } from '../events';
import { getUser, hashPassword, logIn, setupPasswordReset, toSession, validateCode, validateVerificationCode } from '../auth';
const { AccountStatus } = pkg;

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
        name: String!
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

    input LogOutInput {
        id: ID
    }

    input SwitchCurrentAccountInput {
        id: ID!
    }

    input ValidateSessionInput {
        timeZone: String
    }

    input WalletInitInput {
        stakingAddress: String!
        nonceDescription: String
    }

    input WalletCompleteInput {
        stakingAddress: String!
        signedPayload: String!
    }

    type WalletComplete {
        firstLogIn: Boolean!
        session: Session
        wallet: Wallet
    }

    type SessionUser {
        handle: String
        hasPremium: Boolean!
        id: String!
        languages: [String]!
        name: String
        theme: String
    }

    type Session {
        isLoggedIn: Boolean!
        timeZone: String
        users: [SessionUser!]
    }

    type ApiKeyStatus {
        creditsUsed: Int!
        creditsTotal: Int!
        locksWhenCreditsUsed: Boolean!
        nextResetAt: Date!
    }

    extend type Mutation {
        apiKeyCreate: String!
        apiKeyDelete: Success!
        apiKeyValidate: Success!
        emailLogIn(input: EmailLogInInput!): Session!
        emailSignUp(input: EmailSignUpInput!): Session!
        emailRequestPasswordChange(input: EmailRequestPasswordChangeInput!): Success!
        emailResetPassword(input: EmailResetPasswordInput!): Session!
        guestLogIn: Session!
        logOut(input: LogOutInput!): Session!
        validateSession(input: ValidateSessionInput!): Session!
        switchCurrentAccount(input: SwitchCurrentAccountInput!): Session!
        walletInit(input: WalletInitInput!): String!
        walletComplete(input: WalletCompleteInput!): WalletComplete!
    }
`

export const resolvers = {
    AccountStatus: AccountStatus,
    Mutation: {
        apiKeyCreate: async (_parent: undefined, { input }: IWrap<any>, { req }: Context, info: GraphQLResolveInfo): Promise<String> => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ info, maxUser: 5, req });
            // TODO
            throw new CustomError('0316', 'NotImplemented', req.languages);
        },
        apiKeyDelete: async (_parent: undefined, { input }: IWrap<any>, { req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ info, maxUser: 5, req });
            // TODO
            throw new CustomError('0317', 'NotImplemented', req.languages);
        },
        apiKeyValidate: async (_parent: undefined, { input }: IWrap<any>, { req, res }: Context, info: GraphQLResolveInfo): Promise<ApiKeyStatus> => {
            await rateLimit({ info, maxApi: 5000, req });
            // If session is expired
            if (!req.apiToken || !req.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError('0318', 'SessionExpired', req.languages);
            }
            // TODO
            throw new CustomError('0319', 'NotImplemented', req.languages);
        },
        emailLogIn: async (_parent: undefined, { input }: IWrap<EmailLogInInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<Session> => {
            await rateLimit({ info, maxUser: 100, req });
            // Validate arguments with schema
            emailLogInSchema.validateSync(input, { abortEarly: false });
            let user;
            // If email not supplied, check if session is valid. 
            // This is needed when this is called to verify an email address
            if (!input.email) {
                const userId = getUser(req)?.id;
                if (!userId)
                    throw new CustomError('0128', 'BadCredentials', req.languages);
                // Find user by id
                user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: { id: true }
                });
                if (!user)
                    throw new CustomError('0129', 'NoUser', req.languages);
                // Validate verification code
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(':'))
                        throw new CustomError('0130', 'CannotVerifyEmailCode', req.languages);
                    const [, verificationCode] = input.verificationCode.split(':');
                    // Find all emails for user
                    const emails = await prisma.email.findMany({
                        where: {
                            AND: [
                                { userId: user.id },
                                { verificationCode },
                            ]
                        }
                    });
                    if (emails.length === 0)
                        throw new CustomError('0131', 'EmailOrCodeInvalid', req.languages);
                    const verified = await validateVerificationCode(emails[0].emailAddress, user.id, verificationCode, prisma, req.languages);
                    if (!verified)
                        throw new CustomError('0132', 'CannotVerifyEmailCode', req.languages);
                }
                return await toSession(user, prisma, req);
            }
            // If email supplied, validate
            else {
                const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
                if (!email)
                    throw new CustomError('0133', 'EmailNotFound', req.languages);
                // Find user
                user = await prisma.user.findUnique({ where: { id: email.userId ?? '' } });
                if (!user)
                    throw new CustomError('0134', 'NoUser', req.languages);
                // Check for password in database, if doesn't exist, send a password reset link
                if (!Boolean(user.password)) {
                    await setupPasswordReset(user, prisma);
                    throw new CustomError('0135', 'MustResetPassword', req.languages);
                }
                // Validate verification code, if supplied
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(':'))
                        throw new CustomError('0136', 'CannotVerifyEmailCode', req.languages);
                    const [, verificationCode] = input.verificationCode.split(':');
                    const verified = await validateVerificationCode(email.emailAddress, user.id, verificationCode, prisma, req.languages);
                    if (!verified)
                        throw new CustomError('0137', 'CannotVerifyEmailCode', req.languages);
                }
                // Create new session
                const session = await logIn(input?.password as string, user, prisma, req);
                if (session) {
                    // Set session token
                    await generateSessionJwt(res, session);
                    return session;
                } else {
                    throw new CustomError('0138', 'BadCredentials', req.languages);
                }
            }
        },
        emailSignUp: async (_parent: undefined, { input }: IWrap<EmailSignUpInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            emailSignUpSchema.validateSync(input, { abortEarly: false });
            // Check for censored words
            if (hasProfanity(input.name))
                throw new CustomError('0140', 'BannedWord', req.languages);
            // Check if email exists
            const existingEmail = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (existingEmail) throw new CustomError('0141', 'EmailInUse', req.languages);
            // Create user object
            const user = await prisma.user.create({
                data: {
                    name: input.name,
                    password: hashPassword(input.password),
                    theme: input.theme,
                    status: AccountStatus.Unlocked,
                    emails: {
                        create: [
                            { emailAddress: input.email },
                        ]
                    }
                }
            });
            if (!user)
                throw new CustomError('0142', 'FailedToCreate', req.languages);
            // Create session from user object
            const session = await toSession(user, prisma, req);
            // Set up session token
            await generateSessionJwt(res, session);
            // Trigger new account
            await Trigger(prisma, req.languages).acountNew(user.id, input.email);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_parent: undefined, { input }: IWrap<EmailRequestPasswordChangeInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? '' } });
            if (!email)
                throw new CustomError('0143', 'EmailNotFound', req.languages);
            // Find user
            let user = await prisma.user.findUnique({ where: { id: email.userId ?? '' } });
            if (!user)
                throw new CustomError('0144', 'NoUser', req.languages);
            // Generate and send password reset code
            const success = await setupPasswordReset(user, prisma);
            return { success };
        },
        emailResetPassword: async (_parent: undefined, { input }: IWrap<EmailResetPasswordInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            passwordSchema.validateSync(input.newPassword, { abortEarly: false });
            // Find user
            let user = await prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                }
            });
            if (!user)
                throw new CustomError('0145', 'NoUser', req.languages);
            // If code is invalid
            if (!validateCode(input.code, user.resetPasswordCode ?? '', user.lastResetPasswordReqestAttempt as Date)) {
                // Generate and send new code
                await setupPasswordReset(user, prisma);
                // Return error
                throw new CustomError('0156', 'InvalidResetCode', req.languages);
            }
            // Remove request data from user, and set new password
            await prisma.user.update({
                where: { id: user.id as unknown as string },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: hashPassword(input.newPassword)
                }
            })
            // Return session
            return await toSession(user, prisma, req);
        },
        guestLogIn: async (_parent: undefined, _args: undefined, { req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ info, maxUser: 500, req });
            // Create session
            const session: RecursivePartial<Session> = {
                isLoggedIn: false
            }
            // Set up session token
            await generateSessionJwt(res, session);
            return session;
        },
        logOut: async (_parent: undefined, { input }: IWrap<LogOutInput>, { req, res }: Context, _info: GraphQLResolveInfo): Promise<Session> => {
            // If ID not specified OR there is only one user in the session, log out all sessions
            if (!input.id || (!Array.isArray(req.users) || req.users.length <= 1)) {
                res.clearCookie(COOKIE.Jwt);
                // Return guest session
                await generateSessionJwt(res, { isLoggedIn: false });
                return { isLoggedIn: false };
            }
            // Otherwise, remove the specified user from the session
            else {
                const session = { isLoggedIn: true, users: req.users.filter(u => u.id !== input.id) };
                await generateSessionJwt(res, session);
                return session;
            }
        },
        validateSession: async (_parent: undefined, { input }: IWrap<any>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            await rateLimit({ info, maxUser: 5000, req });
            const userId = getUser(req)?.id;
            // If session is expired
            if (!userId || !req.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError('0315', 'SessionExpired', req.languages);
            }
            // If timezone is updated, update session
            updateSessionTimeZone(req, res, input.timeZone);
            // If guest, return default session
            if (req.isLoggedIn !== true) {
                // Make sure session is cleared
                res.clearCookie(COOKIE.Jwt);
                return {
                    isLoggedIn: false,
                }
            }
            // Otherwise, check if session can be verified from userId
            const userData = await prisma.user.findUnique({
                where: { id: userId },
                select: { id: true }
            });
            if (userData) return await toSession(userData, prisma, req);
            // If user data failed to fetch, clear session and return error
            res.clearCookie(COOKIE.Jwt);
            throw new CustomError('0148', 'NotVerified', req.languages);
        },
        switchCurrentAccount: async (_parent: undefined, { input }: IWrap<SwitchCurrentAccountInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Session>> => {
            // Find index of user in session
            const index = req.users?.findIndex(u => u.id === input.id) ?? -1;
            // If user not found, throw error
            if (!req.users || index === -1) throw new CustomError('0272', 'NoUser', req.languages);
            // Filter out user from session, then place at front
            const users = req.users.filter(u => u.id !== input.id) ?? [];
            const session = { isLoggedIn: true, users: [req.users[index], ...users] };
            // Set up session token
            await generateSessionJwt(res, session);
            return session;
        },
        /**
         * Starts handshake for establishing trust between backend and user wallet
         * @returns Nonce that wallet must sign and send to walletComplete endpoint
         */
        walletInit: async (_parent: undefined, { input }: IWrap<WalletInitInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<string> => {
            await rateLimit({ info, maxUser: 100, req });
            // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
            const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
            if (!deserializedStakingAddress.startsWith('stake1'))
                throw new CustomError('0149', 'MustUseMainnet', req.languages);
            // Generate nonce for handshake
            const nonce = await generateNonce(input.nonceDescription as string | undefined);
            // Find existing wallet data in database
            let walletData = await prisma.wallet.findUnique({
                where: {
                    stakingAddress: input.stakingAddress,
                },
                select: {
                    id: true,
                    verified: true,
                    userId: true,
                }
            });
            // If wallet exists, update with new nonce
            if (walletData) {
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
                walletData = await prisma.wallet.create({
                    data: {
                        stakingAddress: input.stakingAddress,
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
        walletComplete: async (_parent: undefined, { input }: IWrap<WalletCompleteInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<WalletComplete> => {
            await rateLimit({ info, maxUser: 100, req });;
            // Find wallet with public address
            const walletData = await prisma.wallet.findUnique({
                where: { stakingAddress: input.stakingAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    user: {
                        select: { id: true }
                    },
                    verified: true,
                }
            });
            // If wallet doesn't exist, throw error
            if (!walletData)
                throw new CustomError('0150', 'WalletNotFound', req.languages);
            // If nonce expired, throw error
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) 
                throw new CustomError('0314', 'NonceExpired', req.languages)
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified)
                throw new CustomError('0151', 'CannotVerifyWallet', req.languages);
            let userId: string | undefined = walletData.user?.id;
            // If wallet is verified and assigned to another user, throw error
            // Otherwise, we can take ownership of wallet
            if (walletData.verified && userId && !req.users?.find(u => u.id === userId)) {
                throw new CustomError('0152', 'NotYourWallet', req.languages);
            }
            // If there are no users in the session, create a new user
            let firstLogIn: boolean = false;
            if (!Array.isArray(req.users) || req.users.length === 0) {
                firstLogIn = true;
                const userData = await prisma.user.create({
                    data: {
                        name: `user${randomString(8)}`,
                        wallets: {
                            connect: { id: walletData.id }
                        }
                    },
                    select: { id: true }
                });
                userId = userData.id;
            }
            // Otherwise, connect wallet to first user in session
            else {
                userId = req.users[0].id;
                await prisma.wallet.update({
                    where: { id: walletData.id },
                    data: {
                        user: {
                            connect: { id: userId }
                        }
                    }
                })
            }
            // Update wallet and remove nonce data
            const wallet = await prisma.wallet.update({
                where: { id: walletData.id },
                data: {
                    verified: true,
                    lastVerifiedTime: new Date().toISOString(),
                    nonce: null,
                    nonceCreationTime: null,
                },
                select: {
                    id: true,
                    name: true,
                    handles: {
                        select: {
                            id: true,
                            handle: true,
                        }
                    },
                    publicAddress: true,
                    stakingAddress: true,
                    verified: true,
                }
            })
            // Create session token
            const session = await toSession({ id: userId as string }, prisma, req)
            // Add session token to return payload
            await generateSessionJwt(res, session);
            return {
                firstLogIn,
                session,
                wallet,
            } as WalletComplete;
        },
    }
}