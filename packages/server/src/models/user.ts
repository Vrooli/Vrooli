import { Session, User, Success, Profile, UserSortBy, UserSearchInput, UserCountInput, UserDeleteInput, FindByIdInput, ProfileUpdateInput } from "../schema/types";
import { addJoinTables, counter, InfoType, JoinMap, MODEL_TYPES, PaginatedSearchResult, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { AccountStatus, CODE, ROLES } from '@local/shared';
import bcrypt from 'bcrypt';
import { sendResetPasswordLink, sendVerificationLink } from "../worker/email/queue";
import { PrismaType, RecursivePartial } from "../types";
import { hasProfanity } from "../utils/censor";
import { user } from "@prisma/client";

const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
const EXPORT_LIMIT = 3;
const EXPORT_LIMIT_TIMEOUT = 24 * 3600 * 1000;

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Describes shape of component that converts between Prisma and GraphQL user object types.
 */
export type UserFormatConverter = {
    joinMapper?: JoinMap;
    toDBProfile: (obj: RecursivePartial<Profile>) => RecursivePartial<user>;
    toDBUser: (obj: RecursivePartial<User>) => RecursivePartial<user>;
    toGraphQLProfile: (obj: RecursivePartial<user>) => RecursivePartial<Profile>;
    toGraphQLUser: (obj: RecursivePartial<user>) => RecursivePartial<User>;
}

/**
 * Component for formatting between graphql and prisma types
 * Users are unique in that they have multiple GraphQL views (your own profile vs. other users)
 */
export const userFormatter = (): UserFormatConverter => {
    const joinMapper = {
        roles: 'role',
    };
    return {
        toDBProfile: (obj: RecursivePartial<Profile>): RecursivePartial<user> => addJoinTables(obj, joinMapper),
        toDBUser: (obj: RecursivePartial<User>): RecursivePartial<user> => {
            let modified = addJoinTables(obj, joinMapper);
            // Remove isStarred, as it is calculated in its own query
            if (modified.isStarred) delete modified.isStarred;
            return modified;
        },
        toGraphQLProfile: (obj: RecursivePartial<user>): RecursivePartial<Profile> => removeJoinTables(obj, joinMapper),
        toGraphQLUser: (obj: RecursivePartial<user>): RecursivePartial<User> => {
            let modified = removeJoinTables(obj, joinMapper);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
export const userSorter = (): Sortable<UserSortBy> => ({
    defaultSort: UserSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [UserSortBy.AlphabeticalAsc]: { username: 'asc' },
            [UserSortBy.AlphabeticalDesc]: { username: 'desc' },
            [UserSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [UserSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [UserSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [UserSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [UserSortBy.StarsAsc]: { starredBy: { _count: 'asc' } },
            [UserSortBy.StarsDesc]: { starredBy: { _count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { username: { ...insensitive } },
            ]
        })
    }
})

/**
 * Component for creating sessions
 */
export const userSessioner = () => ({
    /**
     * Creates session object from user
     */
    toSession(user: RecursivePartial<user>): Session {
        console.log('user toSession', user);
        return {
            id: user.id ?? '',
            theme: user.theme ?? 'light',
            roles: [ROLES.Actor],
        }
    }
})

/**
 * Custom component for email/password validation
 * @param state 
 * @returns 
 */
const validater = (prisma: PrismaType) => ({
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
    async logIn(password: string, user: any): Promise<Session | null> {
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
            return userSessioner().toSession(userData);
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
    async setupPasswordReset(user: any): Promise<boolean> {
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
    async setupVerificationCode(emailAddress: string): Promise<void> {
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
    async validateVerificationCode(emailAddress: string, userId: string, code: string): Promise<boolean> {
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
                await this.setupVerificationCode(emailAddress);
            }
        }
        return true;
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

/**
* Handles the authorized adding, searching, updating, and deleting of users.
*/
const userer = (format: UserFormatConverter, sort: Sortable<UserSortBy>, prisma: PrismaType) => ({
    async findUser(
        userId: string | null | undefined, // Of the user making the request, not the requested user
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<User> | null> {
        // Create selector. Make sure not to select private data
        const select = selectHelper<User, user>(info, format.toDBUser);
        // Access database
        let user = await prisma.user.findUnique({ where: { id: input.id }, ...select });
        // Return user with "isStarred" field
        // If the user is querying themselves, 
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!userId || userId === user.id) return { ...format.toGraphQLUser(user), isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, userId: user.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.toGraphQLUser(user), isStarred };
    },
    async findProfile(
        userId: string,
        info: InfoType = null,
    ): Promise<RecursivePartial<Profile> | null> {
        // Create selector. Make sure not to select private data
        const select = selectHelper<Profile, user>(info, format.toDBProfile);
        // Access database
        const user = await prisma.user.findUnique({ where: { id: userId }, ...select });
        return user ? format.toGraphQLProfile(user) : null;
    },
    async searchUsers(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: UserSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        // Search
        const search = searcher<UserSortBy, UserSearchInput, User, user>(MODEL_TYPES.User, format.toDBUser, format.toGraphQLUser, sort, prisma);
        let searchResults = await search.search({ ...organizationIdQuery, ...projectIdQuery, ...routineIdQuery, ...reportIdQuery, ...standardIdQuery, ...where }, input, info);
        // Compute "isStarred" field for each user
        // If userId not provided, then "isStarred" is false
        if (!userId) {
            searchResults.edges = searchResults.edges.map(({ cursor, node }) => ({ cursor, node: { ...node, isStarred: false } }));
            return searchResults;
        }
        // Otherwise, query votes for all search results in one query
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isStarredArray = await prisma.star.findMany({ where: { byId: userId, userId: { in: resultIds } } });
        searchResults.edges = searchResults.edges.map(({ cursor, node }) => {
            const isStarred = Boolean(isStarredArray.find(({ userId }) => userId === node.id));
            return { cursor, node: { ...node, isStarred } };
        });
        return searchResults;
    },
    async updateProfile(
        userId: string,
        input: ProfileUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Profile>> {
        // Check for valid arguments
        if (!input.username || input.username.length < 1) throw new CustomError(CODE.InternalError, 'Name too short');
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!UserModel(prisma).validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
        // Check for censored words
        if (hasProfanity(input.username, input.bio)) throw new CustomError(CODE.BannedWord);
        // Create user data
        let userData: { [x: string]: any } = { username: input.username, bio: input.bio, theme: input.theme };
        // TODO emails
        // Update user
        user = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper<Profile, user>(info, format.toDBProfile)
        });
        // Return user with "isStarred" field. This will return false, since the user cannot star themselves
        return format.toGraphQLProfile(user)
    },
    async deleteProfile(userId: string, input: UserDeleteInput): Promise<Success> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!UserModel(prisma).validatePassword(input.password, user)) throw new CustomError(CODE.BadCredentials);
        await prisma.user.delete({
            where: { id: userId }
        })
        return { success: true };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function UserModel(prisma: PrismaType) {
    const model = MODEL_TYPES.User;
    const format = userFormatter();
    const sort = userSorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<UserCountInput>(model, prisma),
        ...porter(prisma),
        ...userSessioner(),
        ...userer(format, sort, prisma),
        ...validater(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================