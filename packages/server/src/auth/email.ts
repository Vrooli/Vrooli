import { ErrorKey, Session } from "@local/shared";
import { AccountStatus } from "@prisma/client";
import bcrypt from "bcrypt";
import { Request } from "express";
import { CustomError } from "../events/error";
import { UserModelInfo } from "../models/base/types";
import { Notify } from "../notify/notify";
import { sendResetPasswordLink, sendVerificationLink } from "../tasks/email/queue";
import { PrismaType } from "../types";
import { randomString, validateCode } from "./codes";
import { toSession } from "./session";

export const EMAIL_VERIFICATION_TIMEOUT = 2 * 24 * 3600 * 1000; // 2 days
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 5;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 15;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000; // 15 minutes

/**
 * Generates a URL-safe code for account confirmations and password resets
 * @param length Length of code to generate
 */
export const generateEmailVerificationCode = (length = 8): string => {
    return randomString(length);
};

/**
 * Hashes password for safe storage in database
 * @param password Plaintext password
 * @returns Hashed password
 */
export const hashPassword = (password: string): string => {
    return bcrypt.hashSync(password, HASHING_ROUNDS);
};

const statusToError = (status: AccountStatus): ErrorKey | null => {
    if (status === "HardLocked") return "HardLockout";
    if (status === "SoftLocked") return "SoftLockout";
    if (status === "Deleted") return "AccountDeleted";
    return null;
};

/**
 * Validates a user's password, taking into account the user's account status
 * @param plaintext Plaintext password to check
 * @param user User object
 * @param languages Preferred languages to display error messages in
 * @returns Boolean indicating if the password is valid
 */
export const validatePassword = (plaintext: string, user: Pick<UserModelInfo["PrismaModel"], "status" | "password">, languages: string[]): boolean => {
    if (!user.password) return false;
    // A password is only valid if the user is:
    // 1. Not deleted
    // 2. Not locked out
    // If account is deleted or locked, throw error
    const accountError = statusToError(user.status);
    if (accountError !== null) throw new CustomError("0050", accountError, languages);
    // Validate plaintext password against hash
    return bcrypt.compareSync(plaintext, user.password);
};

/**
 * Attemps to log a user in
 * @param password Plaintext password
 * @param user User object
 * @param prisma Prisma type
 * @param req Express request object
 * @returns Session data
 */
export const logIn = async (
    password: string,
    user: Pick<UserModelInfo["PrismaModel"], "id" | "lastLoginAttempt" | "logInAttempts" | "status" | "password">,
    prisma: PrismaType,
    req: Request,
): Promise<Session | null> => {
    // First, check if the log in fail counter should be reset
    // If account is NOT deleted or hard-locked, and lockout duration has passed
    if (user.status !== "HardLocked" && user.status !== "Deleted" && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
        // Reset log in fail counter
        await prisma.user.update({
            where: { id: user.id },
            data: { logInAttempts: 0 },
        });
    }
    // If account is deleted or locked, throw error
    const accountError = statusToError(user.status);
    if (accountError !== null) throw new CustomError("0060", accountError, req.session.languages);
    // If password is valid
    if (validatePassword(password, user, req.session.languages)) {
        const userData = await prisma.user.update({
            where: { id: user.id },
            data: {
                logInAttempts: 0,
                lastLoginAttempt: new Date().toISOString(),
                resetPasswordCode: null,
                lastResetPasswordReqestAttempt: null,
            },
            select: { id: true },
        });
        return await toSession(userData, prisma, req);
    }
    // If password is invalid
    let new_status: AccountStatus = AccountStatus.Unlocked;
    const log_in_attempts = user.logInAttempts++;
    if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
        new_status = AccountStatus.HardLocked;
    } else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
        new_status = AccountStatus.SoftLocked;
    }
    await prisma.user.update({
        where: { id: user.id },
        data: { status: new_status, logInAttempts: log_in_attempts, lastLoginAttempt: new Date().toISOString() },
    });
    return null;
};

