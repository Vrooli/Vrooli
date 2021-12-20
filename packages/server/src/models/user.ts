import { Session, User, Role, Comment, Email, Wallet, Resource, Project, Organization, Routine, Standard, Tag } from "schema/types";
import { addJoinTables, BaseState, deleter, findByIder, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, selectHelper } from "./base";
import { onlyPrimitives } from "../utils/objectTools";
import { CustomError } from "../error";
import { CODE } from '@local/shared';
import bcrypt from 'bcrypt';
import pkg from '@prisma/client';
import { sendResetPasswordLink, sendVerificationLink } from "../worker/email/queue";
import { GraphQLResolveInfo } from "graphql";
import { PrismaType, RecursivePartial } from "../types";
const { AccountStatus } = pkg;

const CODE_TIMEOUT = 2 * 24 * 3600 * 1000;
const HASHING_ROUNDS = 8;
const LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT = 3;
const LOGIN_ATTEMPTS_TO_HARD_LOCKOUT = 10;
const SOFT_LOCKOUT_DURATION = 15 * 60 * 1000;

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type UserRelationshipList = 'comments' | 'roles' | 'emails' | 'wallets' | 'resources' |
    'projects' | 'starredComments' | 'starredProjects' | 'starredOrganizations' |
    'starredResources' | 'starredStandards' | 'starredTags' | 'starredUsers' |
    'sentReports' | 'reports' | 'votedComments' | 'votedByTag';
