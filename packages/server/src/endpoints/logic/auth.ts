import { AUTH_PROVIDERS, AccountStatus, COOKIE, EmailLogInInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, EmailSignUpInput, LINKS, MINUTES_5_MS, ResourceUsedFor, Session, Success, SwitchCurrentAccountInput, ValidateSessionInput, WalletComplete, WalletCompleteInput, WalletInit, WalletInitInput, emailLogInFormValidation, emailRequestPasswordChangeSchema, emailResetPasswordSchema, emailSignUpValidation, switchCurrentAccountSchema, uuid, validateSessionSchema } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { Response } from "express";
import { AuthTokensService } from "../../auth/auth";
import { randomString, validateCode } from "../../auth/codes";
import { EMAIL_VERIFICATION_TIMEOUT, PasswordAuthService, UserDataForPasswordAuth } from "../../auth/email";
import { JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "../../auth/jwt";
import { RequestService } from "../../auth/request";
import { SessionService } from "../../auth/session";
import { generateNonce, serializedAddressToBech32, verifySignedMessage } from "../../auth/wallet";
import { prismaInstance } from "../../db/instance";
import { Award } from "../../events/awards";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { DEFAULT_USER_NAME_LENGTH } from "../../models/base/user";
import { UI_URL_REMOTE } from "../../server";
import { closeSessionSockets, closeUserSockets } from "../../sockets/events";
import { ApiEndpoint, RecursivePartial, SessionData } from "../../types";
import { hasProfanity } from "../../utils/censor";

export type EndpointsAuth = {
    emailLogIn: ApiEndpoint<EmailLogInInput, Session>;
    emailSignUp: ApiEndpoint<EmailSignUpInput, Session>;
    emailRequestPasswordChange: ApiEndpoint<EmailRequestPasswordChangeInput, Success>;
    emailResetPassword: ApiEndpoint<EmailResetPasswordInput, Session>;
    guestLogIn: ApiEndpoint<Record<string, never>, RecursivePartial<Session>>;
    logOut: ApiEndpoint<Record<string, never>, Session>;
    logOutAll: ApiEndpoint<Record<string, never>, Session>;
    validateSession: ApiEndpoint<ValidateSessionInput, RecursivePartial<Session>>;
    switchCurrentAccount: ApiEndpoint<SwitchCurrentAccountInput, RecursivePartial<Session>>;
    walletInit: ApiEndpoint<WalletInitInput, WalletInit>;
    walletComplete: ApiEndpoint<WalletCompleteInput, WalletComplete>;
}

/** Expiry time for wallet authentication */
const NONCE_VALID_DURATION = MINUTES_5_MS;

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
                            link: `${UI_URL_REMOTE}${LINKS.MyStuff}?type="Reminder"`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Reminders",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.MyStuff}?type="Note"`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Notes",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.History}?type="RunsActive"`,
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
                            link: `${UI_URL_REMOTE}${LINKS.History}?type="Bookmarked"`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Bookmarks",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.Search}?type="Routine"`,
                            usedFor: ResourceUsedFor.Context,
                            translations: {
                                create: [{
                                    language: "en",
                                    name: "Find Routines",
                                    description: "Search for public routines to view, run, schedule, or bookmark.",
                                }],
                            },
                        }, {
                            link: `${UI_URL_REMOTE}${LINKS.Search}?type="Project"`,
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

const GUEST_SESSION: Session & SessionData = {
    __typename: "Session" as const,
    accessExpiresAt: 0,
    isLoggedIn: false,
    users: [],
};

async function establishGuestSession(res: Response) {
    res.clearCookie(COOKIE.Jwt);
    const session = GUEST_SESSION;
    await AuthTokensService.generateSessionToken(res, session);
    return session;
}

/**
 * GraphQL endpoints for all authentication queries and mutations. These include:
 * 1. Wallet login
 * 2. Email sign up, log in, verification, and password reset
 * 3. Guest login
 */
export const auth: EndpointsAuth = {
    emailLogIn: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        // Validate arguments with schema
        emailLogInFormValidation.validateSync(input, { abortEarly: false });
        // If email not supplied, check if session is valid. 
        // This is needed when this is called to verify an email address
        if (!input.email) {
            const user = RequestService.assertRequestFrom(req, { isOfficialUser: true });
            // Validate verification code
            if (input.verificationCode) {
                const email = await prismaInstance.email.findFirst({
                    where: {
                        AND: [
                            { userId: user.id },
                            { verificationCode: input.verificationCode },
                        ],
                    },
                });
                if (!email) {
                    throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
                }
                const verified = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id, input.verificationCode, req.session.languages);
                if (!verified) {
                    throw new CustomError("0132", "CannotVerifyEmailCode");
                }
            }
            return SessionService.createBaseSession(req);
        }
        // If email supplied, validate
        else {
            // Find user
            const user = await prismaInstance.user.findFirst({
                where: {
                    emails: {
                        some: {
                            emailAddress: input.email,
                        },
                    },
                },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            if (!user) {
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            }
            const email = user.emails.find(e => e.emailAddress === input.email);
            if (!email) {
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            }
            // Check for password in database. If it doesn't exist, send a password reset link
            const passwordHash = PasswordAuthService.getAuthPassword(user);
            if (!passwordHash) {
                await PasswordAuthService.setupPasswordReset(user);
                throw new CustomError("0135", "MustResetPassword");
            }
            // Validate verification code, if supplied
            if (input.verificationCode) {
                const isCodeValid = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id, input.verificationCode, req.session.languages);
                if (!isCodeValid) {
                    throw new CustomError("0137", "CannotVerifyEmailCode");
                }
            }
            // Create new session
            const session = await PasswordAuthService.logIn(input?.password as string, user, req);
            if (session) {
                // Set session token
                await AuthTokensService.generateSessionToken(res, session);
                return session;
            } else {
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            }
        }
    },
    emailSignUp: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        // Validate input format
        emailSignUpValidation.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name))
            throw new CustomError("0140", "BannedWord");
        // Check if email exists
        const existingEmail = await prismaInstance.email.findUnique({ where: { emailAddress: input.email ?? "" } });
        if (existingEmail) {
            throw new CustomError("0141", "EmailInUse");
        }
        // Get device info and IP address
        const deviceInfo = RequestService.getDeviceInfo(req);
        const ipAddress = req.ip;
        // Update the database
        const userId = uuid();
        const passwordAuthId = uuid();
        const transactions: PrismaPromise<object>[] = [
            // Create user
            prismaInstance.user.create({
                data: {
                    id: userId,
                    name: input.name,
                    theme: input.theme,
                    status: AccountStatus.Unlocked,
                    emails: {
                        create: [
                            { emailAddress: input.email },
                        ],
                    },
                    auths: {
                        create: {
                            id: passwordAuthId,
                            provider: AUTH_PROVIDERS.Password,
                            hashed_password: PasswordAuthService.hashPassword(input.password),
                        },
                    },
                    ...DEFAULT_USER_DATA,
                },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            }),
            // Create session
            prismaInstance.session.create({
                data: {
                    device_info: deviceInfo,
                    expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                    last_refresh_at: new Date(),
                    ip_address: ipAddress,
                    revoked: false,
                    user: {
                        connect: { id: userId },
                    },
                    auth: {
                        connect: { id: passwordAuthId },
                    },
                },
                select: PasswordAuthService.selectUserForPasswordAuth().sessions.select,
            }),
        ];
        const transactionResults = await prismaInstance.$transaction(transactions);
        // transactions[0]: user.create
        // transactions[1]: session.create
        const userData = transactionResults[0] as Omit<UserDataForPasswordAuth, "session">;
        const sessionData = transactionResults[1] as UserDataForPasswordAuth["sessions"][0];
        userData.sessions = [sessionData];
        if (!userData) {
            throw new CustomError("0142", "FailedToCreate");
        }
        // Give user award for signing up
        await Award(userData.id, req.session.languages).update("AccountNew", 1);
        // Create session from user object
        const session = await SessionService.createSession(userData, sessionData, req);
        // Set up session token
        await AuthTokensService.generateSessionToken(res, session);
        // Trigger new account
        await Trigger(req.session.languages).acountNew(userData.id, input.email);
        // Return user data
        return session;
    },
    emailRequestPasswordChange: async (_, { input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        // Validate input format
        emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
        // Find user
        const user = await prismaInstance.user.findFirst({
            where: {
                emails: {
                    some: {
                        emailAddress: input.email,
                    },
                },
            },
            select: PasswordAuthService.selectUserForPasswordAuth(),
        });
        if (!user) {
            throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
        }
        // Generate and send password reset code
        const success = await PasswordAuthService.setupPasswordReset(user);
        return { __typename: "Success", success };
    },
    emailResetPassword: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        // Validate input format
        emailResetPasswordSchema.validateSync(input, { abortEarly: false });
        // Find user
        const user = await prismaInstance.user.findUnique({
            where: { id: input.id },
            select: PasswordAuthService.selectUserForPasswordAuth(),
        });
        if (!user) {
            throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
        }
        // Check code
        const passwordAuth = user.auths.find(a => a.provider === AUTH_PROVIDERS.Password);
        const isResetPasswordCodeValid =
            passwordAuth
            && passwordAuth.resetPasswordCode
            && passwordAuth.lastResetPasswordRequestAttempt
            && validateCode(
                input.code,
                passwordAuth.resetPasswordCode,
                passwordAuth.lastResetPasswordRequestAttempt,
                EMAIL_VERIFICATION_TIMEOUT,
            );
        if (!isResetPasswordCodeValid) {
            // We used to send a new reset password code here, but have changed this in favor
            // of making the user request a new reset password code. This is to prevent spamming
            // users with emails.
            throw new CustomError("0156", "InvalidResetCode");
        }
        // Get device info and IP address
        const deviceInfo = RequestService.getDeviceInfo(req);
        const ipAddress = req.ip;
        // Update database
        const transactions: PrismaPromise<object>[] = [
            // Update reset password request data and set new password
            prismaInstance.user_auth.update({
                where: {
                    id: passwordAuth.id,
                },
                data: {
                    resetPasswordCode: null,
                    lastResetPasswordRequestAttempt: null,
                    hashed_password: PasswordAuthService.hashPassword(input.newPassword),
                },
            }),
            // Reset log in attempts
            prismaInstance.user.update({
                where: { id: user.id },
                data: {
                    logInAttempts: 0,
                    lastLoginAttempt: new Date(),
                },
            }),
            // Revoke all existing password sessions, even ones using other authentication methods
            // You could also delete them, but this may be required for compliance reasons. 
            // We'll rely on a cron job to delete old sessions.
            prismaInstance.session.updateMany({
                where: {
                    user: {
                        id: user.id,
                    },
                    revoked: false,
                },
                data: {
                    revoked: true,
                },
            }),
            // Create new session
            prismaInstance.session.create({
                data: {
                    device_info: deviceInfo,
                    expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                    last_refresh_at: new Date(),
                    ip_address: ipAddress,
                    revoked: false,
                    user: {
                        connect: { id: user.id },
                    },
                    auth: {
                        connect: { id: passwordAuth.id },
                    },
                },
                select: PasswordAuthService.selectUserForPasswordAuth().sessions.select,
            }),
        ];
        const transactionResults = await prismaInstance.$transaction(transactions);
        // transactions[0]: user_auth.update
        // transactions[1]: user.update
        // transactions[2]: session.updateMany
        // transactions[3]: session.create
        const sessionData = transactionResults[3] as UserDataForPasswordAuth["sessions"][0];
        // Create session from user object
        const session = await SessionService.createSession(user, sessionData, req);
        // Set up session token
        await AuthTokensService.generateSessionToken(res, session);
        return session;
    },
    guestLogIn: async (_p, _d, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        return establishGuestSession(res);
    },
    logOut: async (_p, _d, { req, res }) => {
        const userData = SessionService.getUser(req.session);
        const userId = userData?.id;
        // If user is already logged out, return a guest session
        if (!userId) {
            return establishGuestSession(res);
        }

        // Clear socket connections for session
        const sessionId = userData?.session?.id;
        if (sessionId) {
            closeSessionSockets(sessionId);
        }

        // A session can store multiple users. Check if we're logging out all users or just one
        const remainingUsers = req.session.users?.filter(u => u.id !== userId) ?? [];
        const isStayingLoggedIn = remainingUsers.length > 0;

        // If we're logging out all users, clear the session and return a guest session
        if (!isStayingLoggedIn) {
            return establishGuestSession(res);
        }

        // Revoke the sessions for every user that is being logged out
        const usersNotRemaining = req.session.users?.filter(u => u.id === userId) ?? [];
        const sessionIdsToRevoke = usersNotRemaining.map(u => u.session?.id).filter(Boolean);
        await prismaInstance.session.updateMany({
            where: {
                id: { in: sessionIdsToRevoke },
            },
            data: {
                revoked: true,
            },
        });

        // Create new session with the remaining users
        const session = {
            ...SessionService.createBaseSession(req),
            ...JsonWebToken.createAccessExpiresAt(),
            users: remainingUsers,
        };

        // Check if the updated session is still valid
        const isSessionValid = await AuthTokensService.canRefreshToken(session);
        if (!isSessionValid) {
            throw new CustomError("0274", "ErrorUnknown");
        }

        await AuthTokensService.generateSessionToken(res, session);
        return session;
    },
    logOutAll: async (_p, _d, { req, res }) => {
        // Get user data
        const userData = SessionService.getUser(req.session);
        const userId = userData?.id;
        // If user is already logged out, return a guest session
        if (!userId) {
            return establishGuestSession(res);
        }
        // Use raw query to revoke sessions and return their IDs
        const sessions = await prismaInstance.$queryRaw`
                UPDATE "session"
                SET revoked = true
                WHERE "user_id" = ${userId}::uuid
                RETURNING id;
            `;
        // Clear socket connections for sessions
        const sessionId = userData?.session?.id;
        if (sessionId) {
            closeSessionSockets(sessionId);
        }
        // Clear socket connections for user
        if (userId) {
            closeUserSockets(userId);
        }
        // Return a guest session
        return establishGuestSession(res);
    },
    // NOTE: The `authentiateRequest` middleware should have already validated the session token 
    // and handled the cookie. This makes the function below very simple.
    validateSession: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 5000, req });

        if (!req.session.isLoggedIn) {
            return GUEST_SESSION;
        }

        // Validate input format
        validateSessionSchema.validateSync(input, { abortEarly: false });

        // If time zone is provided, sign the token with the time zone
        if (input.timeZone && input.timeZone !== req.session.timeZone) {
            const { cookies } = req;
            const additionalData = { timeZone: input.timeZone };
            await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt], { additionalData });
            req.session.timeZone = input.timeZone ?? null;
            AuthTokensService.generateSessionToken(res, req.session);
        }

        return req.session;
    },
    switchCurrentAccount: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });

        // Validate input format
        switchCurrentAccountSchema.validateSync(input, { abortEarly: false });

        // Find user we're switching to in the session
        const targetUser = req.session.users?.find(u => u.id === input.id);
        if (!targetUser) {
            throw new CustomError("0272", "NoUser");
        }
        const otherUsers = req.session.users?.filter(u => u.id !== input.id) ?? [];

        // Create new session with the user we're switching to as the first user
        const session = {
            ...SessionService.createBaseSession(req),
            ...JsonWebToken.createAccessExpiresAt(),
            users: [targetUser, ...otherUsers],
        };

        // Check if the updated session is still valid
        const isSessionValid = await AuthTokensService.canRefreshToken(session);
        if (!isSessionValid) {
            throw new CustomError("0273", "ErrorUnknown");
        }

        // Set up session token
        await AuthTokensService.generateSessionToken(res, session);
        return session;
    },
    /**
     * Starts handshake for establishing trust between backend and user wallet
     * @returns Nonce that wallet must sign and send to walletComplete endpoint
     */
    walletInit: async (_, { input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
        const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
        if (!deserializedStakingAddress.startsWith("stake1"))
            throw new CustomError("0149", "MustUseMainnet");
        // Generate nonce for handshake
        const nonce = await generateNonce(input.nonceDescription as string | undefined);
        // Find existing wallet data in database
        let walletData = await prismaInstance.wallet.findUnique({
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
            await prismaInstance.wallet.update({
                where: { id: walletData.id },
                data: {
                    nonce,
                    nonceCreationTime: new Date().toISOString(),
                },
            });
        }
        // If wallet data doesn't exist, create
        if (!walletData) {
            walletData = await prismaInstance.wallet.create({
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
        return {
            __typename: "WalletInit",
            nonce,
        } as const;
    },
    // Verify that signed message from user wallet has been signed by the correct public address
    walletComplete: async (_, { input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        // Find wallet with public address
        const walletData = await prismaInstance.wallet.findUnique({
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
            throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
        // If nonce expired, throw error
        if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION)
            throw new CustomError("0314", "NonceExpired");
        // Verify that message was signed by wallet address
        const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
        if (!walletVerified)
            throw new CustomError("0151", "CannotVerifyWallet");
        let userId: string | undefined = walletData.user?.id;
        let firstLogIn = false;
        // If you are not signed in
        if (!req.session.isLoggedIn) {
            // Wallet must be verified
            if (!walletData.verified) {
                throw new CustomError("0152", "NotYourWallet");
            }
            firstLogIn = true;
            // Create new user
            const userData = await prismaInstance.user.create({
                data: {
                    name: `user${randomString(DEFAULT_USER_NAME_LENGTH, "0123456789")}`,
                    wallets: {
                        connect: { id: walletData.id },
                    },
                    ...DEFAULT_USER_DATA,
                },
                select: { id: true },
            });
            userId = userData.id;
            // Give user award for signing up
            await Award(userId, req.session.languages).update("AccountNew", 1);
        }
        // If you are signed in
        else {
            // If wallet is not verified, link it to your account
            if (!walletData.verified) {
                await prismaInstance.user.update({
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
        const wallet = await prismaInstance.wallet.update({
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
        // const session = await SessionService.createSession(userData, sessionData, req);
        const session = {} as any; //TODO 11/21
        // Add session token to return payload
        await AuthTokensService.generateSessionToken(res, session);
        return {
            __typename: "WalletComplete",
            firstLogIn,
            session,
            wallet: {
                ...wallet,
                __typename: "Wallet",
            }
        } as const;
    },
};
