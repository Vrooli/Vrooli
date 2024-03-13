import { AccountStatus, COOKIE, emailLogInFormValidation, EmailLogInInput, EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, EmailResetPasswordInput, EmailSignUpInput, emailSignUpValidation, LINKS, LogOutInput, password as passwordValidation, ResourceUsedFor, Session, Success, SwitchCurrentAccountInput, ValidateSessionInput, WalletComplete, WalletCompleteInput, WalletInitInput } from "@local/shared";
import { randomString, validateCode } from "../../auth/codes";
import { EMAIL_VERIFICATION_TIMEOUT, hashPassword, logIn, setupPasswordReset, validateEmailVerificationCode } from "../../auth/email";
import { generateSessionJwt, getUser, updateSessionTimeZone } from "../../auth/request";
import { sessionUserTokenToUser, toSession, toSessionUser } from "../../auth/session";
import { generateNonce, serializedAddressToBech32, verifySignedMessage } from "../../auth/wallet";
import { Award } from "../../events/awards";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { rateLimit } from "../../middleware/rateLimit";
import { UI_URL_REMOTE } from "../../server";
import { GQLEndpoint, RecursivePartial } from "../../types";
import { hasProfanity } from "../../utils/censor";

export type EndpointsAuth = {
    Mutation: {
        emailLogIn: GQLEndpoint<EmailLogInInput, Session>;
        emailSignUp: GQLEndpoint<EmailSignUpInput, Session>;
        emailRequestPasswordChange: GQLEndpoint<EmailRequestPasswordChangeInput, Success>;
        emailResetPassword: GQLEndpoint<EmailResetPasswordInput, Session>;
        guestLogIn: GQLEndpoint<Record<string, never>, RecursivePartial<Session>>;
        logOut: GQLEndpoint<LogOutInput, Session>;
        validateSession: GQLEndpoint<ValidateSessionInput, RecursivePartial<Session>>;
        switchCurrentAccount: GQLEndpoint<SwitchCurrentAccountInput, RecursivePartial<Session>>;
        walletInit: GQLEndpoint<WalletInitInput, string>;
        walletComplete: GQLEndpoint<WalletCompleteInput, WalletComplete>;
    }
}

/** Expiry time for wallet authentication */
const NONCE_VALID_DURATION = 5 * 60 * 1000; // 5 minutes

