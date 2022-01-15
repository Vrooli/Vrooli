import { Session, User, Role, Comment, Resource, Project, Organization, Routine, Standard, Tag, Success, Profile, UserSortBy, UserSearchInput, UserCountInput, Email } from "../schema/types";
import { addCountQueries, addJoinTables, counter, deleter, findByIder, FormatConverter, getRelationshipData, InfoType, JoinMap, keepOnly, MODEL_TYPES, PaginatedSearchResult, removeCountQueries, removeJoinTables, reporter, searcher, selectHelper, Sortable } from "./base";
import { CustomError } from "../error";
import { AccountStatus, CODE } from '@local/shared';
import bcrypt from 'bcrypt';
import { sendResetPasswordLink, sendVerificationLink } from "../worker/email/queue";
import { GraphQLResolveInfo } from "graphql";
import { PrismaType, RecursivePartial } from "../types";
import { EmailAllPrimitives } from "./email";

const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;
const EXPORT_LIMIT = 3;
const EXPORT_LIMIT_TIMEOUT = 24 * 3600 * 1000;

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type UserRelationshipList = 'comments' | 'roles' | 'emails' | 'wallets' | 'resources' |
    'donationResources' | 'projects' | 'starredBy' | 'starredComments' | 'starredProjects' | 'starredOrganizations' |
    'starredResources' | 'starredRoutines' | 'starredStandards' | 'starredTags' | 'starredUsers' |
    'sentReports' | 'reports' | 'votedComments' | 'votedByTag';
