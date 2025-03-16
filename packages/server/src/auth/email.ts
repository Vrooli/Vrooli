import { AUTH_PROVIDERS, DAYS_2_MS, MINUTES_15_MS, Session, TranslationKeyError } from "@local/shared";
import { AccountStatus, PrismaPromise, active_focus_mode, email, focus_mode, premium, session, user_auth, user_language } from "@prisma/client";
import bcrypt from "bcrypt";
import { Request } from "express";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { Notify } from "../notify/notify.js";
import { sendResetPasswordLink, sendVerificationLink } from "../tasks/email/queue.js";
import { randomString, validateCode } from "./codes.js";
import { REFRESH_TOKEN_EXPIRATION_MS } from "./jwt.js";
import { RequestService } from "./request.js";
import { SessionService } from "./session.js";

export const EMAIL_VERIFICATION_TIMEOUT = DAYS_2_MS;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 5;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 15;
const SOFT_LOCKOUT_DURATION = MINUTES_15_MS;
const EMAIL_VERIFICATION_CODE_LENGTH = 8;

export type UserDataForPasswordAuth = {
    id: string;
    handle: string | null;
    lastLoginAttempt: Date | null;
    logInAttempts: number;
    name: string;
    profileImage: string | null;
    theme: string;
    status: AccountStatus;
    updated_at: Date;
    activeFocusMode: Pick<active_focus_mode, "stopCondition" | "stopTime"> & {
        focusMode: Pick<focus_mode, "id" | "reminderListId">;
    } | null;
    auths: Pick<user_auth, "id" | "provider" | "hashed_password">[];
    emails: Pick<email, "emailAddress">[];
    languages: Pick<user_language, "language">[];
    premium: Pick<premium, "credits" | "expiresAt"> | null;
    sessions: (Pick<session, "id" | "device_info" | "ip_address" | "last_refresh_at" | "revoked"> & {
        auth: Pick<user_auth, "id" | "provider">;
    })[];
    _count: {
        apis: number;
        codes: number;
        memberships: number;
        notes: number;
        projects: number;
        questionsAsked: number;
        routines: number;
        standards: number;
    }
}