/** Default user data */
const DEFAULT_USER_DATA = {
    isPrivateBookmarks: true,
    isPrivateVotes: true,
    focusModes: {
        create: [{
            name: "Work",
            description: "This is an auto-generated focus mode. You can edit or delete it.",
            reminderList: { create: {} },
            resourceList: {
                create: {
                    resources: {
                        create: [{
                            link: `${UI_URL_REMOTE}${LINKS.Calendar}`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Schedule",
                                    description: "View your schedule and add new events.",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.MyStuff}?type=Reminder`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Reminders",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.MyStuff}?type=Note`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Notes",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.History}?type=RunsActive`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Runs",
                                    description: "View your active routines and projects, and start new ones.",
                                }],
                            },
                        }],
                    },
                },
            },
        }, {
            name: "Study",
            description: "This is an auto-generated focus mode. You can edit or delete it.",
            reminderList: { create: {} },
            resourceList: {
                create: {
                    resources: {
                        create: [{
                            link: `${UI_URL_REMOTE}${LINKS.History}?type=Bookmarked`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Bookmarks",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.Search}?type=Routine`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Find Routines",
                                    description: "Search for public routines to view, run, schedule, or bookmark.",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.Search}?type=Project`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Find Projects",
                                    description: "Search for public projects to view, run, schedule, or bookmark.",
                                }],
                            },
                        }],
                    },
                },
            },
        }],
    },
};

/**
 * GraphQL endpoints for all authentication queries and mutations. These include:
 * 1. Wallet login
 * 2. Email sign up, log in, verification, and password reset
 * 3. Guest login
 */
export const AuthEndpoints: EndpointsAuth = {
    Mutation: {
        emailLogIn: async (_, { input }, { prisma, req, res }) => {
            await rateLimit({ maxUser: 100, req });
            // Validate arguments with schema
            emailLogInFormValidation.validateSync(input, { abortEarly: false });
            let user;
            // If email not supplied, check if session is valid. 
            // This is needed when this is called to verify an email address
            if (!input.email) {
                const userId = getUser(req.session)?.id;
                if (!userId)
                    throw new CustomError("0128", "BadCredentials", req.session.languages);
                // Find user by id
                user = await prisma.user.findUnique({
                    where: { id: userId },
                    select: { id: true },
                });
                if (!user)
                    throw new CustomError("0129", "NoUser", req.session.languages);
                // Validate verification code
                if (input.verificationCode) {
                    if (!input.verificationCode.includes(":"))
                        throw new CustomError("0130", "CannotVerifyEmailCode", req.session.languages);
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
                    const firstEmail = emails[0];
                    if (!firstEmail || !verificationCode)
                        throw new CustomError("0131", "EmailOrCodeInvalid", req.session.languages, { verificationCode });
                    const verified = await validateEmailVerificationCode(firstEmail.emailAddress, user.id, verificationCode, prisma, req.session.languages);
                    if (!verified)
                        throw new CustomError("0132", "CannotVerifyEmailCode", req.session.languages);
                }
                return await toSession(user, prisma, req);
            }
            // If email supplied, validate
            else {
                const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
                if (!email)
                    throw new CustomError("0133", "EmailNotFound", req.session.languages);
                // Find user
                user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
                if (!user)
                    throw new CustomError("0134", "NoUser", req.session.languages);
                // Check for password in database, if doesn't exist, send a password reset link
                if (!user.password) {
                    await setupPasswordReset(user, prisma);
                    throw new CustomError("0135", "MustResetPassword", req.session.languages);
                }
                // Validate verification code, if supplied
                if (input.verificationCode) {
                    const [, verificationCode] = input.verificationCode.includes(":") ? input.verificationCode.split(":") : [undefined, undefined];
                    if (!verificationCode)
                        throw new CustomError("0136", "CannotVerifyEmailCode", req.session.languages);
                    const verified = await validateEmailVerificationCode(email.emailAddress, user.id, verificationCode, prisma, req.session.languages);
                    if (!verified)
                        throw new CustomError("0137", "CannotVerifyEmailCode", req.session.languages);
                }
                // Create new session
                const session = await logIn(input?.password as string, user, prisma, req);
                if (session) {
                    // Set session token
                    await generateSessionJwt(res, session);
                    return session;
                } else {
                    throw new CustomError("0138", "BadCredentials", req.session.languages);
                }
            }
        },
        emailSignUp: async (_, { input }, { prisma, req, res }) => {
            await rateLimit({ maxUser: 10, req });
            // Validate input format
            emailSignUpValidation.validateSync(input, { abortEarly: false });
            // Check for censored words
            if (hasProfanity(input.name))
                throw new CustomError("0140", "BannedWord", req.session.languages);
            // Check if email exists
            const existingEmail = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (existingEmail) throw new CustomError("0141", "EmailInUse", req.session.languages);
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
                    ...DEFAULT_USER_DATA,
                },
            });
            if (!user)
                throw new CustomError("0142", "FailedToCreate", req.session.languages);
            // Give user award for signing up
            await Award(prisma, user.id, req.session.languages).update("AccountNew", 1);
            // Create session from user object
            const session = await toSession(user, prisma, req);
            // Set up session token
            await generateSessionJwt(res, session);
            // Trigger new account
            await Trigger(prisma, req.session.languages).acountNew(user.id, input.email);
            // Return user data
            return session;
        },
        emailRequestPasswordChange: async (_, { input }, { prisma, req }) => {
            await rateLimit({ maxUser: 10, req });
            // Validate input format
            emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
            // Validate email address
            const email = await prisma.email.findUnique({ where: { emailAddress: input.email ?? "" } });
            if (!email)
                throw new CustomError("0143", "EmailNotFound", req.session.languages);
            // Find user
            const user = await prisma.user.findUnique({ where: { id: email.userId ?? "" } });
            if (!user)
                throw new CustomError("0144", "NoUser", req.session.languages);
            // Generate and send password reset code
            const success = await setupPasswordReset(user, prisma);
            return { __typename: "Success", success };
        },
        emailResetPassword: async (_, { input }, { prisma, req, res }) => {
            await rateLimit({ maxUser: 10, req });
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
                throw new CustomError("0145", "NoUser", req.session.languages);
            // If code is invalid
            if (!validateCode(input.code, user.resetPasswordCode ?? "", user.lastResetPasswordReqestAttempt as Date, EMAIL_VERIFICATION_TIMEOUT)) {
                // Generate and send new code
                await setupPasswordReset(user, prisma);
                // Return error
                throw new CustomError("0156", "InvalidResetCode", req.session.languages);
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
            await generateSessionJwt(res, session);
            return session;
        },
        guestLogIn: async (_p, _d, { req, res }) => {
            await rateLimit({ maxUser: 500, req });
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
            if (!input.id || (!Array.isArray(req.session.users) || req.session.users.length <= 1)) {
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
                    users: req.session.users.filter(u => u.id !== input.id).map(sessionUserTokenToUser),
                };
                await generateSessionJwt(res, session);
                return session;
            }
        },
        validateSession: async (_, { input }, { prisma, req, res }) => {
            await rateLimit({ maxUser: 5000, req });
            const userId = getUser(req.session)?.id;
            // If session is expired
            if (!userId || !req.session.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError("0315", "SessionExpired", req.session.languages);
            }
            // If time zone is updated, update session
            updateSessionTimeZone(req, res, input.timeZone);
            // If guest, return default session
            if (req.session.isLoggedIn !== true) {
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
            if (userData) {
                const session = await toSession(userData, prisma, req);
                await generateSessionJwt(res, session);
                return session;
            }
            // If user data failed to fetch, clear session and return error
            res.clearCookie(COOKIE.Jwt);
            throw new CustomError("0148", "NotVerified", req.session.languages);
        },
        switchCurrentAccount: async (_, { input }, { prisma, req, res }) => {
            // Find index of user in session
            const index = req.session.users?.findIndex(u => u.id === input.id) ?? -1;
            // If user not found, throw error
            if (!req.session.users || index === -1) throw new CustomError("0272", "NoUser", req.session.languages);
            // Filter out user from session, then place at front
            const otherUsers = (req.session.users.filter(u => u.id !== input.id) ?? []).map(sessionUserTokenToUser);
            const currentUser = await toSessionUser(req.session.users[index]!, prisma, req);
            const session = { isLoggedIn: true, users: [currentUser, ...otherUsers] };
            // Set up session token
            await generateSessionJwt(res, session);
            return session;
        },
        /**
         * Starts handshake for establishing trust between backend and user wallet
         * @returns Nonce that wallet must sign and send to walletComplete endpoint
         */
        walletInit: async (_, { input }, { prisma, req }) => {
            await rateLimit({ maxUser: 100, req });
            // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
            const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
            if (!deserializedStakingAddress.startsWith("stake1"))
                throw new CustomError("0149", "MustUseMainnet", req.session.languages);
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
        walletComplete: async (_, { input }, { prisma, req, res }) => {
            await rateLimit({ maxUser: 100, req });
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
                throw new CustomError("0150", "WalletNotFound", req.session.languages);
            // If nonce expired, throw error
            if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION)
                throw new CustomError("0314", "NonceExpired", req.session.languages);
            // Verify that message was signed by wallet address
            const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
            if (!walletVerified)
                throw new CustomError("0151", "CannotVerifyWallet", req.session.languages);
            let userId: string | undefined = walletData.user?.id;
            let firstLogIn = false;
            // If you are not signed in
            if (!req.session.isLoggedIn) {
                // Wallet must be verified
                if (!walletData.verified) {
                    throw new CustomError("0152", "NotYourWallet", req.session.languages);
                }
                firstLogIn = true;
                // Create new user
                const userData = await prisma.user.create({
                    data: {
                        name: `user${randomString(8, "0123456789")}`,
                        wallets: {
                            connect: { id: walletData.id },
                        },
                        ...DEFAULT_USER_DATA,
                    },
                    select: { id: true },
                });
                userId = userData.id;
                // Give user award for signing up
                await Award(prisma, userId, req.session.languages).update("AccountNew", 1);
            }
            // If you are signed in
            else {
                // If wallet is not verified, link it to your account
                if (!walletData.verified) {
                    await prisma.user.update({
                        where: { id: req.session.users?.[0]?.id as string },
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
                    publicAddress: true,
                    stakingAddress: true,
                    verified: true,
                },
            });
            // Create session token
            const session = await toSession({ id: userId as string }, prisma, req);
            // Add session token to return payload
            await generateSessionJwt(res, session);
            return {
                firstLogIn,
                session,
                wallet,
            } as WalletComplete;
        },
    },
};
