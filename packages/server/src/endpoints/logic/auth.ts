import { AUTH_PROVIDERS, AccountStatus, COOKIE, EmailLogInInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, EmailSignUpInput, MINUTES_5_MS, Session, Success, SwitchCurrentAccountInput, ValidateSessionInput, WalletComplete, WalletCompleteInput, WalletInit, WalletInitInput, emailLogInFormValidation, emailRequestPasswordChangeSchema, emailResetPasswordSchema, emailSignUpValidation, switchCurrentAccountSchema, validateSessionSchema } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { Response } from "express";
import { AuthTokensService } from "../../auth/auth.js";
import { randomString, validateCode } from "../../auth/codes.js";
import { EMAIL_VERIFICATION_TIMEOUT, PasswordAuthService, UserDataForPasswordAuth } from "../../auth/email.js";
import { JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "../../auth/jwt.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { generateNonce, serializedAddressToBech32, verifySignedMessage } from "../../auth/wallet.js";
import { DbProvider } from "../../db/provider.js";
import { Award } from "../../events/awards.js";
import { CustomError } from "../../events/error.js";
import { Trigger } from "../../events/trigger.js";
import { DEFAULT_USER_NAME_LENGTH } from "../../models/base/user.js";
import { closeSessionSockets, closeUserSockets } from "../../sockets/events.js";
import { ApiEndpoint, SessionData } from "../../types.js";
import { hasProfanity } from "../../utils/censor.js";

export type EndpointsAuth = {
    emailLogIn: ApiEndpoint<EmailLogInInput, Session>;
    emailSignUp: ApiEndpoint<EmailSignUpInput, Session>;
    emailRequestPasswordChange: ApiEndpoint<EmailRequestPasswordChangeInput, Success>;
    emailResetPassword: ApiEndpoint<EmailResetPasswordInput, Session>;
    guestLogIn: ApiEndpoint<never, Session>;
    logOut: ApiEndpoint<never, Session>;
    logOutAll: ApiEndpoint<never, Session>;
    validateSession: ApiEndpoint<ValidateSessionInput, Session>;
    switchCurrentAccount: ApiEndpoint<SwitchCurrentAccountInput, Session>;
    walletInit: ApiEndpoint<WalletInitInput, WalletInit>;
    walletComplete: ApiEndpoint<WalletCompleteInput, WalletComplete>;
}

/** Expiry time for wallet authentication */
const NONCE_VALID_DURATION = MINUTES_5_MS;

/** Default user data */
const DEFAULT_USER_DATA = {
    isPrivateBookmarks: true,
    isPrivateVotes: true,
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
    emailLogIn: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Validate arguments with schema
        emailLogInFormValidation.validateSync(input, { abortEarly: false });
        // If email not supplied, check if session is valid. 
        // This is needed when this is called to verify an email address
        if (!input.email) {
            const user = RequestService.assertRequestFrom(req, { isOfficialUser: true });
            // Validate verification code
            if (input.verificationCode) {
                const email = await DbProvider.get().email.findFirst({
                    where: {
                        AND: [
                            { userId: BigInt(user.id) },
                            { verificationCode: input.verificationCode },
                        ],
                    },
                });
                if (!email) {
                    throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
                }
                const verified = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id, input.verificationCode);
                if (!verified) {
                    throw new CustomError("0132", "CannotVerifyEmailCode");
                }
            }
            return SessionService.createBaseSession(req);
        }
        // If email supplied, validate
        else {
            // Find user
            const user = await DbProvider.get().user.findFirst({
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
                const isCodeValid = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id, input.verificationCode);
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
    emailSignUp: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Validate input format
        emailSignUpValidation.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.name))
            throw new CustomError("0140", "BannedWord");
        // Check if email exists
        const existingEmail = await DbProvider.get().email.findUnique({ where: { emailAddress: input.email ?? "" } });
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
            DbProvider.get().user.create({
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
            DbProvider.get().session.create({
                data: {
                    device_info: deviceInfo,
                    expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                    last_refresh_at: new Date(),
                    ip_address: ipAddress,
                    revokedAt: null,
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
        const transactionResults = await DbProvider.get().$transaction(transactions);
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
    emailRequestPasswordChange: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Validate input format
        emailRequestPasswordChangeSchema.validateSync(input, { abortEarly: false });
        // Find user
        const user = await DbProvider.get().user.findFirst({
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
    emailResetPassword: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Validate input format
        emailResetPasswordSchema.validateSync(input, { abortEarly: false });
        // Find user
        const user = await DbProvider.get().user.findUnique({
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
            DbProvider.get().user_auth.update({
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
            DbProvider.get().user.update({
                where: { id: user.id },
                data: {
                    logInAttempts: 0,
                    lastLoginAttempt: new Date(),
                },
            }),
            // Revoke all existing password sessions, even ones using other authentication methods
            // You could also delete them, but this may be required for compliance reasons. 
            // We'll rely on a cron job to delete old sessions.
            DbProvider.get().session.updateMany({
                where: {
                    user: {
                        id: user.id,
                    },
                    revokedAt: null,
                },
                data: {
                    revokedAt: new Date(),
                },
            }),
            // Create new session
            DbProvider.get().session.create({
                data: {
                    device_info: deviceInfo,
                    expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                    last_refresh_at: new Date(),
                    ip_address: ipAddress,
                    revokedAt: null,
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
        const transactionResults = await DbProvider.get().$transaction(transactions);
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
    guestLogIn: async (_d, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        return establishGuestSession(res);
    },
    logOut: async (_d, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        const userData = SessionService.getUser(req);
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
        await DbProvider.get().session.updateMany({
            where: {
                id: { in: sessionIdsToRevoke.map(id => BigInt(id)) },
            },
            data: {
                revokedAt: new Date(),
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
    logOutAll: async (_d, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Get user data
        const userData = SessionService.getUser(req);
        const userId = userData?.id;
        // If user is already logged out, return a guest session
        if (!userId) {
            return establishGuestSession(res);
        }
        // Use raw query to revoke sessions and return their IDs
        const sessions = await DbProvider.get().$queryRaw`
                UPDATE "session"
                SET revokedAt = now()
                WHERE "user_id" = ${BigInt(userId)}
                RETURNING id;
            `;
        // Iterate through all revoked session IDs and close their sockets
        if (sessions && Array.isArray(sessions)) {
            // Assuming sessions is an array of objects like { id: bigint }
            for (const session of sessions as { id: bigint }[]) {
                if (session.id) {
                    // Convert BigInt to string for the function call
                    closeSessionSockets(session.id.toString());
                }
            }
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
    validateSession: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 5000, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });

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
    switchCurrentAccount: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 500, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });

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
    walletInit: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // // Make sure that wallet is on mainnet (i.e. starts with 'stake1')
        const deserializedStakingAddress = serializedAddressToBech32(input.stakingAddress);
        if (!deserializedStakingAddress.startsWith("stake1"))
            throw new CustomError("0149", "MustUseMainnet");
        // Generate nonce for handshake
        const nonce = await generateNonce(input.nonceDescription as string | undefined);
        // Find existing wallet data in database
        let walletData = await DbProvider.get().wallet.findUnique({
            where: {
                stakingAddress: input.stakingAddress,
            },
            select: {
                id: true,
                verifiedAt: true,
                userId: true,
            },
        });
        // If wallet exists, update with new nonce
        if (walletData) {
            await DbProvider.get().wallet.update({
                where: { id: walletData.id },
                data: {
                    nonce,
                    nonceCreationTime: new Date().toISOString(),
                },
            });
        }
        // If wallet data doesn't exist, create
        if (!walletData) {
            walletData = await DbProvider.get().wallet.create({
                data: {
                    stakingAddress: input.stakingAddress,
                    nonce,
                    nonceCreationTime: new Date().toISOString(),
                },
                select: {
                    id: true,
                    verifiedAt: true,
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
    walletComplete: async ({ input }, { req, res }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });
        // Find wallet with public address
        const walletData = await DbProvider.get().wallet.findUnique({
            where: { stakingAddress: input.stakingAddress },
            select: {
                id: true,
                nonce: true,
                nonceCreationTime: true,
                user: {
                    select: { id: true },
                },
                verifiedAt: true,
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
            const userData = await DbProvider.get().user.create({
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
                await DbProvider.get().user.update({
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
        const wallet = await DbProvider.get().wallet.update({
            where: { id: walletData.id },
            data: {
                verifiedAt: new Date().toISOString(),
                nonce: null,
                nonceCreationTime: null,
            },
            select: {
                id: true,
                verifiedAt: true,
                name: true,
                publicAddress: true,
                stakingAddress: true,
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
            },
        } as const;
    },
};
