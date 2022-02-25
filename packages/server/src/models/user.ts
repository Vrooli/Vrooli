import { Session, User, Success, Profile, UserSortBy, UserSearchInput, UserCountInput, UserDeleteInput, FindByIdInput, ProfileUpdateInput, Tag, ProfileEmailUpdateInput } from "../schema/types";
import { addJoinTables, counter, FormatterMap, infoToPartialSelect, InfoType, JoinMap, MODEL_TYPES, PaginatedSearchResult, relationshipFormatter, removeJoinTables, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { AccountStatus, CODE, ROLES } from '@local/shared';
import bcrypt from 'bcrypt';
import { sendResetPasswordLink, sendVerificationLink } from "../worker/email/queue";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "../types";
import { hasProfanity } from "../utils/censor";
import { user } from "@prisma/client";
import { TagModel } from "./tag";
import { EmailModel } from "./email";
import _ from "lodash";

export const userDBFields = ['id', 'created_at', 'updated_at', 'bio', 'username', 'theme', 'numExports', 'lastExport', 'status', 'stars'];

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
    selectToDBProfile: (obj: RecursivePartial<Profile>) => RecursivePartial<user>;
    dbShapeUser: (obj: PartialSelectConvert<User>) => PartialSelectConvert<user>;
    dbPruneUser: (obj: PartialSelectConvert<User>) => PartialSelectConvert<user>;
    selectToDBUser: (obj: PartialSelectConvert<User>) => PartialSelectConvert<user>;
    selectToGraphQLProfile: (obj: RecursivePartial<user>) => RecursivePartial<Profile>;
    selectToGraphQLUser: (obj: RecursivePartial<user>) => RecursivePartial<User>;
}

/**
 * Component for formatting between graphql and prisma types
 * Users are unique in that they have multiple GraphQL views (your own profile vs. other users)
 */
