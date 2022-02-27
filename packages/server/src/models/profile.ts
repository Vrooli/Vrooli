import { PrismaType, RecursivePartial } from "types";
import { Profile, ProfileEmailUpdateInput, ProfileUpdateInput, Session, Success, Tag, TagSearchInput, User, UserDeleteInput } from "../schema/types";
import { sendResetPasswordLink, sendVerificationLink } from "../worker/email/queue";
import { addJoinTablesHelper, FormatConverter, InfoType, modelToGraphQL, PaginatedSearchResult, readManyHelper, readOneHelper, removeJoinTablesHelper, selectHelper } from "./base";
import { user } from "@prisma/client";
import { CODE, ROLES } from "@local/shared";
import { CustomError } from "../error";
import bcrypt from 'bcrypt';
import { hasProfanity } from "../utils/censor";
import { TagModel, tagSearcher } from "./tag";
import { EmailModel } from "./email";
import pkg from '@prisma/client';
const { AccountStatus } = pkg;

const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
const EXPORT_LIMIT = 3;
const EXPORT_LIMIT_TIMEOUT = 24 * 3600 * 1000;

const joinMapper = { hiddenTags: 'tag', roles: 'role', starredBy: 'user' };
export const profileFormatter = (): FormatConverter<User> => ({
    removeCalculatedFields: (partial) => {
        let { starredTags, hiddenTags, ...rest } = partial;
        return rest;
    },
    addJoinTables: (partial) => {
        return addJoinTablesHelper(partial, joinMapper);
    },
    removeJoinTables: (data) => {
        return removeJoinTablesHelper(data, joinMapper);
    },
})

/**
 * Custom component for email/password validation
 */
