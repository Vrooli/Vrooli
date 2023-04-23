import { COOKIE } from "@local/consts";
import { emailLogInFormValidation, emailRequestPasswordChangeSchema, emailSignUpFormValidation, password as passwordValidation } from "@local/validation";
import pkg from "@prisma/client";
import { gql } from "apollo-server-express";
import { getUser, hashPassword, logIn, sessionUserTokenToUser, setupPasswordReset, toSession, toSessionUser, validateCode, validateVerificationCode } from "../auth";
import { generateSessionJwt, updateSessionTimeZone } from "../auth/request.js";
import { generateNonce, randomString, serializedAddressToBech32, verifySignedMessage } from "../auth/wallet";
import { Award, Trigger } from "../events";
import { CustomError } from "../events/error";
import { rateLimit } from "../middleware";
import { hasProfanity } from "../utils/censor";
const { AccountStatus } = pkg;
const NONCE_VALID_DURATION = 5 * 60 * 1000;
export const typeDef = gql `
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
export const resolvers = {
    AccountStatus,
    Mutation: {
        emailLogIn: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            emailLogInFormValidation.validateSync(input, { abortEarly: false });
            let user;
            if (!input.email) {
                const userId = getUser(req)?.id;
                if (!userId)
                    throw new CustomError("0128", "BadCredentials", req.languages);
                user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: { id: true },
                });
                if (!user)
                    throw new CustomError("0129", "NoUser", req.languages);
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(":"))
                        throw new CustomError("0130", "CannotVerifyEmailCode", req.languages);
                    const [, verificationCode] = input.verificationCode.split(":");
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
            else {
                const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
                if (!email)
                    throw new CustomError("0133", "EmailNotFound", req.languages);
                user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
                if (!user)
                    throw new CustomError("0134", "NoUser", req.languages);
                if (!user.password) {
                    await setupPasswordReset(user, prisma);
                    throw new CustomError("0135", "MustResetPassword", req.languages);
                }
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(":"))
                        throw new CustomError("0136", "CannotVerifyEmailCode", req.languages);
                    const [, verificationCode] = input.verificationCode.split(":");
                    const verified = await validateVerificationCode(email.emailAddress, user.id, verificationCode, prisma, req.languages);
                    if (!verified)
                        throw new CustomError("0137", "CannotVerifyEmailCode", req.languages);
                }
                const session = await logIn(input?.password, user, prisma, req);
                if (session) {
                    await generateSessionJwt(res, session);
                    return session;
                }
                else {
                    throw new CustomError("0138", "BadCredentials", req.languages);
                }
            }
        },
        emailSignUp: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            emailSignUpFormValidation.validateSync(input, { abortEarly: false });
            if (hasProfanity(input.name))
                throw new CustomError("0140", "BannedWord", req.languages);
            const existingEmail = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (existingEmail)
                throw new CustomError("0141", "EmailInUse", req.languages);
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
            await Award(prisma, user.id, req.languages).update("AccountNew", 1);
            const session = await toSession(user, prisma, req);
            await generateSessionJwt(res, session);
            await Trigger(prisma, req.languages).acountNew(user.id, input.email);
            return session;
        },
        emailRequestPasswordChange: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (!email)
                throw new CustomError("0143", "EmailNotFound", req.languages);
            const user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
            if (!user)
                throw new CustomError("0144", "NoUser", req.languages);
            const success = await setupPasswordReset(user, prisma);
            return { __typename: "Success", success };
        },
        emailResetPassword: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            passwordValidation.validateSync(input.newPassword, { abortEarly: false });
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
            if (!validateCode(input.code, user.resetPasswordCode ?? "", user.lastResetPasswordReqestAttempt)) {
                await setupPasswordReset(user, prisma);
                throw new CustomError("0156", "InvalidResetCode", req.languages);
            }
            await prisma.user.update({
                where: { id: user.id },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null,
                    password: hashPassword(input.newPassword),
                },
            });
            const session = await toSession(user, prisma, req);
            await generateSessionJwt(res, session);
            return session;
        },
        guestLogIn: async (_p, _d, { req, res }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            const session = {
                __typename: "Session",
                isLoggedIn: false,
                users: [],
            };
            await generateSessionJwt(res, session);
            return session;
        },
        logOut: async (_, { input }, { req, res }) => {
            if (!input.id || (!Array.isArray(req.users) || req.users.length <= 1)) {
                res.clearCookie(COOKIE.Jwt);
                await generateSessionJwt(res, { isLoggedIn: false });
                return { __typename: "Session", isLoggedIn: false };
            }
            else {
                const session = {
                    __typename: "Session",
                    isLoggedIn: true,
                    users: req.users.filter(u => u.id !== input.id).map(sessionUserTokenToUser),
                };
                await generateSessionJwt(res, session);
                return session;
            }
        },
        validateSession: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 5000, req });
            const userId = getUser(req)?.id;
            if (!userId || !req.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError("0315", "SessionExpired", req.languages);
            }
            updateSessionTimeZone(req, res, input.timeZone);
            if (req.isLoggedIn !== true) {
                res.clearCookie(COOKIE.Jwt);
                return {
                    isLoggedIn: false,
                };
            }
            const userData = await prisma.user.findUnique({
                where: { id: userId },
                select: { id: true },
            });
            if (userData)
                return await toSession(userData, prisma, req);
            res.clearCookie(COOKIE.Jwt);
            throw new CustomError("0148", "NotVerified", req.languages);
        },
        switchCurrentAccount: async (_, { input }, { prisma, req, res }) => {
            const index = req.users?.findIndex(u => u.id === input.id) ?? -1;
            if (!req.users || index === -1)
                throw new CustomError("0272", "NoUser", req.languages);
            const otherUsers = (req.users.filter(u => u.id !== input.id) ?? []).map(sessionUserTokenToUser);
            const currentUser = await toSessionUser(req.users[index], prisma, req);
            const session = { isLoggedIn: true, users: [currentUser, ...otherUsers] };
            await generateSessionJwt(res, session);
            return session;
        },
        walletInit: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
            if (!deserializedStakingAddress.startsWith("stake1"))
                throw new CustomError("0149", "MustUseMainnet", req.languages);
            const nonce = await generateNonce(input.nonceDescription);
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
            if (walletData) {
                await prisma.wallet.update({
                    where: { id: walletData.id },
                    data: {
                        nonce,
                        nonceCreationTime: new Date().toISOString(),
                    },
                });
            }
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
        walletComplete: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 100, req });
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
            if (!walletData)
                throw new CustomError("0150", "WalletNotFound", req.languages);
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION)
                throw new CustomError("0314", "NonceExpired", req.languages);
            const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified)
                throw new CustomError("0151", "CannotVerifyWallet", req.languages);
            let userId = walletData.user?.id;
            let firstLogIn = false;
            if (!req.isLoggedIn) {
                if (!walletData.verified) {
                    throw new CustomError("0152", "NotYourWallet", req.languages);
                }
                firstLogIn = true;
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
                await Award(prisma, userId, req.languages).update("AccountNew", 1);
            }
            else {
                if (!walletData.verified) {
                    await prisma.user.update({
                        where: { id: req.users?.[0].id },
                        data: {
                            wallets: {
                                connect: { id: walletData.id },
                            },
                        },
                    });
                }
            }
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
            const session = await toSession({ id: userId }, prisma, req);
            await generateSessionJwt(res, session);
            return {
                firstLogIn,
                session,
                wallet,
            };
        },
    },
};
//# sourceMappingURL=auth.js.map