// Type 2. QueryablePrimitives
// 2.1 Profile primitives (i.e. your own account)
export type UserProfileQueryablePrimitives = Omit<Profile, UserRelationshipList | 'status'> & {
    status: AccountStatus;
}
// 2.2 User primitives (i.e. other users)
export type UserQueryablePrimitives = Omit<User, UserRelationshipList>;
// Type 3. AllPrimitives
export type UserAllPrimitives = UserProfileQueryablePrimitives & {
    password: string | null,
    logInAttempts: number,
    lastLoginAttempt: Date,
    numExports: number,
    lastExport: Date | null,
};
// type 4. Database shape
export type UserDB = UserAllPrimitives &
    Pick<Profile, 'comments' | 'emails' | 'wallets' | 'sentReports' | 'reports'> &
{
    roles: { role: Role }[],
    resources: { resource: Resource }[],
    donationResources: { resource: Resource }[],
    projects: { project: Project }[],
    starredComments: { starred: Comment }[],
    starredProjects: { starred: Project }[],
    starredOrganizations: { starred: Organization }[],
    starredResources: { starred: Resource }[],
    starredRoutines: { starred: Routine }[],
    starredStandards: { starred: Standard }[],
    starredTags: { starred: Tag }[],
    starredUsers: { starred: User }[],
    votedComments: { voted: Comment }[],
    votedByTag: { tag: Tag }[],
    _count: { starredBy: number }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Describes shape of component that converts between Prisma and GraphQL user object types.
 */
export type UserFormatConverter = {
    joinMapper?: JoinMap;
    toDBProfile: (obj: RecursivePartial<Profile>) => RecursivePartial<UserDB>;
    toDBUser: (obj: RecursivePartial<User>) => RecursivePartial<UserDB>;
    toGraphQLProfile: (obj: RecursivePartial<UserDB>) => RecursivePartial<Profile>;
    toGraphQLUser: (obj: RecursivePartial<UserDB>) => RecursivePartial<User>;
}

/**
 * Component for formatting between graphql and prisma types
 * Users are unique in that they have multiple GraphQL views (your own profile vs. other users)
 */
const formatter = (): UserFormatConverter => {
    const joinMapper = {
        donationResources: 'resource',
        roles: 'role',
        resources: 'resource',
        projects: 'project',
        starredComments: 'starred',
        starredProjects: 'starred',
        starredOrganizations: 'starred',
        starredResources: 'starred',
        starredRoutines: 'starred',
        starredStandards: 'starred',
        starredTags: 'starred',
        starredUsers: 'starred',
        votedComments: 'voted',
        votedByTag: 'tag',
    };
    const countMapper = {
        stars: 'starredBy',
    }
    return {
        toDBProfile: (obj: RecursivePartial<Profile>): RecursivePartial<UserDB> => addJoinTables(obj, joinMapper),
        toDBUser: (obj: RecursivePartial<User>): RecursivePartial<UserDB> => {
            let modified = addJoinTables(obj, joinMapper);
            modified = addCountQueries(modified, countMapper);
            return modified;
        },
        toGraphQLProfile: (obj: RecursivePartial<UserDB>): RecursivePartial<Profile> => removeJoinTables(obj, joinMapper),
        toGraphQLUser: (obj: RecursivePartial<UserDB>): RecursivePartial<User> => {
            let modified = removeJoinTables(obj, joinMapper);
            modified = removeCountQueries(modified, countMapper);
            return modified;
        },
    }
}

/**
 * Component for search filters
 */
const sorter = (): Sortable<UserSortBy> => ({
    defaultSort: UserSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [UserSortBy.AlphabeticalAsc]: { username: 'asc' },
            [UserSortBy.AlphabeticalDesc]: { username: 'desc' },
            [UserSortBy.CommentsAsc]: { comments: { _count: 'asc' } },
            [UserSortBy.CommentsDesc]: { comments: { _count: 'desc' } },
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
 * Custom component for email/password validation
 * @param state 
 * @returns 
 */
const validater = (prisma?: PrismaType) => ({
    /**
     * Creates session object from user
     */
    toSession(user: RecursivePartial<UserDB>): RecursivePartial<Session> {
        return {
            id: user.id ?? '',
            theme: user.theme ?? 'light',
            roles: user.roles ? user.roles.map(r => (r?.role as RecursivePartial<Role>)?.title ?? undefined) : []
        }
    },
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
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // First, check if the log in fail counter should be reset
        const unable_to_reset = [AccountStatus.HardLocked, AccountStatus.Deleted];
        // If account is not deleted or hard-locked, and lockout duration has passed
        if (!unable_to_reset.includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
            console.log('returning with reset log in');
            return await prisma.user.update({
                where: { id: user.id },
                data: { logInAttempts: 0 },
                select: {
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            }) as any;
        }
        // If password is valid
        if (this.validatePassword(password, user)) {
            return await prisma.user.update({
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
            }) as any;
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
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
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
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Generate new code
        const verificationCode = this.generateCode();
        // Store code and request time in email row
        const email = await prisma.email.update({
            where: { emailAddress },
            data: { verificationCode, lastVerificationCodeRequestAttempt: new Date().toISOString() },
            select: { userId: true}
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
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
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
 * Customer component for finding users by email
 * @param state 
 * @returns 
 */
const findByEmailer = (prisma?: PrismaType) => ({
    /**
     * Find a user by email address
     * @param email The user's email address
     * @returns A user object without relationships
     */
    async findByEmail(emailAddress: string): Promise<{user: UserAllPrimitives, email: EmailAllPrimitives}> {
        // Check arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!emailAddress) throw new CustomError(CODE.BadCredentials);
        // Validate email address
        const email = await prisma.email.findUnique({ where: { emailAddress } });
        if (!email) throw new CustomError(CODE.BadCredentials);
        // Find user
        let user = await prisma.user.findUnique({ where: { id: email.userId ?? '' } });
        if (!user) throw new CustomError(CODE.ErrorUnknown);
        return { user, email }
    }
})

/**
 * Custom component for creating users. 
 * NOTE: Data should be in Prisma shape, not GraphQL
 */
 const userCreater = (toDB: FormatConverter<User, UserDB>['toDB'], prisma?: PrismaType) => ({
    async create(
        data: any, 
        info: GraphQLResolveInfo | null = null,
    ): Promise<RecursivePartial<UserDB> | null> {
        // Check for valid arguments
        if (!prisma || !data.username) throw new CustomError(CODE.InvalidArgs);
        // Remove any relationships should not be created/connected in this operation
        data = keepOnly(data, ['roles', 'emails', 'resources', 'donationResources', 'hiddenTags', 'starredTags']);
        // Perform additional checks
        // Check for valid emails
        for (const email of getRelationshipData(data, 'emails')) {
            if (!email) continue;
            const emailExists = await prisma.email.findUnique({ where: { emailAddress: email.emailAddress } });
            if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
        }
        console.log('email boop', getRelationshipData(data, 'emails'))
        // Make sure roles have valid ids
        for (const role of getRelationshipData(data, 'roles')) {
            if (!role?.role) continue;
            if (!role.role.id) throw new CustomError(CODE.InvalidArgs);
            const roleExists = await prisma.role.findUnique({ where: { id: role.role.id } });
            if (!roleExists) throw new CustomError(CODE.InvalidArgs);
        }
        console.log('role boop', getRelationshipData(data, 'roles'))
        // Create
        const { id } = await prisma.user.create({ data });
        // Query database
        return await prisma.user.findUnique({ where: { id }, ...selectHelper<User, UserDB>(info, toDB) }) as RecursivePartial<UserDB> | null;
    }
})

// /**
//  * Custom component for upserting users.
//  * Contains an overall upsert, as well as one for each relationship
//  */
// const userUpserter = (toDB: FormatConverter<User, UserDB>['toDB'], prisma?: PrismaType) => ({
//     async upsert(data: UpsertUserInput, info: GraphQLResolveInfo | null = null): Promise<RecursivePartial<UserDB> | null> {
//         // Check for valid arguments
//         if (!prisma || !data.username) throw new CustomError(CODE.InvalidArgs);
//         // Remove relationship data, as they are handled on a case-by-case basis
//         let cleanedData = onlyPrimitives(data);


//         const user = await prisma.user.upsert({
//             where: { id: cleanedData.id },
//         })


//         // Upsert user
//         let user;
//         if (!data.id) {
//             // Check for valid username
//             //TODO
//             // Make sure username isn't in use
//             if (await prisma.user.findUnique({ where: { username: data.username } })) throw new CustomError(CODE.UsernameInUse);
//             user = await prisma.user.create({ data: cleanedData })
//         } else {
//             user = await prisma.user.update({
//                 where: { id: data.id },
//                 data: cleanedData
//             })
//         }
//         // Upsert emails
//         for (const email of (data.emails ?? [])) {
//             // Fetch all existing 
//             // Check if child is allowed to be added
//             if (!email.id) {
//                 const emailExists = await prisma.email.findUnique({ where: { emailAddress: email.emailAddress } });
//                 if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
//             }
//             // Add child
//             await upsertOneToManyRelationship(MODEL_TYPES.Email, 'userId', prisma, email, user.id);
//         }
//         // Upsert roles
//         for (const role of (data.roles ?? [])) {
//             if (!role.id) continue;
//             const joinData = { userId: user.id, roleId: role.id };
//             await upsertManyToManyRelationship('user_roles', 'user_roles_userid_roleid_unique', 'userId', 'roleId', prisma, joinData);
//         }
//         // Query database
//         return await prisma.user.findUnique({ where: { id: user.id }, ...selectHelper<User, UserDB>(info, toDB) }) as RecursivePartial<UserDB> | null;
//     }
// })

/**
 * Custom component for importing/exporting data from Vrooli
 * @param state 
 * @returns 
 */
const porter = (prisma?: PrismaType) => ({
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
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Find user
        const user = await prisma.user.findUnique({ where: { id }, select: { numExports: true, lastExport: true } });
        if (!user) throw new CustomError(CODE.ErrorUnknown);
        // Check if export is allowed TODO export reset and whatnot
        if (user.numExports >= EXPORT_LIMIT) throw new CustomError(CODE.ExportLimitReached);
        throw new CustomError(CODE.NotImplemented)
    },
})

/**
 * Component for searching
 */
 export const userSearcher = (
    toDB: FormatConverter<User, UserDB>['toDB'],
    toGraphQL: FormatConverter<User, UserDB>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    async search(where: { [x: string]: any }, input: UserSearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        // Many-to-many search queries
        const organizationIdQuery = input.organizationId ? { organizations: { some: { organizationId: input.organizationId } } } : {};
        const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
        const routineIdQuery = input.routineId ? { routines: { some: { routineId: input.routineId } } } : {};
        // One-to-many search queries
        const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
        const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
        const search = searcher<UserSortBy, UserSearchInput, User, UserDB>(MODEL_TYPES.User, toDB, toGraphQL, sorter, prisma);
        return search.search({...organizationIdQuery, ...projectIdQuery, ...routineIdQuery, ...reportIdQuery, ...standardIdQuery, ...where}, input, info);
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function UserModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.User;
    const format = formatter();
    const sort = sorter();

    return {
        prisma,
        model,
        ...format,
        ...sort,
        ...counter<UserCountInput>(model, prisma),
        ...deleter(model, prisma),
        ...findByEmailer(prisma),
        ...findByIder<User, UserDB>(model, format.toDBUser, prisma),
        ...porter(prisma),
        ...reporter(),
        ...userCreater(format.toDBUser, prisma),
        ...userSearcher(format.toDBUser, format.toGraphQLUser, sort, prisma),
        ...validater(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================