export const profileValidater = () => ({
    /**
     * Generates a URL-safe code for account confirmations and password resets
     * @returns Hashed and salted code, with invalid characters removed
     */
    generateCode(): string {
        return bcrypt.genSaltSync(HASHING_ROUNDS).replace('/', '')
    },
    /**
     * Verifies if a confirmation or password reset code is valid
     * @param providedCode Code provided by GraphQL mutation
     * @param storedCode Code stored in user cell in database
     * @param dateRequested Date of request, also stored in database
     * @returns Boolean indicating if the code is valid
     */
    validateCode(providedCode: string | null, storedCode: string | null, dateRequested: Date | null): boolean {
        return Boolean(providedCode) && Boolean(storedCode) && Boolean(dateRequested) &&
            providedCode === storedCode && Date.now() - new Date(dateRequested as Date).getTime() < CODE_TIMEOUT;
    },
    /**
     * Hashes password for safe storage in database
     * @param password Plaintext password
     * @returns Hashed password
     */
    hashPassword(password: string): string {
        return bcrypt.hashSync(password, HASHING_ROUNDS)
    },
    /**
     * Validates a user's password, taking into account the user's account status
     * @param plaintext Plaintext password to check
     * @param user User object
     * @returns Boolean indicating if the password is valid
     */
    validatePassword(plaintext: string, user: any): boolean {
        // A password is only valid if the user is:
        // 1. Not deleted
        // 2. Not locked out
        const status_to_code: any = {
            [AccountStatus.Deleted]: CODE.NoUser,
            [AccountStatus.SoftLocked]: CODE.SoftLockout,
            [AccountStatus.HardLocked]: CODE.HardLockout
        }
        if (user.status in status_to_code) throw new CustomError(status_to_code[user.status]);
        // Validate plaintext password against hash
        return bcrypt.compareSync(plaintext, user.password)
    },
    /**
     * Attemps to log a user in
     * @param password Plaintext password
     * @param user User object
     * @param info Prisma query info
     * @returns Session data
     */
    async logIn(password: string, user: any, prisma: PrismaType): Promise<Session | null> {
        // First, check if the log in fail counter should be reset
        const unable_to_reset = [AccountStatus.HardLocked, AccountStatus.Deleted];
        // If account is not deleted or hard-locked, and lockout duration has passed
        if (!unable_to_reset.includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
            // Reset log in fail counter
            await prisma.user.update({
                where: { id: user.id },
                data: { logInAttempts: 0 },
            });
        }
        // If account is deleted or hard-locked, throw error
        if (unable_to_reset.includes(user.status)) throw new CustomError(CODE.BadCredentials);
        // If password is valid
        if (this.validatePassword(password, user)) {
            const userData = await prisma.user.update({
                where: { id: user.id },
                data: {
                    logInAttempts: 0,
                    lastLoginAttempt: new Date().toISOString(),
                    resetPasswordCode: null,
                    lastResetPasswordReqestAttempt: null
                },
                select: {
                    id: true,
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
            return this.toSession(userData);
        }
        // If password is invalid
        let new_status: any = AccountStatus.Unlocked;
        let log_in_attempts = user.logInAttempts++;
        if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
            new_status = AccountStatus.HardLocked;
        } else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
            new_status = AccountStatus.SoftLocked;
        }
        await prisma.user.update({
            where: { id: user.id },
            data: { status: new_status, logInAttempts: log_in_attempts, lastLoginAttempt: new Date().toISOString() }
        })
        return null;
    },
    /**
     * Updated user object with new password reset code, and sends email to user with reset link
     * @param user User object
     */
    async setupPasswordReset(user: any, prisma: PrismaType): Promise<boolean> {
        // Generate new code
        const resetPasswordCode = this.generateCode();
        // Store code and request time in user row
        const updatedUser = await prisma.user.update({
            where: { id: user.id },
            data: { resetPasswordCode, lastResetPasswordReqestAttempt: new Date().toISOString() },
            select: { emails: { select: { emailAddress: true } } }
        })
        // Send new verification emails
        for (const email of updatedUser.emails) {
            sendResetPasswordLink(email.emailAddress, user.id, resetPasswordCode);
        }
        return true;
    },
    /**
    * Updates email object with new verification code, and sends email to user with link
    * @param user User object
    */
    async setupVerificationCode(emailAddress: string, prisma: PrismaType): Promise<void> {
        // Generate new code
        const verificationCode = this.generateCode();
        // Store code and request time in email row
        const email = await prisma.email.update({
            where: { emailAddress },
            data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
            select: { userId: true }
        })
        // If email is not associated with a user, throw error
        if (!email.userId) throw new CustomError(CODE.ErrorUnknown, 'Email not associated with a user');
        // Send new verification email
        sendVerificationLink(emailAddress, email.userId, verificationCode);
        // TODO send email to existing emails from user, warning of new email
    },
    /**
     * Validate verification code and update user's account status
     * @param emailAddress Email address string
     * @param userId ID of user who owns email
     * @param code Verification code
     * @returns Updated user object
     */
    async validateVerificationCode(emailAddress: string, userId: string, code: string, prisma: PrismaType): Promise<boolean> {
        // Find email data
        const email: any = prisma.email.findUnique({
            where: { emailAddress },
            select: {
                id: true,
                userId: true,
                verified: true,
                verificationCode: true,
                lastVerificationCodeRequestAttempt: true
            }
        })
        if (!email) throw new CustomError(CODE.EmailNotVerified, 'Email not found');
        // Check that userId matches email's userId
        if (email.userId !== userId) throw new CustomError(CODE.EmailNotVerified, 'Email does not belong to user');
        // If email already verified, remove old verification code
        if (email.verified) {
            await prisma.email.update({
                where: { id: email.id },
                data: { verificationCode: null, lastVerificationCodeRequestAttempt: null }
            })
        }
        // Otherwise, validate code
        else {
            // If code is correct and not expired
            if (this.validateCode(code, email.verificationCode, email.lastVerificationCodeRequestAttempt)) {
                await prisma.email.update({
                    where: { id: email.id },
                    data: { verified: true, verificationCode: null, lastVerificationCodeRequestAttempt: null }
                })
            }
            // If code is incorrect or expired, create new code and send email
            else {
                await this.setupVerificationCode(emailAddress, prisma);
            }
        }
        return true;
    },
    /**
     * Creates session object from user
     */
    toSession(user: RecursivePartial<user>): Session {
        return {
            id: user.id ?? '',
            theme: user.theme ?? 'light',
            roles: [ROLES.Actor],
        }
    }
})

/**
 * Custom component for importing/exporting data from Vrooli
 * @param state 
 * @returns 
 */
const porter = (prisma: PrismaType) => ({
    /**
     * Import JSON data to Vrooli. Useful if uploading data created offline, or if
     * you're switching from a competitor to Vrooli. :)
     * @param id 
     */
    async importData(data: string): Promise<Success> {
        throw new CustomError(CODE.NotImplemented);
    },
    /**
     * Export data to JSON. Useful if you want to use Vrooli data on your own,
     * or switch to a competitor :(
     * @param id 
     */
    async exportData(id: string): Promise<string> {
        // Find user
        const user = await prisma.user.findUnique({ where: { id }, select: { numExports: true, lastExport: true } });
        if (!user) throw new CustomError(CODE.ErrorUnknown);
        // Check if export is allowed TODO export reset and whatnot
        if (user.numExports >= EXPORT_LIMIT) throw new CustomError(CODE.ExportLimitReached);
        throw new CustomError(CODE.NotImplemented)
    },
})

const profileQuerier = (prisma: PrismaType) => ({
    async findProfile(
        userId: string,
        info: InfoType,
    ): Promise<RecursivePartial<Profile> | null> {
        const profileData = await readOneHelper(userId, { id: userId }, info, prisma);
        const starredTags: Tag[] = (await prisma.star.findMany({
            where: {
                AND: [
                    { byId: userId },
                    { NOT: { tagId: null } }
                ]
            },
            select: {
                tag: {
                    select: {
                        id: true,
                        tag: true,
                        description: true,
                        stars: true,
                    }
                }
            }
        })).map(({ tag }) => tag) as Tag[]
        const hiddenTags: Tag[] = (await prisma.user_tag_hidden.findMany({
            where: { userId: userId },
            select: {
                tag: {
                    select: {
                        id: true,
                        tag: true,
                        description: true,
                        stars: true,
                    }
                }
            }
        })).map(({ tag }) => tag) as Tag[]
        return { ...profileData, starredTags, hiddenTags };
    },
    /**
     * Custom search for finding tags you have starred/hidden
     */
    async myTags(
        where: { [x: string]: any },
        userId: string | null,
        input: TagSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        // If myId or hidden specified, limit results.
        let idsLimit: string[] | undefined = undefined;
        // Looking for tags the requesting user has starred
        if (userId && input.myTags) {
            idsLimit = (await prisma.star.findMany({
                where: {
                    AND: [
                        { byId: userId },
                        { NOT: { tagId: null } }
                    ]
                }
            })).map(s => s.tagId).filter(s => s !== null) as string[];
        }
        // Looking for tags the requesting user has hidden
        else if (userId && input.hidden) {
            idsLimit = (await prisma.user_tag_hidden.findMany({
                where: { userId }
            })).map(s => s.tagId).filter(s => s !== null) as string[]
        }
        return await readManyHelper(userId, { ...input, ids: idsLimit }, info, prisma, tagSearcher())
    },
})

const profileMutater = (validater: any, prisma: PrismaType) => ({
    async updateProfile(
        userId: string,
        input: ProfileUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Profile>> {
        // Check for valid arguments
        if (!input.username || input.username.length < 1) throw new CustomError(CODE.InternalError, 'Name too short');
        // Check for censored words
        if (hasProfanity(input.username, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create user data
        let userData: { [x: string]: any } = {
            username: input.username,
            bio: input.bio,
            theme: input.theme,
            // Handle tags TODO probably doesn't work
            stars: await TagModel(prisma).relationshipBuilder(userId, {
                tagsCreate: input.starredTagsCreate,
                tagsConnect: input.starredTagsConnect,
                tagsDisconnect: input.starredTagsDisconnect,
            }, true),
            hiddenTags: await TagModel(prisma).relationshipBuilder(userId, {
                tagsCreate: input.hiddenTagsCreate,
                tagsConnect: input.hiddenTagsConnect,
                tagsDisconnect: input.hiddenTagsDisconnect,
            }, true),
        };
        // Update user
        const user = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper(info)
        });
        return modelToGraphQL(user, info);
    },
    async updateEmails(
        userId: string,
        input: ProfileEmailUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Profile>> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!validater.validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
        // Create user data
        let userData: { [x: string]: any } = {
            password: input.newPassword ? validater.hashPassword(input.newPassword) : undefined,
            emails: await EmailModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Send verification emails
        if (Array.isArray(input.emailsCreate)) {
            for (const email of input.emailsCreate) {
                await validater.setupVerificationCode(email.emailAddress);
            }
        }
        // Update user
        user = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper(info)
        });
        return modelToGraphQL(user, info);
    },
    async deleteProfile(userId: string, input: UserDeleteInput): Promise<Success> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!validater.validatePassword(input.password, user)) throw new CustomError(CODE.BadCredentials);
        await prisma.user.delete({
            where: { id: userId }
        })
        return { success: true };
    },
})


export function ProfileModel(prisma: PrismaType) {
    const prismaObject = prisma.user;
    const format = profileFormatter();
    const validate = profileValidater();
    const port = porter(prisma);
    const mutate = profileMutater(validate, prisma);
    const query = profileQuerier(prisma);

    return {
        prismaObject,
        ...format,
        ...validate,
        ...port,
        ...mutate,
        ...query,
    }
}