// Type 2. QueryablePrimitives
export type UserQueryablePrimitives = Omit<User, UserRelationshipList>;
// Type 3. AllPrimitives
export type UserAllPrimitives = UserQueryablePrimitives & {
    password: string,
    logInAttempts: number,
    lastLoginAttempt: Date,
};
// type 4. FullModel
export type UserFullModel = UserAllPrimitives &
    Pick<User, 'comments' | 'emails' | 'wallets' | 'sentReports' | 'reports'> &
{
    roles: { role: Role[] }[],
    resources: { resource: Resource[] }[],
    projects: { project: Project[] }[],
    starredComments: { starred: Comment[] }[],
    starredProjects: { starred: Project[] }[],
    starredOrganizations: { starred: Organization[] }[],
    starredResources: { starred: Resource[] }[],
    starredRoutines: { starred: Routine[] }[],
    starredStandards: { starred: Standard[] }[],
    starredTags: { starred: Tag[] }[],
    starredUsers: { starred: User[] }[],
    votedComments: { voted: Comment[] }[],
    votedByTag: { tag: Tag[] }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<User, UserFullModel> => {
    const joinMapper = {
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
    return {
        toDB: (obj: RecursivePartial<User>): RecursivePartial<UserFullModel> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<UserFullModel>): RecursivePartial<User> => removeJoinTables(obj, joinMapper)
    }
}

/**
 * Custom component for email/password validation
 * @param state 
 * @returns 
 */
const validater = (state: any) => ({
    /**
     * Creates session object from user
     */
    toSession(user: RecursivePartial<User>): RecursivePartial<Session> {
        return {
            id: user.id ?? '',
            theme: user.theme ?? 'light',
            roles: user.roles ?? []
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
    validateCode(providedCode: string, storedCode: string, dateRequested: Date): boolean {
        return Boolean(providedCode) && Boolean(storedCode) && Boolean(dateRequested) &&
            providedCode === storedCode && Date.now() - new Date(dateRequested).getTime() < CODE_TIMEOUT;
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
            [AccountStatus.DELETED]: CODE.NoUser,
            [AccountStatus.SOFT_LOCKED]: CODE.SoftLockout,
            [AccountStatus.HARD_LOCKED]: CODE.HardLockout
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
        const unable_to_reset = [AccountStatus.HARD_LOCKED, AccountStatus.DELETED];
        // If account is not deleted or hard-locked, and lockout duration has passed
        if (!unable_to_reset.includes(user.status) && Date.now() - new Date(user.lastLoginAttempt).getTime() > SOFT_LOCKOUT_DURATION) {
            console.log('returning with reset log in');
            return await state.prisma.user.update({
                where: { id: user.id },
                data: { logInAttempts: 0 },
                select: {
                    theme: true,
                    roles: { select: { role: { select: { title: true } } } }
                }
            });
        }
        // If password is valid
        if (this.validatePassword(password, user)) {
            return await state.prisma.user.update({
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
            })
        }
        // If password is invalid
        let new_status: any = AccountStatus.UNLOCKED;
        let log_in_attempts = user.logInAttempts++;
        if (log_in_attempts > LOGIN_ATTEMPTS_TO_HARD_LOCKOUT) {
            new_status = AccountStatus.HARD_LOCKED;
        } else if (log_in_attempts > LOGIN_ATTEMPTS_TO_SOFT_LOCKOUT) {
            new_status = AccountStatus.SOFT_LOCKED;
        }
        await state.prisma.user.update({
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
        const updatedUser = await state.prisma.user.update({
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
    * Updated user object with new account verification code, and sends email to user with link
    * @param user User object
    */
    async setupVerificationCode(user: any): Promise<void> {
        // Generate new code
        const verificationCode = this.generateCode();
        // Store code and request time in user row
        const updatedUser = await state.prisma.user.update({
            where: { id: user.id },
            data: { verificationCode, lastResetPasswordReqestAttempt: new Date().toISOString() },
            select: { emails: { select: { emailAddress: true } } }
        })
        // Send new verification emails
        for (const email of updatedUser.emails) {
            sendVerificationLink(email.emailAddress, user.id, verificationCode);
        }
    },
    /**
     * Validate verification code and update user's account status
     * @param user User object
     * @param code Verification code
     * @returns Updated user object
     */
    async validateVerificationCode(user: any, code: string): Promise<any> {
        // If email already verified, remove old verification code
        if (user.emailVerified) {
            user = await state.prisma.user.update({
                where: { id: user.id },
                data: { verificationCode: null }
            })
        }
        // Otherwise, validate code
        else {
            // If code is correct and not expired
            if (this.validateCode(code, user.verificationCode, user.lastVerificationCodeRequestAttempt)) {
                user = await state.prisma.user.update({
                    where: { id: user.id },
                    data: { status: AccountStatus.UNLOCKED, emailVerified: true, verificationCode: null, lastVerificationCodeRequestAttempt: null }
                })
            }
            // If code is incorrect or expired, create new code and send email
            else {
                user = await this.setupVerificationCode(user);
            }
        }
        return user;
    }
})

/**
 * Customer component for finding users by email
 * @param state 
 * @returns 
 */
const findByEmailer = (state: any) => ({
    /**
     * Find a user by email address
     * @param email The user's email address
     * @returns A user object without relationships
     */
    async findByEmail(email: string): Promise<UserAllPrimitives> {
        if (!email) throw new CustomError(CODE.BadCredentials);
        // Validate email address
        const emailRow = await state.prisma.email.findUnique({ where: { emailAddress: email } });
        if (!emailRow) throw new CustomError(CODE.BadCredentials);
        // Find user
        let user = await state.prisma.user.findUnique({ where: { id: emailRow.userId } });
        if (!user) throw new CustomError(CODE.ErrorUnknown);
        return user;
    }
})

/**
 * Custom upserter component for upserting resources
 * @param state 
 * @returns 
 */
const upserter = ({ prisma, format }: BaseState<User>) => ({
    async upsertUser(data: any, info: GraphQLResolveInfo | null = null): Promise<RecursivePartial<User>> {
        // Check arguments
        if (!prisma) throw new CustomError(CODE.ErrorUnknown);
        // Remove relationship data, as they are handled on a case-by-case basis
        let cleanedData = onlyPrimitives(data);
        // Upsert user
        let user;
        if (!data.id) {
            // Check for valid username
            //TODO
            // Make sure username isn't in use
            if (await prisma.user.findUnique({ where: { username: data.username } })) throw new CustomError(CODE.UsernameInUse);
            user = await prisma.user.create({ data: cleanedData })
        } else {
            user = await prisma.user.update({
                where: { id: data.id },
                data: cleanedData
            })
        }
        // Upsert emails
        for (const email of (data.emails ?? [])) {
            const emailExists = await prisma.email.findUnique({ where: { emailAddress: email.emailAddress } });
            if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
            if (!email.id) {
                await prisma.email.create({ data: { ...email, id: undefined, user: user.id } })
            } else {
                await prisma.email.update({
                    where: { id: email.id },
                    data: email
                })
            }
        }
        // Upsert roles
        for (const role of (data.roles ?? [])) {
            if (!role.id) continue;
            const roleData = { userId: user.id, roleId: role.id };
            await prisma.user_roles.upsert({
                where: { user_roles_userid_roleid_unique: roleData },
                create: roleData,
                update: roleData
            })
        }
        // Create selector
        const select = selectHelper(info);
        // Query database
        user = await prisma.user.findUnique({ where: { id: user.id }, ...select }) as RecursivePartial<UserFullModel>;
        // Return formatted user
        return format.toGraphQL(user);
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function UserModel(prisma?: PrismaType) {
    let obj: BaseState<User> = {
        prisma,
        model: MODEL_TYPES.User,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<User>(obj),
        ...findByEmailer(obj),
        ...formatter(),
        ...upserter(obj),
        ...deleter(obj),
        ...reporter(),
        ...validater(obj),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================