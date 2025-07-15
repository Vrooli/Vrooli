import type { PrismaPromise } from "@prisma/client";
import { AUTH_PROVIDERS, AccountStatus, COOKIE, type EmailLogInInput, type EmailRequestPasswordChangeInput, type EmailResetPasswordInput, type EmailSignUpInput, MINUTES_5_MS, type Session, type Success, type SwitchCurrentAccountInput, type ValidateSessionInput, type WalletComplete, type WalletCompleteInput, type WalletInit, type WalletInitInput, emailLogInFormValidation, emailRequestPasswordChangeSchema, emailResetPasswordSchema, emailSignUpValidation, generatePK, generatePublicId, switchCurrentAccountSchema, validateSessionSchema } from "@vrooli/shared";
import type { Response } from "express";
import { AuthTokensService } from "../../auth/auth.js";
import { randomString, validateCode } from "../../auth/codes.js";
import { EMAIL_VERIFICATION_TIMEOUT, PasswordAuthService, type UserDataForPasswordAuth } from "../../auth/email.js";
import { JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "../../auth/jwt.js";
import { RequestService } from "../../auth/request.js";
import { SessionService } from "../../auth/session.js";
import { generateNonce, serializedAddressToBech32, verifySignedMessage } from "../../auth/wallet.js";
import { DbProvider } from "../../db/provider.js";
import { Award } from "../../events/awards.js";
import { CustomError, createInvalidCredentialsError } from "../../events/error.js";
import { Trigger } from "../../events/trigger.js";
import { DEFAULT_USER_NAME_LENGTH } from "../../models/base/user.js";
import { SocketService } from "../../sockets/io.js";
import type { ApiEndpoint, SessionData } from "../../types.js";
import { hasProfanity } from "../../utils/censor.js";

// AI_CHECK: TYPE_SAFETY=server-auth-type-safety-fixes | LAST: 2025-07-02 - Fixed OAuth config 'as any' casting and unsafe string casting in database queries

// Constants
const OAUTH_STATE_LENGTH = 32;
const OAUTH_STATE_EXPIRY_MINUTES = 15;
const MS_IN_SECOND = 1000;
const SECONDS_IN_MINUTE = 60;
const MINUTES_TO_MS = SECONDS_IN_MINUTE * MS_IN_SECOND;

// Type guards for safer type checking
function isValidStringId(value: unknown): value is string {
    return typeof value === "string" && value.trim().length > 0;
}

function isValidNonceDescription(value: unknown): value is string | undefined {
    return value === undefined || (typeof value === "string" && value.trim().length > 0);
}

function isValidPassword(value: unknown): value is string {
    return typeof value === "string" && value.length > 0;
}

function getSessionUserId(session: { users?: Array<{ id?: unknown }> }): string {
    const userId = session.users?.[0]?.id;
    if (!isValidStringId(userId)) {
        throw new CustomError("0323", "InvalidSessionUser");
    }
    return userId;
}

function validateUserId(value: unknown, context: string): string {
    if (!isValidStringId(value)) {
        throw new CustomError("0321", "InvalidArgs", { detail: `Invalid user ID in ${context}` });
    }
    return value;
}

export type OAuthInitiateInput = {
    resourceId: string;
    redirectUri: string;
};

export type OAuthInitiateResult = {
    authUrl: string;
    state: string;
};

export type OAuthCallbackInput = {
    code: string;
    state: string;
};

export type OAuthCallbackResult = {
    success: boolean;
    provider: string;
};

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
    oauthInitiate: ApiEndpoint<OAuthInitiateInput, OAuthInitiateResult>;
    oauthCallback: ApiEndpoint<OAuthCallbackInput, OAuthCallbackResult>;
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

async function establishGuestSession(res: Response): Promise<Session> {
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
                    throw createInvalidCredentialsError();
                }
                const verified = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id.toString(), input.verificationCode);
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
                throw createInvalidCredentialsError();
            }
            const email = user.emails.find(e => e.emailAddress === input.email);
            if (!email) {
                throw createInvalidCredentialsError();
            }
            // Check for password in database. If it doesn't exist, send a password reset link
            const passwordHash = PasswordAuthService.getAuthPassword(user);
            if (!passwordHash) {
                await PasswordAuthService.setupPasswordReset(user);
                throw new CustomError("0135", "MustResetPassword");
            }
            // Validate verification code, if supplied
            if (input.verificationCode) {
                const isCodeValid = await PasswordAuthService.validateEmailVerificationCode(email.emailAddress, user.id.toString(), input.verificationCode);
                if (!isCodeValid) {
                    throw new CustomError("0137", "CannotVerifyEmailCode");
                }
            }
            // Create new session
            if (!isValidPassword(input?.password)) {
                throw new CustomError("0325", "InvalidArgs", { detail: "Invalid password provided" });
            }
            const session = await PasswordAuthService.logIn(input.password, user, req);
            if (session) {
                // Set session token
                await AuthTokensService.generateSessionToken(res, session);
                return session;
            } else {
                throw createInvalidCredentialsError();
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

        // Create user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: input.name,
                theme: input.theme,
                status: AccountStatus.Unlocked,
                emails: {
                    create: [
                        {
                            id: generatePK(),
                            emailAddress: input.email,
                        },
                    ],
                },
                auths: {
                    create: {
                        id: generatePK(),
                        provider: AUTH_PROVIDERS.Password,
                        hashed_password: PasswordAuthService.hashPassword(input.password),
                    },
                },
                creditAccount: {
                    create: {
                        id: generatePK(),
                        currentBalance: BigInt(0),
                    },
                },
                ...DEFAULT_USER_DATA,
            },
            select: PasswordAuthService.selectUserForPasswordAuth(),
        });

        // Find the created password auth record's ID
        const createdAuthId = user.auths.find(a => a.provider === AUTH_PROVIDERS.Password)?.id;
        if (!createdAuthId) {
            // Use a valid error key like InternalError
            throw new CustomError("0580", "InternalError", { detail: "Failed to find created password auth record" });
        }

        // Create session AFTER transaction
        const recordedSessionData = await SessionService.createAndRecordSession(user.id.toString(), createdAuthId.toString(), req);
        // Pass the recordedSessionData which matches PrismaSessionData type
        const responseSession = await SessionService.createSession(user, recordedSessionData, req);
        await AuthTokensService.generateSessionToken(res, responseSession);

        // Trigger new account
        await Trigger(req.session.languages).accountNew(user.id.toString(), user.publicId, input.email);

        // Return session data
        return responseSession;
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
            throw createInvalidCredentialsError();
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
        // Find user with proper type validation
        const whereClause = input.id 
            ? { id: BigInt(validateUserId(input.id, "email reset password")) }
            : { publicId: validateUserId(input.publicId, "email reset password") };
        
        const user = await DbProvider.get().user.findUnique({
            where: whereClause,
            select: PasswordAuthService.selectUserForPasswordAuth(),
        });
        if (!user) {
            throw createInvalidCredentialsError();
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
            // Revoke all existing password sessions (may need to keep old ones around for compliance purposes)
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
                    id: generatePK(),
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
        const createdSessionData = transactionResults[3] as UserDataForPasswordAuth["sessions"][0];

        // Create final session object for response
        const responseSession = await SessionService.createSession(user, createdSessionData, req);
        // Set up session token
        await AuthTokensService.generateSessionToken(res, responseSession);
        return responseSession;
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
            SocketService.get().closeSessionSockets(sessionId);
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
                    SocketService.get().closeSessionSockets(session.id.toString());
                }
            }
        }
        // Clear socket connections for user
        if (userId) {
            SocketService.get().closeUserSockets(userId);
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
        if (!isValidNonceDescription(input.nonceDescription)) {
            throw new CustomError("0324", "InvalidArgs", { detail: "Invalid nonce description" });
        }
        const nonce = await generateNonce(input.nonceDescription);
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
                    id: generatePK(),
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
        if (!walletData) {
            throw createInvalidCredentialsError();
        }
        // If nonce expired, throw error
        if (!walletData.nonce || !walletData.nonceCreationTime || Date.now() - new Date(walletData.nonceCreationTime).getTime() > NONCE_VALID_DURATION) {
            throw new CustomError("0314", "NonceExpired");
        }

        // Verify that message was signed by wallet address
        const walletVerified = verifySignedMessage(input.stakingAddress, walletData.nonce, input.signedPayload);
        if (!walletVerified) {
            throw new CustomError("0151", "CannotVerifyWallet");
        }

        let userId: bigint | undefined = walletData.user?.id;
        let firstLogIn = false;
        // If you are not signed in
        if (!req.session.isLoggedIn) {
            // Wallet must be verified
            if (!walletData.verifiedAt) {
                throw new CustomError("0152", "NotYourWallet");
            }
            firstLogIn = true;
            // Create new user
            const userData = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: `user${randomString(DEFAULT_USER_NAME_LENGTH, "0123456789")}`,
                    wallets: {
                        connect: { id: walletData.id },
                    },
                    creditAccount: {
                        create: {
                            id: generatePK(),
                            currentBalance: BigInt(0),
                        },
                    },
                    ...DEFAULT_USER_DATA,
                },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            userId = userData.id;
            // Give user award for signing up
            await Award(userId.toString(), req.session.languages).update("AccountNew", 1);
        }
        // If you are signed in
        else {
            // If wallet is not verified, link it to your account
            if (!walletData.verifiedAt) {
                const currentUserId = getSessionUserId(req.session);
                await DbProvider.get().user.update({
                    where: { id: BigInt(currentUserId) },
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
        // Create session token if user exists
        let session: Session | undefined;
        if (userId) {
            // Fetch user data for session creation
            const userData = await DbProvider.get().user.findUnique({
                where: { id: userId },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            if (userData) {
                // Find the wallet auth record's ID
                const walletAuthId = userData.auths.find(a => a.provider === "wallet")?.id;
                if (walletAuthId) {
                    // Create session record
                    const recordedSessionData = await SessionService.createAndRecordSession(userId.toString(), walletAuthId.toString(), req);
                    // Create response session
                    session = await SessionService.createSession(userData, recordedSessionData, req);
                    await AuthTokensService.generateSessionToken(res, session);
                }
            }
        }
        
        return {
            __typename: "WalletComplete",
            firstLogIn,
            session,
            wallet: wallet ? {
                ...wallet,
                id: wallet.id.toString(),
                __typename: "Wallet" as const,
                verifiedAt: typeof wallet.verifiedAt === 'string' ? wallet.verifiedAt : wallet.verifiedAt?.toISOString() ?? new Date().toISOString(),
                user: undefined,
                team: undefined,
            } : undefined,
        };
    },
    oauthInitiate: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });

        const userData = SessionService.getUser(req);
        if (!userData?.id) {
            throw new CustomError("0002", "NotLoggedIn");
        }

        const resourceId = BigInt(input.resourceId);

        // Get the API resource with its config
        const resource = await DbProvider.get().resource.findUnique({
            where: { id: resourceId },
            include: {
                versions: {
                    where: { isLatest: true },
                    include: {
                        translations: {
                            where: { language: "en" },
                            take: 1,
                        },
                    },
                },
            },
        });

        if (!resource?.versions?.[0]) {
            throw new CustomError("0305", "ResourceNotFound", { id: input.resourceId });
        }

        const version = resource.versions[0];
        const { ApiVersionConfig } = await import("@vrooli/shared");
        
        // Validate config structure before creating ApiVersionConfig
        if (!version.config || typeof version.config !== "object") {
            throw new CustomError("0320", "InvalidArgs", { detail: "Invalid API resource configuration" });
        }
        
        // AI_CHECK: TYPE_SAFETY=phase1-4 | LAST: 2025-07-03 - Added missing __version property to ApiVersionConfig object
        const config = new ApiVersionConfig({ config: { ...version.config as Record<string, unknown>, __version: "1.0" } });

        if (config.authentication?.type !== "oauth2") {
            throw new CustomError("0315", "InvalidArgs", { expected: "oauth2", actual: config.authentication?.type });
        }

        const oauthSettings = config.authentication.settings;
        if (!oauthSettings?.clientId || !oauthSettings?.authUrl) {
            throw new CustomError("0316", "InvalidArgs");
        }

        // Generate state for CSRF protection
        const state = randomString(OAUTH_STATE_LENGTH);

        // Store state in Redis (in production) or in-memory for now
        // For production, use Redis with TTL
        const _stateKey = `oauth:state:${state}`;
        const stateData = {
            userId: userData.id,
            resourceId: input.resourceId,
            expires: Date.now() + OAUTH_STATE_EXPIRY_MINUTES * MINUTES_TO_MS, // 15 minutes
        };

        // TODO: Use Redis in production
        // await redis.setex(stateKey, 900, JSON.stringify(stateData));

        // For now, store in session
        if (!req.session.oauthState) {
            req.session.oauthState = {};
        }
        req.session.oauthState[state] = stateData;

        // Build authorization URL
        const authUrl = new URL(oauthSettings.authUrl);
        authUrl.searchParams.append("client_id", oauthSettings.clientId);
        authUrl.searchParams.append("redirect_uri", input.redirectUri);
        authUrl.searchParams.append("response_type", "code");
        authUrl.searchParams.append("state", state);

        if (oauthSettings.scopes?.length > 0) {
            authUrl.searchParams.append("scope", oauthSettings.scopes.join(" "));
        }

        return {
            authUrl: authUrl.toString(),
            state,
        };
    },
    oauthCallback: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        RequestService.assertRequestFrom(req, { isApiToken: false });

        const userData = SessionService.getUser(req);
        if (!userData?.id) {
            throw new CustomError("0002", "NotLoggedIn");
        }

        // Verify state from session
        const stateData = req.session.oauthState?.[input.state];
        if (!stateData || stateData.userId !== userData.id) {
            throw new CustomError("0317", "InvalidArgs");
        }

        // Check if state expired
        if (stateData.expires < Date.now()) {
            if (req.session.oauthState) {
                delete req.session.oauthState[input.state];
            }
            throw new CustomError("0318", "SessionExpired");
        }

        // Remove state after validation
        if (req.session.oauthState) {
            delete req.session.oauthState[input.state];
        }

        const resourceId = BigInt(stateData.resourceId);

        // Get the API resource with its config
        const resource = await DbProvider.get().resource.findUnique({
            where: { id: resourceId },
            include: {
                versions: {
                    where: { isLatest: true },
                    include: {
                        translations: {
                            where: { language: "en" },
                            take: 1,
                        },
                    },
                },
            },
        });

        if (!resource?.versions?.[0]) {
            throw new CustomError("0305", "ResourceNotFound", { id: stateData.resourceId });
        }

        const version = resource.versions[0];
        const { ApiVersionConfig } = await import("@vrooli/shared");
        
        // Validate config structure before creating ApiVersionConfig
        if (!version.config || typeof version.config !== "object") {
            throw new CustomError("0320", "InvalidArgs", { detail: "Invalid API resource configuration" });
        }
        
        // AI_CHECK: TYPE_SAFETY=phase1-4 | LAST: 2025-07-03 - Added missing __version property to ApiVersionConfig object
        const config = new ApiVersionConfig({ config: { ...version.config as Record<string, unknown>, __version: "1.0" } });
        const providerName = version.translations[0].name.toLowerCase();

        const oauthSettings = config.authentication?.settings;
        if (!oauthSettings?.clientId || !oauthSettings?.clientSecret || !oauthSettings?.tokenUrl) {
            throw new CustomError("0316", "InvalidArgs");
        }

        // Exchange code for tokens
        const tokenResponse = await fetch(oauthSettings.tokenUrl, {
            method: "POST",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded",
            },
            body: new URLSearchParams({
                grant_type: "authorization_code",
                code: input.code,
                client_id: oauthSettings.clientId,
                client_secret: oauthSettings.clientSecret,
                redirect_uri: oauthSettings.redirectUri || "",
            }),
        });

        if (!tokenResponse.ok) {
            const errorText = await tokenResponse.text();
            const { logger } = await import("../../events/logger.js");
            logger.error("OAuth token exchange failed", { error: errorText });
            throw new CustomError("0319", "ExternalServiceError");
        }

        const tokenData = await tokenResponse.json();

        // Calculate token expiry
        let tokenExpiresAt: Date | null = null;
        if (tokenData.expires_in) {
            tokenExpiresAt = new Date(Date.now() + tokenData.expires_in * MS_IN_SECOND);
        }

        // Store or update OAuth tokens
        const userId = BigInt(userData.id);
        await DbProvider.get().user_auth.upsert({
            where: {
                user_auth_provider_provider_user_id_unique: {
                    provider: providerName,
                    provider_user_id: tokenData.user_id || userData.id,
                },
            },
            create: {
                id: generatePK(),
                user_id: userId,
                provider: providerName,
                provider_user_id: tokenData.user_id || userData.id,
                access_token: tokenData.access_token,
                refresh_token: tokenData.refresh_token,
                token_expires_at: tokenExpiresAt,
                granted_scopes: tokenData.scope ? tokenData.scope.split(" ") : [],
                last_used_at: new Date(),
            },
            update: {
                access_token: tokenData.access_token,
                refresh_token: tokenData.refresh_token,
                token_expires_at: tokenExpiresAt,
                granted_scopes: tokenData.scope ? tokenData.scope.split(" ") : [],
                last_used_at: new Date(),
            },
        });

        return {
            __typename: "Success" as const,
            success: true,
            provider: providerName,
        };
    },
};