/**
 * Updated user object with new password reset code, and sends email to user with reset link
 * @param user User object
 */
export const setupPasswordReset = async (user: { id: string, resetPasswordCode: string | null }, prisma: PrismaType): Promise<boolean> => {
    // Generate new code
    const resetPasswordCode = generateEmailVerificationCode();
    // Store code and request time in user row
    const updatedUser = await prisma.user.update({
        where: { id: user.id },
        data: { resetPasswordCode, lastResetPasswordReqestAttempt: new Date().toISOString() },
        select: { emails: { select: { emailAddress: true } } },
    });
    // Send new verification emails
    for (const email of updatedUser.emails) {
        sendResetPasswordLink(email.emailAddress, user.id, resetPasswordCode);
    }
    return true;
};

const emailSelect = {
    id: true,
    lastVerificationCodeRequestAttempt: true,
    verified: true,
    user: {
        select: {
            id: true,
        },
    },
} as const;

/**
* Updates email object with new verification code, and sends email to user with link.
* When the user clicks the link, they'll be directed to a page in our UI.
* This page sends an `emailLogin` mutation to the server to verify the code.
*/
export const setupEmailVerificationCode = async (
    emailAddress: string,
    userId: string,
    prisma: PrismaType,
    languages: string[],
): Promise<void> => {
    // Find the email
    let email = await prisma.email.findUnique({
        where: { emailAddress },
        select: emailSelect,
    });
    // Create if not found
    if (!email) {
        email = await prisma.email.create({
            data: {
                emailAddress,
                user: {
                    connect: { id: userId },
                },
            },
            select: emailSelect,
        });
    }
    // Check if it belongs to the user
    if (email.user && email.user.id !== userId) {
        throw new CustomError("0061", "EmailNotYours", languages);
    }
    // Check if it's already verified
    if (email.verified) {
        throw new CustomError("0059", "EmailAlreadyVerified", languages);
    }
    // Generate new code
    const verificationCode = generateEmailVerificationCode();
    // Store code and request time
    await prisma.email.update({
        where: { id: email.id },
        data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
    });
    // Send new verification email
    sendVerificationLink(emailAddress, userId, verificationCode);
    // Warn of new verification email to existing devices (if applicable)
    Notify(prisma, languages).pushNewEmailVerification().toUser(userId);
};

/**
 * Validates email verification code and update user's account status
 * @param emailAddress Email address string
 * @param userId ID of user who owns email
 * @param code Verification code
 * @param prisma The Prisma client
 * @param languages Preferred languages to display error messages in
 * @returns True if email was is verified
 */
export const validateEmailVerificationCode = async (
    emailAddress: string,
    userId: string,
    code: string,
    prisma: PrismaType,
    languages: string[],
): Promise<boolean> => {
    // Find data
    const email = await prisma.email.findUnique({
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
        throw new CustomError("0062", "EmailNotFound", languages);
    // Check that userId matches email's userId
    if (email.userId !== userId)
        throw new CustomError("0063", "EmailNotYours", languages);
    // If email already verified, remove old verification code
    if (email.verified) {
        await prisma.email.update({
            where: { id: email.id },
            data: { verificationCode: null, lastVerificationCodeRequestAttempt: null },
        });
        return true;
    }
    // Otherwise, validate code
    else {
        // If code is correct and not expired
        if (validateCode(code, email.verificationCode, email.lastVerificationCodeRequestAttempt, EMAIL_VERIFICATION_TIMEOUT)) {
            await prisma.email.update({
                where: { id: email.id },
                data: {
                    verified: true,
                    lastVerifiedTime: new Date().toISOString(),
                    verificationCode: null,
                    lastVerificationCodeRequestAttempt: null,
                },
            });
            return true;
        }
        // If email is not verified, set up new verification code
        else if (!email.verified) {
            await setupEmailVerificationCode(emailAddress, userId, prisma, languages);
        }
        return false;
    }
};
