// GraphQL endpoints for all authentication queries and mutations. These include:
// 1. Wallet login
// 2. Email sign up, log in, verification, and password reset
// 3. Guest login
import { COOKIE, emailLogInFormValidation, EmailLogInInput, EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, EmailResetPasswordInput, emailSignUpFormValidation, EmailSignUpInput, LogOutInput, password as passwordValidation, Session, Success, SwitchCurrentAccountInput, ValidateSessionInput, WalletComplete, WalletCompleteInput, WalletInitInput } from "@local/shared";
import pkg from "@prisma/client";
import { gql } from "apollo-server-express";
import { getUser, hashPassword, logIn, sessionUserTokenToUser, setupPasswordReset, toSession, toSessionUser, validateCode, validateVerificationCode } from "../auth";
import { generateSessionJwt, updateSessionTimeZone } from "../auth/request.js";
import { generateNonce, randomString, serializedAddressToBech32, verifySignedMessage } from "../auth/wallet";
import { Award, Trigger } from "../events";
import { CustomError } from "../events/error";
import { rateLimit } from "../middleware";
import { GQLEndpoint, RecursivePartial } from "../types";
import { hasProfanity } from "../utils/censor";

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
        timeZone: String!
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
        activeFocusMode: ActiveFocusMode
        apisCount: Int!
        bookmarkLists: [BookmarkList!]! # Will not include the bookmarks themselves, just info about the lists
        focusModes: [FocusMode!]!
        handle: String
        hasPremium: Boolean!
        id: String!
        languages: [String!]!
        membershipsCount: Int!
        name: String
        notesCount: Int!
        projectsCount: Int!
        questionsAskedCount: Int!
        routinesCount: Int!
        smartContractsCount: Int!
        standardsCount: Int!
        theme: String
    }

    type Session {
        isLoggedIn: Boolean!
        timeZone: String
        users: [SessionUser!]
    }

    extend type Mutation {
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
`;

export const resolvers: {
    AccountStatus: typeof AccountStatus;
    Mutation: {
        emailLogIn: GQLEndpoint<EmailLogInInput, Session>;
        emailSignUp: GQLEndpoint<EmailSignUpInput, Session>;
        emailRequestPasswordChange: GQLEndpoint<EmailRequestPasswordChangeInput, Success>;
        emailResetPassword: GQLEndpoint<EmailResetPasswordInput, Session>;
        guestLogIn: GQLEndpoint<{}, RecursivePartial<Session>>;
        logOut: GQLEndpoint<LogOutInput, Session>;
        validateSession: GQLEndpoint<ValidateSessionInput, RecursivePartial<Session>>;
        switchCurrentAccount: GQLEndpoint<SwitchCurrentAccountInput, RecursivePartial<Session>>;
        walletInit: GQLEndpoint<WalletInitInput, string>;
        walletComplete: GQLEndpoint<WalletCompleteInput, WalletComplete>;
    }
} = {
    AccountStatus,
    Mutation: {
        emailLogIn: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            // Validate arguments with schema
            emailLogInFormValidation.validateSync(input, { abortEarly: false });
            let user;
            // If email not supplied, check if session is valid. 
            // This is needed when this is called to verify an email address
            if (!input.email) {
                const userId = getUser(req)?.id;
                if (!userId)
                    throw new CustomError("0128", "BadCredentials", req.languages);
                // Find user by id
                user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: { id: true },
                });
                if (!user)
                    throw new CustomError("0129", "NoUser", req.languages);
                // Validate verification code
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(":"))
                        throw new CustomError("0130", "CannotVerifyEmailCode", req.languages);
                    const [, verificationCode] = input.verificationCode.split(":");
                    // Find all emails for user
                    const emails = await prisma.email.findMany({
                        where: {
                            AND: [
                                { userId: user.id },
                                { verificationCode },
                            ],
                        },
                    });
                    if (emails.length === 0)
                        throw new CustomError("0131", "EmailOrCodeInvalid", req.languages);
                    const verified = await validateVerificationCode(emails[0].emailAddress, user.id, verificationCode, prisma, req.languages);
                    if (!verified)
                        throw new CustomError("0132", "CannotVerifyEmailCode", req.languages);
                }
                return await toSession(user, prisma, req);
            }
            // If email supplied, validate
            else {
                const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
                if (!email)
                    throw new CustomError("0133", "EmailNotFound", req.languages);
                // Find user
                user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
                if (!user)
                    throw new CustomError("0134", "NoUser", req.languages);
                // Check for password in database, if doesn't exist, send a password reset link
                if (!user.password) {
                    await setupPasswordReset(user, prisma);
                    throw new CustomError("0135", "MustResetPassword", req.languages);
                }
                // Validate verification code, if supplied
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(":"))
                        throw new CustomError("0136", "CannotVerifyEmailCode", req.languages);
                    const [, verificationCode] = input.verificationCode.split(":");
                    const verified = await validateVerificationCode(email.emailAddress, user.id, verificationCode, prisma, req.languages);
                    if (!verified)
                        throw new CustomError("0137", "CannotVerifyEmailCode", req.languages);
                }
                // Create new session
                const session = await logIn(input?.password as string, user, prisma, req);
                if (session) {
                    // Set session token
                    await generateSessionJwt(res, session as any);
                    return session;
                } else {
                    throw new CustomError("0138", "BadCredentials", req.languages);
                }
            }
        },
        emailSignUp: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            emailSignUpFormValidation.validateSync(input, { abortEarly: false });
            // Check for censored words
            if (hasProfanity(input.name))
                throw new CustomError("0140", "BannedWord", req.languages);
            // Check if email exists
            const existingEmail = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (existingEmail) throw new CustomError("0141", "EmailInUse", req.languages);
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
                        ],
                    },
                    focusModes: {
                        create: [{
                            name: "Work",
                            description: "This is an auto-generated focus mode. You can edit or delete it.",
                            reminderList: { create: {} },
                            resourceList: { create: {} },
                        }, {
                            name: "Study",
                            description: "This is an auto-generated focus mode. You can edit or delete it.",
                            reminderList: { create: {} },
                            resourceList: { create: {} },
                        }],
                    },
                },
            });
            if (!user)
                throw new CustomError("0142", "FailedToCreate", req.languages);
            // Give user award for signing up
            await Award(prisma, user.id, req.languages).update("AccountNew", 1);
            // Create session from user object
            const session = await toSession(user, prisma, req);
            // Set up session token
            await generateSessionJwt(res, session as any);
            // Trigger new account
            await Trigger(prisma, req.languages).acountNew(user.id, input.email);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (!email)
                throw new CustomError("0143", "EmailNotFound", req.languages);
            // Find user
            const user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
            if (!user)
                throw new CustomError("0144", "NoUser", req.languages);
            // Generate and send password reset code
            const success = await setupPasswordReset(user, prisma);
            return { __typename: "Success", success };
        },
        emailResetPassword: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            // Validate input format
            passwordValidation.validateSync(input.newPassword, { abortEarly: false });
            // Find user
            const user = await prisma.user.findUnique({
                where: { id: input.id },
                select: {
                    id: true,
                    resetPasswordCode: true,
                    lastResetPasswordReqestAttempt: true,
                },
            });
            if (!user)
                throw new CustomError("0145", "NoUser", req.languages);
            // If code is invalid
            if (!validateCode(input.code, user.resetPasswordCode ?? "", user.lastResetPasswordReqestAttempt as Date)) {
                // Generate and send new code
                await setupPasswordReset(user, prisma);
                // Return error
                throw new CustomError("0156", "InvalidResetCode", req.languages);
            }
            // Remove request data from user, and set new password
            await prisma.user.update({
                where: { id: user.id as unknown as string },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: hashPassword(input.newPassword),
                },
            });
            // Create session from user object
            const session = await toSession(user, prisma, req);
            // Set up session token
            await generateSessionJwt(res, session as any);
            return session;
        },
        guestLogIn: async (_p, _d, { req, res }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            // Create session
            const session = {
                __typename: "Session" as const,
                isLoggedIn: false,
                users: [],
            };
            // Set up session token
            await generateSessionJwt(res, session);
            return session;
        },
        logOut: async (_, { input }, { req, res }) => {
            // If ID not specified OR there is only one user in the session, log out all sessions
            if (!input.id || (!Array.isArray(req.users) || req.users.length <= 1)) {
                res.clearCookie(COOKIE.Jwt);
                // Return guest session
                await generateSessionJwt(res, { isLoggedIn: false });
                return { __typename: "Session" as const, isLoggedIn: false };
            }
            // Otherwise, remove the specified user from the session
            else {
                const session = {
                    __typename: "Session" as const,
                    isLoggedIn: true,
                    users: req.users.filter(u => u.id !== input.id).map(sessionUserTokenToUser),
                };
                await generateSessionJwt(res, session);
                return session as any;
            }
        },
        validateSession: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const userId = getUser(req)?.id;
            // If session is expired
            if (!userId || !req.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError("0315", "SessionExpired", req.languages);
            }
            // If time zone is updated, update session
            updateSessionTimeZone(req, res, input.timeZone);
            // If guest, return default session
            if (req.isLoggedIn !== true) {
                // Make sure session is cleared
                res.clearCookie(COOKIE.Jwt);
                return {
                    isLoggedIn: false,
                };
            }
            // Otherwise, check if session can be verified from userId
            const userData = await prisma.user.findUnique({
                where: { id: userId },
                select: { id: true },
            });
            if (userData) return await toSession(userData, prisma, req);
            // If user data failed to fetch, clear session and return error
            res.clearCookie(COOKIE.Jwt);
            throw new CustomError("0148", "NotVerified", req.languages);
        },
        switchCurrentAccount: async (_, { input }, { prisma, req, res }) => {
            // Find index of user in session
            const index = req.users?.findIndex(u => u.id === input.id) ?? -1;
            // If user not found, throw error
            if (!req.users || index === -1) throw new CustomError("0272", "NoUser", req.languages);
            // Filter out user from session, then place at front
            const otherUsers = (req.users.filter(u => u.id !== input.id) ?? []).map(sessionUserTokenToUser);
            const currentUser = await toSessionUser(req.users[index], prisma, req);
            const session = { isLoggedIn: true, users: [currentUser, ...otherUsers] };
            // Set up session token
            await generateSessionJwt(res, session);
            return session as any;
        },
        /**
         * Starts handshake for establishing trust between backend and user wallet
         * @returns Nonce that wallet must sign and send to walletComplete endpoint
         */
        walletInit: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
            const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
            if (!deserializedStakingAddress.startsWith("stake1"))
                throw new CustomError("0149", "MustUseMainnet", req.languages);
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
                },
            });
            // If wallet exists, update with new nonce
            if (walletData) {
                await prisma.wallet.update({
                    where: { id: walletData.id },
                    data: {
                        nonce,
                        nonceCreationTime: new Date().toISOString(),
                    },
                });
            }
            // If wallet data doesn't exist, create
            if (!walletData) {
                walletData = await prisma.wallet.create({
                    data: {
                        stakingAddress: input.stakingAddress,
                        nonce,
                        nonceCreationTime: new Date().toISOString(),
                    },
                    select: {
                        id: true,
                        verified: true,
                        userId: true,
                    },
                });
            }
            return nonce;
        },
        // Verify that signed message from user wallet has been signed by the correct public address
        walletComplete: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            // Find wallet with public address
            const walletData = await prisma.wallet.findUnique({
                where: { stakingAddress: input.stakingAddress },
                select: {
                    id: true,
                    nonce: true,
                    nonceCreationTime: true,
                    user: {
                        select: { id: true },
                    },
                    verified: true,
                },
            });
            // If wallet doesn't exist, throw error
            if (!walletData)
                throw new CustomError("0150", "WalletNotFound", req.languages);
            // If nonce expired, throw error
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION)
                throw new CustomError("0314", "NonceExpired", req.languages);
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified)
                throw new CustomError("0151", "CannotVerifyWallet", req.languages);
            let userId: string | undefined = walletData.user?.id;
            let firstLogIn = false;
            // If you are not signed in
            if (!req.isLoggedIn) {
                // Wallet must be verified
                if (!walletData.verified) {
                    throw new CustomError("0152", "NotYourWallet", req.languages);
                }
                firstLogIn = true;
                // Create new user
                const userData = await prisma.user.create({
                    data: {
                        name: `user${randomString(8)}`,
                        wallets: {
                            connect: { id: walletData.id },
                        },
                        focusModes: {
                            create: [{
                                name: "Work",
                                description: "This is an auto-generated focus mode. You can edit or delete it.",
                                reminderList: { create: {} },
                                resourceList: { create: {} },
                            }, {
                                name: "Study",
                                description: "This is an auto-generated focus mode. You can edit or delete it.",
                                reminderList: { create: {} },
                                resourceList: { create: {} },
                            }],
                        },
                    },
                    select: { id: true },
                });
                userId = userData.id;
                // Give user award for signing up
                await Award(prisma, userId, req.languages).update("AccountNew", 1);
            }
            // If you are signed in
            else {
                // If wallet is not verified, link it to your account
                if (!walletData.verified) {
                    await prisma.user.update({
                        where: { id: req.users?.[0].id as string },
                        data: {
                            wallets: {
                                connect: { id: walletData.id },
                            },
                        },
                    });
                }
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
                        },
                    },
                    publicAddress: true,
                    stakingAddress: true,
                    verified: true,
                },
            });
            // Create session token
            const session = await toSession({ id: userId as string }, prisma, req);
            // Add session token to return payload
            await generateSessionJwt(res, session as any);
            return {
                firstLogIn,
                session,
                wallet,
            } as WalletComplete;
        },
    },
};