export const userFormatter = (): UserFormatConverter => {
    const joinMapper = {
        hiddenTags: 'tag',
        roles: 'role',
        starredBy: 'user',
    };
    return {
        selectToDBProfile: (partial: RecursivePartial<Profile>): RecursivePartial<user> => {
            let modified = partial;
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.selectToDB],
                ['hiddenTags', FormatterMap.Tag.selectToDB],
                ['projects', FormatterMap.Project.selectToDB],
                ['projectsCreated', FormatterMap.Project.selectToDB],
                ['reports', FormatterMap.Report.selectToDB],
                ['resources', FormatterMap.Resource.selectToDB],
                ['routines', FormatterMap.Routine.selectToDB],
                ['routinesCreated', FormatterMap.Routine.selectToDB],
                ['sentReports', FormatterMap.Report.selectToDB],
                ['starred', FormatterMap.Star.selectToDB],
                ['starredBy', FormatterMap.User.selectToDBUser],
                ['tags', FormatterMap.Tag.selectToDB],
                ['votes', FormatterMap.Vote.selectToDB],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            // Remove calculated fields
            let { starredTags, hiddenTags, ...rest } = modified;
            return rest as any;
        },
        dbShapeUser: (partial: PartialSelectConvert<User>): PartialSelectConvert<user> => {
            // Convert relationships
            let modified = relationshipFormatter(partial, [
                ['comments', FormatterMap.Comment.dbShape],
                ['projects', FormatterMap.Project.dbShape],
                ['projectsCreated', FormatterMap.Project.dbShape],
                ['reports', FormatterMap.Report.dbShape],
                ['resources', FormatterMap.Resource.dbShape],
                ['routines', FormatterMap.Routine.dbShape],
                ['routinesCreated', FormatterMap.Routine.dbShape],
                ['starredBy', FormatterMap.User.dbShapeUser],
                ['tags', FormatterMap.Tag.dbShape],
            ]);
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPruneUser: (info: InfoType): PartialSelectConvert<user> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Remove calculated fields
            let { isStarred, ...rest } = modified;
            modified = rest;
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['comments', FormatterMap.Comment.dbPrune],
                ['projects', FormatterMap.Project.dbPrune],
                ['projectsCreated', FormatterMap.Project.dbPrune],
                ['reports', FormatterMap.Report.dbPrune],
                ['resources', FormatterMap.Resource.dbPrune],
                ['routines', FormatterMap.Routine.dbPrune],
                ['routinesCreated', FormatterMap.Routine.dbPrune],
                ['starredBy', FormatterMap.User.dbPruneUser],
                ['tags', FormatterMap.Tag.dbPrune],
            ]);
            return modified;
        },
        selectToDBUser: (info: InfoType): PartialSelectConvert<user> => {
            return userFormatter().dbShapeUser(userFormatter().dbPruneUser(info));
        },
        selectToGraphQLProfile: (obj: RecursivePartial<user>): RecursivePartial<Profile> => {
            if (!_.isObject(obj)) return obj;
            // Convert relationships
            let modified = relationshipFormatter(obj, [
                ['comments', FormatterMap.Comment.selectToGraphQL],
                ['hiddenTags', FormatterMap.Tag.selectToGraphQL],
                ['projects', FormatterMap.Project.selectToGraphQL],
                ['projectsCreated', FormatterMap.Project.selectToGraphQL],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['resources', FormatterMap.Resource.selectToGraphQL],
                ['routines', FormatterMap.Routine.selectToGraphQL],
                ['routinesCreated', FormatterMap.Routine.selectToGraphQL],
                ['sentReports', FormatterMap.Report.selectToGraphQL],
                ['starred', FormatterMap.Star.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['tags', FormatterMap.Tag.selectToGraphQL],
                ['votes', FormatterMap.Vote.selectToGraphQL],
            ]);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            modified = removeJoinTables(modified, joinMapper);
            return modified;
        },
        selectToGraphQLUser: (obj: RecursivePartial<user>): RecursivePartial<User> => {
            if (!_.isObject(obj)) return obj;
            // Convert relationships
            let modified = relationshipFormatter(obj, [
                ['comments', FormatterMap.Comment.selectToGraphQL],
                ['projects', FormatterMap.Project.selectToGraphQL],
                ['projectsCreated', FormatterMap.Project.selectToGraphQL],
                ['reports', FormatterMap.Report.selectToGraphQL],
                ['resources', FormatterMap.Resource.selectToGraphQL],
                ['routines', FormatterMap.Routine.selectToGraphQL],
                ['routinesCreated', FormatterMap.Routine.selectToGraphQL],
                ['starredBy', FormatterMap.User.selectToGraphQLUser],
                ['tags', FormatterMap.Tag.selectToGraphQL],
            ]);
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            modified = removeJoinTables(modified, joinMapper);
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
            [UserSortBy.StarsAsc]: { stars: 'asc' },
            [UserSortBy.StarsDesc]: { stars: 'desc' },
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
        info: InfoType,
    ): Promise<RecursivePartial<User> | null> {
        // Create selector. Make sure not to select private data
        const select = selectHelper(info, format.selectToDBUser);
        // Access database
        let user = await prisma.user.findUnique({ where: { id: input.id }, ...select });
        // Return user with "isStarred" field
        // If the user is querying themselves, 
        if (!user) throw new CustomError(CODE.InternalError, 'User not found.');
        if (!userId || userId === user.id) return { ...format.selectToGraphQLUser(user), isStarred: false };
        const star = await prisma.star.findFirst({ where: { byId: userId, userId: user.id } });
        const isStarred = Boolean(star) ?? false;
        return { ...format.selectToGraphQLUser(user), isStarred };
    },
    async findProfile(
        userId: string,
        info: InfoType,
    ): Promise<RecursivePartial<Profile> | null> {
        // Create selector. Make sure not to select private data
        const select = selectHelper(info, format.selectToDBProfile);
        // Access database
        const user = await prisma.user.findUnique({ where: { id: userId }, ...select });
        // Return user with "starredTags" and "hiddenTags" fields
        if (!user) throw new CustomError(CODE.InternalError, 'Error accessing profile.');
        const starredTags: Tag[] = await (await prisma.star.findMany({
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
        return { ...format.selectToGraphQLProfile(user), starredTags, hiddenTags };
    },
    async searchUsers(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: UserSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        // One-to-many search queries
        const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        // Search
        const search = searcher<UserSortBy, UserSearchInput, User, user>(MODEL_TYPES.User, format.selectToDBUser, sort, prisma);
        console.log('calling user search...')
        let searchResults = await search.search({ ...organizationIdQuery, ...projectIdQuery, ...routineIdQuery, ...reportIdQuery, ...standardIdQuery, ...where }, input, info);
        // Compute "isStarred" field for each user
        const resultIds = searchResults.edges.map(({ node }) => node.id).filter(id => Boolean(id));
        const isStarredArray = userId
            ? await prisma.star.findMany({ where: { byId: userId, userId: { in: resultIds } } })
            : Array(resultIds.length).fill(null);
        searchResults.edges = searchResults.edges.map(({ cursor, node }, i) => ({
            cursor, node: { ...node, isStarred: isStarredArray[i] }
        }));
        return searchResults;
    },
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
            // Handle tags
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
            ...selectHelper(info, format.selectToDBProfile)
        });
        return format.selectToGraphQLProfile(user)
    },
    async updateEmails(
        userId: string,
        input: ProfileEmailUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Profile>> {
        // Check for correct password
        let user = await prisma.user.findUnique({ where: { id: userId } });
        if (!user) throw new CustomError(CODE.InternalError, 'User not found');
        if (!UserModel(prisma).validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
        // Create user data
        let userData: { [x: string]: any } = {
            password: input.newPassword ? UserModel(prisma).hashPassword(input.newPassword) : undefined,
            emails: await EmailModel(prisma).relationshipBuilder(userId, input, true),
        };
        // Send verification emails
        if (Array.isArray(input.emailsCreate)) {
            for (const email of input.emailsCreate) {
                await UserModel(prisma).setupVerificationCode(email.emailAddress);
            }
        }
        // Update user
        user = await prisma.user.update({
            where: { id: userId },
            data: userData,
            ...selectHelper(info, format.selectToDBProfile)
        });
        return format.selectToGraphQLProfile(user)
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