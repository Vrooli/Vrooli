import { AccountStatus } from "@prisma/client";
import bcrypt from "bcrypt";
import { CustomError } from "../events";
import { Notify, sendResetPasswordLink, sendVerificationLink } from "../notify";
import { toSession } from "./session";
const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
export const generateCode = () => {
    return bcrypt.genSaltSync(HASHING_ROUNDS).replace("/", "");
};
export const validateCode = (providedCode, storedCode, dateRequested) => {
    return Boolean(providedCode) && Boolean(storedCode) && Boolean(dateRequested) &&
        providedCode === storedCode && Date.now() - new Date(dateRequested).getTime() < CODE_TIMEOUT;
};
export const hashPassword = (password) => {
    return bcrypt.hashSync(password, HASHING_ROUNDS);
};
export const validatePassword = (plaintext, user, languages) => {
    if (!user.password)
        return false;
    if (user.status === "HardLocked")
        throw new CustomError("0060", "HardLockout", languages);
    if (user.status === "SoftLocked")
        throw new CustomError("0330", "SoftLockout", languages);
    if (user.status === "Deleted")
        throw new CustomError("0061", "AccountDeleted", languages);
    return bcrypt.compareSync(plaintext, user.password);
};
export const logIn = async (password, user, prisma, req) => {
    if (user.status !== "HardLocked" && user.status !== "Deleted" && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
        await prisma.user.update({
            where: { id: user.id },
            data: { logInAttempts: 0 },
        });
    }
    if (user.status === AccountStatus.HardLocked)
        throw new CustomError("0060", "HardLockout", req.languages);
    if (user.status === AccountStatus.SoftLocked)
        throw new CustomError("0331", "SoftLockout", req.languages);
    if (user.status === AccountStatus.Deleted)
        throw new CustomError("0061", "AccountDeleted", req.languages);
    if (validatePassword(password, user, req.languages)) {
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
    let new_status = AccountStatus.Unlocked;
    const log_in_attempts = user.logInAttempts++;
    if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
        new_status = AccountStatus.HardLocked;
    }
    else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
        new_status = AccountStatus.SoftLocked;
    }
    await prisma.user.update({
        where: { id: user.id },
        data: { status: new_status, logInAttempts: log_in_attempts, lastLoginAttempt: new Date().toISOString() },
    });
    return null;
};
export const setupPasswordReset = async (user, prisma) => {
    const resetPasswordCode = generateCode();
    const updatedUser = await prisma.user.update({
        where: { id: user.id },
        data: { resetPasswordCode, lastResetPasswordReqestAttempt: new Date().toISOString() },
        select: { emails: { select: { emailAddress: true } } },
    });
    for (const email of updatedUser.emails) {
        sendResetPasswordLink(email.emailAddress, user.id, resetPasswordCode);
    }
    return true;
};
export const setupVerificationCode = async (emailAddress, prisma, languages) => {
    const verificationCode = generateCode();
    const email = await prisma.email.update({
        where: { emailAddress },
        data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
        select: { userId: true },
    });
    if (!email.userId)
        throw new CustomError("0061", "EmailNotYours", languages);
    sendVerificationLink(emailAddress, email.userId, verificationCode);
    Notify(prisma, languages).pushNewEmailVerification().toUser(email.userId);
};
export const validateVerificationCode = async (emailAddress, userId, code, prisma, languages) => {
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
    if (email.userId !== userId)
        throw new CustomError("0063", "EmailNotYours", languages);
    if (email.verified) {
        await prisma.email.update({
            where: { id: email.id },
            data: { verificationCode: null, lastVerificationCodeRequestAttempt: null },
        });
        return true;
    }
    else {
        if (validateCode(code, email.verificationCode, email.lastVerificationCodeRequestAttempt)) {
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
        else if (!email.verified) {
            await setupVerificationCode(emailAddress, prisma, languages);
        }
        return false;
    }
};
//# sourceMappingURL=email.js.map