export class PasswordAuthService {
    private static instance: PasswordAuthService;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): PasswordAuthService {
        if (!PasswordAuthService.instance) {
            PasswordAuthService.instance = new PasswordAuthService();
        }
        return PasswordAuthService.instance;
    }

    static selectUserForPasswordAuth() {
        return {
            id: true,
            handle: true,
            lastLoginAttempt: true,
            logInAttempts: true,
            name: true,
            profileImage: true,
            status: true,
            theme: true,
            updated_at: true,
            activeFocusMode: {
                select: {
                    stopCondition: true,
                    stopTime: true,
                    focusMode: {
                        select: {
                            id: true,
                            reminderListId: true,
                        },
                    },
                },
            },
            auths: {
                select: {
                    id: true,
                    provider: true,
                    hashed_password: true,
                    resetPasswordCode: true,
                    lastResetPasswordRequestAttempt: true,
                },
            },
            emails: {
                select: {
                    emailAddress: true,
                },
            },
            languages: {
                select: {
                    language: true,
                },
            },
            premium: {
                select: {
                    credits: true,
                    expiresAt: true,
                },
            },
            sessions: {
                select: {
                    id: true,
                    device_info: true,
                    ip_address: true,
                    last_refresh_at: true,
                    revoked: true,
                    auth: {
                        select: {
                            id: true,
                            provider: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    apis: true,
                    codes: true,
                    memberships: true,
                    notes: true,
                    projects: true,
                    questionsAsked: true,
                    routines: true,
                    standards: true,
                },
            },
        } as const;
    }

    /**
     * Hashes password for safe storage in database
     * @param password Plaintext password
     * @returns Hashed password
     */
    static hashPassword(password: string): string {
        return bcrypt.hashSync(password, HASHING_ROUNDS);
    }

    static statusToError(status: AccountStatus): TranslationKeyError | null {
        if (status === "HardLocked") return "HardLockout";
        if (status === "SoftLocked") return "SoftLockout";
        if (status === "Deleted") return "AccountDeleted";
        return null;
    }

    /**
     * Finds the password in the user's "auths" relationship array
     * @param user User object containing auths array
     * @returns Hashed password or null
     */
    static getAuthPassword({ auths }: Pick<UserDataForPasswordAuth, "auths">): string | null {
        const passwordTable = auths.find((auth) => auth.provider === AUTH_PROVIDERS.Password);
        return passwordTable?.hashed_password ?? null;
    }

    /**
     * Validates a user's password, taking into account the user's account status
     * @param plaintext Plaintext password to check
     * @param user User object
     * @returns Boolean indicating if the password is valid
     */
    static validatePassword(plaintext: string, user: UserDataForPasswordAuth): boolean {
        if (!Array.isArray(user.auths) || user.auths.length === 0) return false;
        const passwordHash = this.getAuthPassword(user);
        if (!passwordHash) return false;
        // A password is only valid if the user is:
        // 1. Not deleted
        // 2. Not locked out
        // If account is deleted or locked, throw error
        const accountError = this.statusToError(user.status);
        if (accountError !== null) throw new CustomError("0050", accountError);
        // Validate plaintext password against hash
        return bcrypt.compareSync(plaintext, passwordHash);
    }

    static isLogInAttemptsResettable(user: Pick<UserDataForPasswordAuth, "lastLoginAttempt" | "status">): boolean {
        // Account must not be deleted or hard-locked
        if (user.status === "HardLocked" || user.status === "Deleted") return false;
        // Last login attempt must exist
        if (!user.lastLoginAttempt) return false;
        // Can reset if last login attempt was more than SOFT_LOCKOUT_DURATION ago
        const timeSinceLastAttemptMs = Date.now() - new Date(user.lastLoginAttempt).getTime();
        return timeSinceLastAttemptMs > SOFT_LOCKOUT_DURATION;
    }

    /**
     * Attemps to log a user in
     * @param password Plaintext password
     * @param user User object
     * @param req Express request object
     * @returns Session data
     */
    static async logIn(
        password: string,
        user: UserDataForPasswordAuth,
        req: Request,
    ): Promise<Session | null> {
        // First, check if the login attempts are resettable
        const willResetLoginAttempts = this.isLogInAttemptsResettable(user);

        // If account is deleted or locked, throw error
        const accountError = this.statusToError(user.status);
        if (accountError) {
            // For soft lockout, check if login attempts will reset
            if (accountError === "SoftLockout" && !willResetLoginAttempts) {
                throw new CustomError("0060", accountError);
            }
            // For hard lockout or deleted accounts, always throw an error
            else if (accountError !== "SoftLockout") {
                throw new CustomError("0070", accountError);
            }
        }

        // Get device info and IP address
        const deviceInfo = RequestService.getDeviceInfo(req);
        const ipAddress = req.ip;

        // Find the matching session
        const matchingSession = user.sessions.find((s) =>
            s.device_info === deviceInfo
            && s.auth.provider === AUTH_PROVIDERS.Password
            && s.revoked === false,
        );

        // Find matching auth
        const passwordAuth = user.auths.find(a => a.provider === AUTH_PROVIDERS.Password);
        if (!passwordAuth) {
            throw new CustomError("0577", "InternalError");
        }

        // If password is valid
        if (this.validatePassword(password, user)) {
            const transactions: PrismaPromise<object>[] = [
                // Remove reset password reset code and log in attempts from auth row
                DbProvider.get().user_auth.update({
                    where: {
                        id: passwordAuth.id,
                    },
                    data: {
                        resetPasswordCode: null,
                        lastResetPasswordRequestAttempt: null,
                    },
                }),
                // Update log in attempts and last login attempt in user row
                DbProvider.get().user.update({
                    where: { id: user.id },
                    data: {
                        logInAttempts: 0,
                        lastLoginAttempt: new Date(),
                    },
                    select: { id: true },
                }),
            ];
            // Upsert session
            if (matchingSession) {
                transactions.push(
                    DbProvider.get().session.update({
                        where: { id: matchingSession.id },
                        data: {
                            expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                            ip_address: ipAddress,
                            last_refresh_at: new Date(),
                        },
                        select: this.selectUserForPasswordAuth().sessions.select,
                    }),
                );
            } else {
                transactions.push(
                    DbProvider.get().session.create({
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
                        select: this.selectUserForPasswordAuth().sessions.select,
                    }),
                );
            }
            const transactionResults = await DbProvider.get().$transaction(transactions);
            // transactions[0]: user_auth.update
            // transactions[1]: user.update
            // transactions[2]: session.update/create
            const sessionData = transactionResults[2] as UserDataForPasswordAuth["sessions"][0];

            // If session is new
            if (!matchingSession) {
                // TODO When 2FA is ready, handle it here (if enabled by user)
            }

            return await SessionService.createSession(user, sessionData, req);
        }
        // Reset log in fail counter if possible
        if (willResetLoginAttempts) {
            await DbProvider.get().user.update({
                where: { id: user.id },
                data: { logInAttempts: 1 }, // We just failed to log in, so set to 1
            });
        }
        // Otherwise, increment log in fail counter and update account status
        else {
            let new_status: AccountStatus = AccountStatus.Unlocked;
            const log_in_attempts = user.logInAttempts + 1;
            if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
                new_status = AccountStatus.HardLocked;
            } else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
                new_status = AccountStatus.SoftLocked;
            }
            await DbProvider.get().user.update({
                where: { id: user.id },
                data: {
                    status: new_status,
                    logInAttempts: log_in_attempts,
                    lastLoginAttempt: new Date(),
                },
            });
        }
        return null;
    }

    /**
     * Generates a URL-safe code for account confirmations and password resets
     * @param length Length of code to generate
     */
    static generateEmailVerificationCode(length?: number): string {
        return randomString(length ?? EMAIL_VERIFICATION_CODE_LENGTH);
    }

    /**
     * Updated user object with new password reset code, and sends email to user with reset link
     * @param user User object
     */
    static async setupPasswordReset(user: Pick<UserDataForPasswordAuth, "id" | "emails" | "auths">): Promise<boolean> {
        // Make sure the user has at least one email to send the reset link to
        if (!user.emails || user.emails.length === 0) {
            throw new CustomError("0063", "NoEmails");
        }
        // Generate new code
        const resetPasswordCode = this.generateEmailVerificationCode();
        const commonAuthData = {
            resetPasswordCode,
            lastResetPasswordRequestAttempt: new Date(),
        } as const;
        // Find the password auth
        const passwordAuth = user.auths.find(a => a.provider === AUTH_PROVIDERS.Password);
        // If no password auth exists, create one. 
        // This may happen if the account was created with an OAuth provider.
        if (!passwordAuth) {
            await DbProvider.get().user_auth.create({
                data: {
                    ...commonAuthData,
                    provider: AUTH_PROVIDERS.Password,
                    user: {
                        connect: { id: user.id },
                    },
                },
            });
        }
        // Otherwise, update existing password auth
        else {
            await DbProvider.get().user_auth.update({
                where: {
                    id: passwordAuth.id,
                },
                data: commonAuthData,
            });
        }
        // Send new verification emails
        for (const email of user.emails) {
            sendResetPasswordLink(email.emailAddress, user.id, resetPasswordCode);
        }
        return true;
    }

    static emailVerificationCodeSelect() {
        return {
            id: true,
            lastVerificationCodeRequestAttempt: true,
            verified: true,
            user: {
                select: {
                    id: true,
                },
            },
        } as const;
    }

    /**
    * Updates email object with new verification code, and sends email to user with link.
    * When the user clicks the link, they'll be directed to a page in our UI.
    * This page sends an `emailLogin` mutation to the server to verify the code.
    */
    static async setupEmailVerificationCode(
        emailAddress: string,
        userId: string,
        languages: string[] | undefined,
    ): Promise<void> {
        // Find the email
        let email = await DbProvider.get().email.findUnique({
            where: { emailAddress },
            select: this.emailVerificationCodeSelect(),
        });
        // Create if not found
        if (!email) {
            email = await DbProvider.get().email.create({
                data: {
                    emailAddress,
                    user: {
                        connect: { id: userId },
                    },
                },
                select: this.emailVerificationCodeSelect(),
            });
        }
        // Check if it belongs to the user
        if (email.user && email.user.id !== userId) {
            throw new CustomError("0064", "EmailNotYours");
        }
        // Check if it's already verified
        if (email.verified) {
            throw new CustomError("0059", "EmailAlreadyVerified");
        }
        // Generate new code
        const verificationCode = this.generateEmailVerificationCode();
        // Store code and request time
        await DbProvider.get().email.update({
            where: { id: email.id },
            data: {
                verificationCode,
                lastVerificationCodeRequestAttempt: new Date(),
            },
        });
        // Send new verification email
        sendVerificationLink(emailAddress, userId, verificationCode);
        // Warn of new verification email to existing devices (if applicable)
        Notify(languages).pushNewEmailVerification().toUser(userId);
    }

    /**
     * Validates email verification code and update user's account status
     * @param emailAddress Email address string
     * @param userId ID of user who owns email
     * @param code Verification code
     * @param languages Preferred languages to display error messages in
     * @returns True if email was is verified
     */
    static async validateEmailVerificationCode(
        emailAddress: string,
        userId: string,
        code: string,
    ): Promise<boolean> {
        // Find data
        const email = await DbProvider.get().email.findUnique({
            where: { emailAddress },
            select: {
                id: true,
                userId: true,
                verified: true,
                verificationCode: true,
                lastVerificationCodeRequestAttempt: true,
            },
        });
        if (!email)
            throw new CustomError("0062", "CannotVerifyEmailCode"); // Purposefully vague with duplicate code for security
        // Check that userId matches email's userId
        if (email.userId !== userId)
            throw new CustomError("0062", "CannotVerifyEmailCode"); // Purposefully vague with duplicate code for security
        // If email already verified, remove old verification code
        if (email.verified) {
            await DbProvider.get().email.update({
                where: { id: email.id },
                data: { verificationCode: null, lastVerificationCodeRequestAttempt: null },
            });
            return true;
        }
        // Otherwise, validate code
        else {
            // The code may contain other information, such as the user's ID. 
            // Remove anything before and including the last colon
            const colonIndex = code.lastIndexOf(":");
            if (colonIndex !== -1) {
                code = code.slice(colonIndex + 1);
            }
            // If code is correct and not expired
            if (validateCode(code, email.verificationCode, email.lastVerificationCodeRequestAttempt, EMAIL_VERIFICATION_TIMEOUT)) {
                await DbProvider.get().email.update({
                    where: { id: email.id },
                    data: {
                        verified: true,
                        lastVerifiedTime: new Date(),
                        verificationCode: null,
                        lastVerificationCodeRequestAttempt: null,
                    },
                });
                return true;
            }
            // We used to send a new verification email here, but have changed this in favor 
            // of making the user request a new verification code. This is to prevent spamming 
            // users with emails.
            return false;
        }
    }
}
